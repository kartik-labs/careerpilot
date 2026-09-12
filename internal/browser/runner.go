package browser

import (
	"context"
	"fmt"
	"time"

	"github.com/kartik-labs/careerpilot/internal/application"
)

// Notifier delivers a human-facing message when a session needs
// attention. Kept minimal and swappable — Phase 5 provides a concrete
// implementation; this package only depends on the interface.
type Notifier interface {
	NotifyHumanRequired(ctx context.Context, sessionID, applicationID, reason, recoveryURL string) error
}

// ScreenshotStore persists screenshot bytes out-of-band and returns an
// opaque reference — Session/Checkpoint records only ever hold the
// reference, never the raw image bytes inline
// (docs/BROWSER-RECOVERY.md checkpoint fields: "screenshot_reference").
type ScreenshotStore interface {
	Save(ctx context.Context, sessionID string, shot Screenshot) (ref string, err error)
}

// Limits are the daily/per-session caps the runner must respect
// (CLAUDE.md "Initial Resource Limits").
type Limits struct {
	BrowserSessionsPerDay int
	MaxSessionMinutes     int
}

// Runner orchestrates one browser session against the application state
// machine. It never owns application business state — it only reports
// PageState observations and persists Session/Checkpoint records; all
// Application.Status transitions happen through
// internal/application.Transition, called by this package but decided by
// the caller's rules, not invented here.
type Runner struct {
	Driver      Driver
	Sessions    Store
	Tokens      RecoveryTokenStore
	Screens     ScreenshotStore
	Notifier    Notifier
	Limits      Limits
	NewID       func() string
	Now         func() time.Time
	RecoveryTTL time.Duration
}

// NewRunner builds a Runner.
func NewRunner(driver Driver, sessions Store, tokens RecoveryTokenStore, screens ScreenshotStore, notifier Notifier, limits Limits, newID func() string) *Runner {
	return &Runner{
		Driver:      driver,
		Sessions:    sessions,
		Tokens:      tokens,
		Screens:     screens,
		Notifier:    notifier,
		Limits:      limits,
		NewID:       newID,
		Now:         time.Now,
		RecoveryTTL: 30 * time.Minute,
	}
}

// StartInput carries what Start needs to begin a session for an
// application that has already passed the safety gate for READY status.
type StartInput struct {
	ApplicationID string
	Platform      string
	StartURL      string
	Fields        []FieldValue
}

// Result is what a Start/Resume call returns: the persisted Session and,
// if the session reached a terminal outcome, enough information for the
// caller to update Application status accordingly. The Runner itself
// never mutates an Application — it hands back this Result so the
// application package's own Transition function makes that call.
type Result struct {
	Session Session
	// Outcome summarizes what happened, for the caller to map onto
	// application.Status via application.Transition:
	//   "human_required", "submitted", "failed", "expired"
	Outcome string
}

// Start begins a new session: creates the Session record, launches the
// driver, navigates, fills deterministic fields, and detects state. It
// stops at the first sign of CAPTCHA/unknown state (-> HUMAN_REQUIRED) or
// error (-> FAILED); it never proceeds to Submit itself — that is a
// separate explicit call so the safety gate always runs immediately
// before submission, not earlier in the flow.
func (r *Runner) Start(ctx context.Context, in StartInput) (Result, error) {
	startOfDay := r.startOfToday()
	count, err := r.Sessions.CountStartedSince(startOfDay)
	if err != nil {
		return Result{}, fmt.Errorf("browser: check session limit: %w", err)
	}
	if count >= r.Limits.BrowserSessionsPerDay {
		return Result{}, fmt.Errorf("browser: daily browser session limit (%d) reached", r.Limits.BrowserSessionsPerDay)
	}

	now := r.Now()
	session := Session{
		ID:            r.NewID(),
		ApplicationID: in.ApplicationID,
		Platform:      in.Platform,
		State:         SessionStarting,
		StartedAt:     now,
		ExpiresAt:     now.Add(time.Duration(r.Limits.MaxSessionMinutes) * time.Minute),
	}

	if err := r.Driver.Launch(ctx); err != nil {
		session.State = SessionFailed
		session.FailureReason = err.Error()
		saved, saveErr := r.saveSession(session)
		if saveErr != nil {
			return Result{}, fmt.Errorf("browser: launch failed (%v) and save session also failed: %w", err, saveErr)
		}
		return Result{Session: saved, Outcome: "failed"}, fmt.Errorf("browser: launch: %w", err)
	}

	session.State = SessionRunning
	session, err = r.saveSession(session)
	if err != nil {
		return Result{}, err
	}

	return r.navigateAndFill(ctx, session, in.StartURL, in.Fields)
}

func (r *Runner) navigateAndFill(ctx context.Context, session Session, url string, fields []FieldValue) (Result, error) {
	if r.isExpired(session) {
		return r.expire(ctx, session)
	}

	state, err := r.Driver.Navigate(ctx, url)
	if err != nil {
		return r.fail(ctx, session, fmt.Sprintf("navigation failed: %v", err))
	}
	session.CurrentURL = url

	if handled, result, herr := r.handleNonNormalState(ctx, session, state); handled {
		return result, herr
	}

	return r.fillAndCheckpoint(ctx, session, fields)
}

// fillAndCheckpoint fills deterministic fields on the current page,
// re-detects state, and checkpoints on success. Shared by the initial
// navigation flow and Resume (which must not re-navigate).
func (r *Runner) fillAndCheckpoint(ctx context.Context, session Session, fields []FieldValue) (Result, error) {
	if err := r.Driver.Fill(ctx, fields); err != nil {
		return r.fail(ctx, session, fmt.Sprintf("fill failed: %v", err))
	}

	state, err := r.Driver.DetectState(ctx)
	if err != nil {
		return r.fail(ctx, session, fmt.Sprintf("state detection failed: %v", err))
	}
	if handled, result, herr := r.handleNonNormalState(ctx, session, state); handled {
		return result, herr
	}

	session, err = r.checkpoint(ctx, session, "form_section")
	if err != nil {
		return Result{}, err
	}

	return Result{Session: session, Outcome: "ready_for_review"}, nil
}

// handleNonNormalState routes CAPTCHA/unknown/login-wall states to the
// human handoff, and navigation errors to failure. Returns handled=false
// for StateNormal, meaning the caller should continue.
func (r *Runner) handleNonNormalState(ctx context.Context, session Session, state PageState) (bool, Result, error) {
	switch state {
	case StateNormal:
		return false, Result{}, nil
	case StateCaptcha:
		result, err := r.pauseForHuman(ctx, session, "CAPTCHA detected")
		return true, result, err
	case StateLoginWall, StateUnknown:
		result, err := r.pauseForHuman(ctx, session, fmt.Sprintf("unrecognized page state: %s", state))
		return true, result, err
	case StateNavigateErr:
		result, err := r.fail(ctx, session, "navigation error")
		return true, result, err
	default:
		result, err := r.pauseForHuman(ctx, session, fmt.Sprintf("unhandled page state: %s", state))
		return true, result, err
	}
}

// pauseForHuman implements the CAPTCHA/uncertain-state protocol exactly:
// save session, capture screenshot, persist current URL, mark
// HUMAN_REQUIRED, generate a recovery token, notify — and stop. It never
// attempts to solve or work around whatever triggered the pause
// (CLAUDE.md "CAPTCHA and anti-bot handling").
func (r *Runner) pauseForHuman(ctx context.Context, session Session, reason string) (Result, error) {
	shot, shotErr := r.Driver.Screenshot(ctx)
	var screenshotRef string
	if shotErr == nil {
		screenshotRef, _ = r.Screens.Save(ctx, session.ID, shot)
	}

	url, _ := r.Driver.CurrentURL(ctx)
	if url != "" {
		session.CurrentURL = url
	}

	session.State = SessionHumanReq
	session.HumanRequired = true
	session.FailureReason = reason
	session.ScreenshotRef = screenshotRef

	session, err := r.saveSession(session)
	if err != nil {
		return Result{}, err
	}

	token, err := NewRecoveryToken(session.ID, r.Now(), r.RecoveryTTL)
	if err != nil {
		return Result{}, fmt.Errorf("browser: generate recovery token: %w", err)
	}
	if err := r.Tokens.SaveTokenHash(token.Hash, session.ID, token.ExpiresAt); err != nil {
		return Result{}, fmt.Errorf("browser: save recovery token: %w", err)
	}

	recoveryURL := "/recovery/" + token.Raw
	if r.Notifier != nil {
		if err := r.Notifier.NotifyHumanRequired(ctx, session.ID, session.ApplicationID, reason, recoveryURL); err != nil {
			return Result{Session: session, Outcome: "human_required"}, fmt.Errorf("browser: notify human required: %w", err)
		}
	}

	return Result{Session: session, Outcome: "human_required"}, nil
}

// Resume continues a session after a human completed the required action
// (e.g. solved a CAPTCHA), validated via a recovery token. If the page has
// materially changed since the pause, it returns to HUMAN_REQUIRED rather
// than guessing (docs/BROWSER-RECOVERY.md "If the page has materially
// changed: HUMAN_REQUIRED. Do not guess.").
func (r *Runner) Resume(ctx context.Context, rawToken string, fields []FieldValue) (Result, error) {
	sessionID, err := Redeem(r.Tokens, rawToken, r.Now())
	if err != nil {
		return Result{}, fmt.Errorf("browser: resume: %w", err)
	}

	session, err := r.Sessions.GetSession(sessionID)
	if err != nil {
		return Result{}, fmt.Errorf("browser: resume: load session: %w", err)
	}

	if r.isExpired(session) {
		return r.expire(ctx, session)
	}

	state, err := r.Driver.DetectState(ctx)
	if err != nil {
		return r.fail(ctx, session, fmt.Sprintf("resume state detection failed: %v", err))
	}

	if state != StateNormal {
		result, err := r.pauseForHuman(ctx, session, fmt.Sprintf("page still not resumable: %s", state))
		return result, err
	}

	session.State = SessionResumed
	session, err = r.saveSession(session)
	if err != nil {
		return Result{}, err
	}

	// The page is already confirmed normal at session.CurrentURL — do not
	// re-navigate (that could reset in-progress form state); just resume
	// filling from where the human left off.
	return r.fillAndCheckpoint(ctx, session, fields)
}

// SubmitInput carries what Submit needs. GateResult must come from
// application.EvaluateSafetyGate — Submit refuses to proceed without a
// passed gate, full stop (CLAUDE.md "Application Safety Gate": "Only then
// may submission proceed.").
type SubmitInput struct {
	Session        Session
	GateResult     application.SafetyGateResult
	SubmitSelector string
}

// Submit performs the final submission action, but only if the safety
// gate passed. On an uncertain post-submit state, it does NOT retry —
// it surfaces HUMAN_REQUIRED so a human confirms the real outcome
// (CLAUDE.md "Never blindly retry SUBMITTING.").
func (r *Runner) Submit(ctx context.Context, in SubmitInput) (Result, error) {
	if !in.GateResult.Passed {
		return Result{}, fmt.Errorf("browser: refusing to submit: safety gate failed: %v", in.GateResult.Reasons)
	}

	session := in.Session
	if r.isExpired(session) {
		return r.expire(ctx, session)
	}

	session.State = SessionRunning
	session, err := r.saveSession(session)
	if err != nil {
		return Result{}, err
	}

	state, err := r.Driver.Submit(ctx, in.SubmitSelector)
	if err != nil {
		// An error mid-submission is uncertain-outcome, not a clean
		// failure — the browser action may or may not have gone through
		// server-side. Route to human review rather than retrying.
		result, herr := r.pauseForHuman(ctx, session, fmt.Sprintf("submission outcome uncertain: %v", err))
		return result, herr
	}

	switch state {
	case StateSubmitted:
		session.State = SessionCompleted
		session, err = r.saveSession(session)
		if err != nil {
			return Result{}, err
		}
		return Result{Session: session, Outcome: "submitted"}, nil
	case StateNormal:
		// The driver saw no explicit success signal. Treat as uncertain
		// rather than assuming success or retrying the click.
		return r.pauseForHuman(ctx, session, "submission outcome could not be confirmed")
	default:
		result, herr := r.handleNonNormalStateAfterSubmit(ctx, session, state)
		return result, herr
	}
}

func (r *Runner) handleNonNormalStateAfterSubmit(ctx context.Context, session Session, state PageState) (Result, error) {
	if state == StateCaptcha {
		return r.pauseForHuman(ctx, session, "CAPTCHA detected after submission attempt")
	}
	return r.pauseForHuman(ctx, session, fmt.Sprintf("unexpected state after submission: %s", state))
}

func (r *Runner) fail(ctx context.Context, session Session, reason string) (Result, error) {
	session.State = SessionFailed
	session.FailureReason = reason
	session, err := r.saveSession(session)
	return Result{Session: session, Outcome: "failed"}, err
}

func (r *Runner) expire(ctx context.Context, session Session) (Result, error) {
	session.State = SessionExpired
	session, err := r.saveSession(session)
	return Result{Session: session, Outcome: "expired"}, err
}

func (r *Runner) checkpoint(ctx context.Context, session Session, step string) (Session, error) {
	shot, err := r.Driver.Screenshot(ctx)
	var ref string
	if err == nil {
		ref, _ = r.Screens.Save(ctx, session.ID, shot)
	}

	now := r.Now()
	if _, err := r.Sessions.SaveCheckpoint(Checkpoint{
		ID:            r.NewID(),
		SessionID:     session.ID,
		Step:          step,
		CurrentURL:    session.CurrentURL,
		ScreenshotRef: ref,
		CreatedAt:     now,
	}); err != nil {
		return session, fmt.Errorf("browser: save checkpoint: %w", err)
	}

	session.LastCheckpointAt = now
	session.ScreenshotRef = ref
	return r.saveSession(session)
}

func (r *Runner) saveSession(s Session) (Session, error) {
	saved, err := r.Sessions.SaveSession(s)
	if err != nil {
		return Session{}, fmt.Errorf("browser: save session: %w", err)
	}
	return saved, nil
}

func (r *Runner) isExpired(s Session) bool {
	return s.IsExpired(r.Now())
}

func (r *Runner) startOfToday() time.Time {
	now := r.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}
