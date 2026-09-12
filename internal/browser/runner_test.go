package browser

import (
	"context"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/application"
)

type fakeSessionStore struct {
	sessions    map[string]Session
	checkpoints []Checkpoint
	started     []time.Time
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]Session{}}
}

func (s *fakeSessionStore) GetSession(id string) (Session, error) {
	return s.sessions[id], nil
}

func (s *fakeSessionStore) SaveSession(sess Session) (Session, error) {
	if _, exists := s.sessions[sess.ID]; !exists {
		s.started = append(s.started, sess.StartedAt)
	}
	s.sessions[sess.ID] = sess
	return sess, nil
}

func (s *fakeSessionStore) SaveCheckpoint(c Checkpoint) (Checkpoint, error) {
	s.checkpoints = append(s.checkpoints, c)
	return c, nil
}

func (s *fakeSessionStore) ListCheckpoints(sessionID string) ([]Checkpoint, error) {
	var out []Checkpoint
	for _, c := range s.checkpoints {
		if c.SessionID == sessionID {
			out = append(out, c)
		}
	}
	return out, nil
}

func (s *fakeSessionStore) CountStartedSince(since time.Time) (int, error) {
	count := 0
	for _, t := range s.started {
		if !t.Before(since) {
			count++
		}
	}
	return count, nil
}

type fakeScreenshotStore struct {
	saved map[string][]Screenshot
}

func newFakeScreenshotStore() *fakeScreenshotStore {
	return &fakeScreenshotStore{saved: map[string][]Screenshot{}}
}

func (s *fakeScreenshotStore) Save(ctx context.Context, sessionID string, shot Screenshot) (string, error) {
	s.saved[sessionID] = append(s.saved[sessionID], shot)
	return sessionID + "-shot-" + string(rune('0'+len(s.saved[sessionID]))), nil
}

type fakeNotifier struct {
	calls []notifyCall
	err   error
}

type notifyCall struct {
	sessionID, applicationID, reason, recoveryURL string
}

func (n *fakeNotifier) NotifyHumanRequired(ctx context.Context, sessionID, applicationID, reason, recoveryURL string) error {
	n.calls = append(n.calls, notifyCall{sessionID, applicationID, reason, recoveryURL})
	return n.err
}

func newTestRunner(driver Driver, sessions *fakeSessionStore, tokens *fakeTokenStore, screens *fakeScreenshotStore, notifier *fakeNotifier) *Runner {
	ids := 0
	r := NewRunner(driver, sessions, tokens, screens, notifier, Limits{BrowserSessionsPerDay: 10, MaxSessionMinutes: 30}, func() string {
		ids++
		return "id-" + string(rune('0'+ids))
	})
	r.Now = func() time.Time { return time.Unix(1_000_000, 0) }
	return r
}

// --- Scenario: normal flow ---

func TestRunner_NormalFlow(t *testing.T) {
	driver := &FakeDriver{
		NavigateStates: []PageState{StateNormal},
		DetectStates:   []PageState{StateNormal},
	}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	result, err := r.Start(context.Background(), StartInput{
		ApplicationID: "app-1",
		Platform:      "greenhouse",
		StartURL:      "https://example.com/apply",
		Fields:        []FieldValue{{Selector: "#name", Value: "Ada"}},
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if result.Outcome != "ready_for_review" {
		t.Errorf("Outcome = %q, want ready_for_review", result.Outcome)
	}
	if result.Session.State != SessionRunning {
		t.Errorf("Session.State = %s, want RUNNING", result.Session.State)
	}
	if len(sessions.checkpoints) != 1 {
		t.Errorf("checkpoints = %d, want 1", len(sessions.checkpoints))
	}
}

// --- Scenario: CAPTCHA interruption ---

func TestRunner_CaptchaInterruption(t *testing.T) {
	driver := &FakeDriver{
		NavigateStates: []PageState{StateCaptcha},
	}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	result, err := r.Start(context.Background(), StartInput{
		ApplicationID: "app-1",
		StartURL:      "https://example.com/apply",
	})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	if result.Outcome != "human_required" {
		t.Errorf("Outcome = %q, want human_required", result.Outcome)
	}
	if result.Session.State != SessionHumanReq {
		t.Errorf("Session.State = %s, want HUMAN_REQUIRED", result.Session.State)
	}
	if !result.Session.HumanRequired {
		t.Error("expected HumanRequired = true")
	}
	if result.Session.ScreenshotRef == "" {
		t.Error("expected screenshot to be captured")
	}
	if len(notifier.calls) != 1 {
		t.Fatalf("notifier calls = %d, want 1", len(notifier.calls))
	}
	if notifier.calls[0].reason != "CAPTCHA detected" {
		t.Errorf("reason = %q, want CAPTCHA detected", notifier.calls[0].reason)
	}
	if len(tokens.hashes) != 1 {
		t.Errorf("recovery tokens saved = %d, want 1", len(tokens.hashes))
	}
}

// --- Scenario: resume after human handles CAPTCHA ---

func TestRunner_Resume(t *testing.T) {
	driver := &FakeDriver{
		NavigateStates: []PageState{StateCaptcha},
	}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	start, err := r.Start(context.Background(), StartInput{ApplicationID: "app-1", StartURL: "https://example.com/apply"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}

	// Recover the raw token the same way a real recovery link would carry
	// it: from the notification the runner sent.
	rawToken := notifier.calls[0].recoveryURL[len("/recovery/"):]

	// Human "solved" the CAPTCHA; driver now reports a normal page.
	driver.DetectStates = []PageState{StateNormal, StateNormal}

	result, err := r.Resume(context.Background(), rawToken, []FieldValue{{Selector: "#name", Value: "Ada"}})
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if result.Outcome != "ready_for_review" {
		t.Errorf("Outcome = %q, want ready_for_review", result.Outcome)
	}
	if result.Session.ID != start.Session.ID {
		t.Errorf("resumed session ID = %q, want %q", result.Session.ID, start.Session.ID)
	}

	// Token must be single-use.
	if _, err := Redeem(tokens, rawToken, r.Now()); err == nil {
		t.Error("expected recovery token to be single-use")
	}
}

func TestRunner_Resume_PageStillNotNormal(t *testing.T) {
	driver := &FakeDriver{NavigateStates: []PageState{StateCaptcha}}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	_, err := r.Start(context.Background(), StartInput{ApplicationID: "app-1", StartURL: "https://example.com/apply"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	rawToken := notifier.calls[0].recoveryURL[len("/recovery/"):]

	driver.DetectStates = []PageState{StateCaptcha} // still blocked

	result, err := r.Resume(context.Background(), rawToken, nil)
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if result.Outcome != "human_required" {
		t.Errorf("Outcome = %q, want human_required (still blocked)", result.Outcome)
	}
}

// --- Scenario: session expiry ---

func TestRunner_SessionExpiry(t *testing.T) {
	driver := &FakeDriver{NavigateStates: []PageState{StateCaptcha}}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)
	r.Limits.MaxSessionMinutes = 1

	start, err := r.Start(context.Background(), StartInput{ApplicationID: "app-1", StartURL: "https://example.com/apply"})
	if err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	rawToken := notifier.calls[0].recoveryURL[len("/recovery/"):]

	// Advance clock past expiry.
	r.Now = func() time.Time { return time.Unix(1_000_000, 0).Add(2 * time.Minute) }
	tokens.hashes[HashRecoveryToken(rawToken)] = tokenEntry{
		sessionID: start.Session.ID,
		expiresAt: r.Now().Add(time.Hour), // token itself still valid; session is what's expired
	}

	result, err := r.Resume(context.Background(), rawToken, nil)
	if err != nil {
		t.Fatalf("Resume() error: %v", err)
	}
	if result.Outcome != "expired" {
		t.Errorf("Outcome = %q, want expired", result.Outcome)
	}
	if result.Session.State != SessionExpired {
		t.Errorf("Session.State = %s, want EXPIRED", result.Session.State)
	}
}

// --- Scenario: failed navigation ---

func TestRunner_FailedNavigation(t *testing.T) {
	driver := &FakeDriver{NavigateErr: context.DeadlineExceeded}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	result, err := r.Start(context.Background(), StartInput{ApplicationID: "app-1", StartURL: "https://example.com/apply"})
	if err != nil {
		t.Fatalf("Start() returned unexpected top-level error: %v", err)
	}
	if result.Outcome != "failed" {
		t.Errorf("Outcome = %q, want failed", result.Outcome)
	}
	if result.Session.State != SessionFailed {
		t.Errorf("Session.State = %s, want FAILED", result.Session.State)
	}
	if result.Session.FailureReason == "" {
		t.Error("expected FailureReason to be set")
	}
}

// --- Scenario: daily session limit (guards against unbounded sessions) ---

func TestRunner_DailySessionLimitReached(t *testing.T) {
	driver := &FakeDriver{NavigateStates: []PageState{StateNormal}}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)
	r.Limits.BrowserSessionsPerDay = 1

	if _, err := r.Start(context.Background(), StartInput{ApplicationID: "app-1", StartURL: "https://example.com/apply"}); err != nil {
		t.Fatalf("first Start() error: %v", err)
	}

	if _, err := r.Start(context.Background(), StartInput{ApplicationID: "app-2", StartURL: "https://example.com/apply"}); err == nil {
		t.Fatal("second Start() expected error (daily limit reached), got nil")
	}
}

// --- Scenario: submission uncertainty ---

func passedGate() application.SafetyGateResult {
	return application.SafetyGateResult{Passed: true}
}

func TestRunner_Submit_ClearSuccess(t *testing.T) {
	driver := &FakeDriver{SubmitState: StateSubmitted}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	session := Session{ID: "sess-1", ApplicationID: "app-1", ExpiresAt: r.Now().Add(time.Hour)}

	result, err := r.Submit(context.Background(), SubmitInput{
		Session:        session,
		GateResult:     passedGate(),
		SubmitSelector: "#submit",
	})
	if err != nil {
		t.Fatalf("Submit() error: %v", err)
	}
	if result.Outcome != "submitted" {
		t.Errorf("Outcome = %q, want submitted", result.Outcome)
	}
	if result.Session.State != SessionCompleted {
		t.Errorf("Session.State = %s, want COMPLETED", result.Session.State)
	}
}

func TestRunner_Submit_UncertainOutcome_NoRetry(t *testing.T) {
	driver := &FakeDriver{SubmitState: StateNormal} // no explicit success signal
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	session := Session{ID: "sess-1", ApplicationID: "app-1", ExpiresAt: r.Now().Add(time.Hour)}

	result, err := r.Submit(context.Background(), SubmitInput{
		Session:        session,
		GateResult:     passedGate(),
		SubmitSelector: "#submit",
	})
	if err != nil {
		t.Fatalf("Submit() error: %v", err)
	}
	if result.Outcome != "human_required" {
		t.Errorf("Outcome = %q, want human_required (uncertain outcome must not be silently treated as success)", result.Outcome)
	}
	if len(notifier.calls) != 1 {
		t.Errorf("notifier calls = %d, want exactly 1 (no retry loop)", len(notifier.calls))
	}
}

func TestRunner_Submit_ErrorDuringSubmit_NoRetry(t *testing.T) {
	driver := &FakeDriver{SubmitErr: context.DeadlineExceeded}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	session := Session{ID: "sess-1", ApplicationID: "app-1", ExpiresAt: r.Now().Add(time.Hour)}

	result, err := r.Submit(context.Background(), SubmitInput{
		Session:        session,
		GateResult:     passedGate(),
		SubmitSelector: "#submit",
	})
	if err != nil {
		t.Fatalf("Submit() error: %v", err)
	}
	if result.Outcome != "human_required" {
		t.Errorf("Outcome = %q, want human_required", result.Outcome)
	}
}

func TestRunner_Submit_RefusesWithoutPassedGate(t *testing.T) {
	driver := &FakeDriver{SubmitState: StateSubmitted}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	session := Session{ID: "sess-1", ApplicationID: "app-1", ExpiresAt: r.Now().Add(time.Hour)}
	failedGate := application.SafetyGateResult{Passed: false, Reasons: []string{"platform automation is denied"}}

	_, err := r.Submit(context.Background(), SubmitInput{
		Session:        session,
		GateResult:     failedGate,
		SubmitSelector: "#submit",
	})
	if err == nil {
		t.Fatal("Submit() expected error when safety gate failed, got nil")
	}
	if driver.CurrentURLValue != "" {
		t.Error("driver should never have been invoked when gate failed")
	}
}

// --- Scenario: duplicate prevention is enforced at the application layer,
// but the runner must not be invoked for a duplicate — verify Submit still
// requires an explicit passed gate, which application.EvaluateSafetyGate
// would refuse to grant for a duplicate. ---

func TestRunner_Submit_RefusesForDuplicateGateResult(t *testing.T) {
	driver := &FakeDriver{SubmitState: StateSubmitted}
	sessions := newFakeSessionStore()
	tokens := newFakeTokenStore()
	screens := newFakeScreenshotStore()
	notifier := &fakeNotifier{}
	r := newTestRunner(driver, sessions, tokens, screens, notifier)

	gate := application.EvaluateSafetyGate(application.SafetyGateInput{
		JobVerified:    true,
		ResumeVerified: true,
		IsDuplicate:    true,
		Platform:       application.PlatformPolicyAllowed,
	})

	session := Session{ID: "sess-1", ApplicationID: "app-1", ExpiresAt: r.Now().Add(time.Hour)}

	_, err := r.Submit(context.Background(), SubmitInput{Session: session, GateResult: gate, SubmitSelector: "#submit"})
	if err == nil {
		t.Fatal("Submit() expected error for duplicate application, got nil")
	}
}
