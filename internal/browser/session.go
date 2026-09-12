package browser

import "time"

// SessionState is the browser session lifecycle
// (docs/BROWSER-RECOVERY.md "Session Lifecycle").
type SessionState string

const (
	SessionQueued    SessionState = "QUEUED"
	SessionStarting  SessionState = "STARTING"
	SessionRunning   SessionState = "RUNNING"
	SessionHumanReq  SessionState = "HUMAN_REQUIRED"
	SessionResumed   SessionState = "RESUMED"
	SessionCompleted SessionState = "COMPLETED"
	SessionFailed    SessionState = "FAILED"
	SessionExpired   SessionState = "EXPIRED"
)

// Session is a browser automation session, kept entirely separate from
// Application business state (CLAUDE.md: "The browser runner must never
// own career/business state."). It never stores raw cookies or
// credentials — see BrowserContextRef.
type Session struct {
	ID            string
	ApplicationID string
	Platform      string

	CurrentURL string
	State      SessionState

	StartedAt        time.Time
	LastCheckpointAt time.Time
	ScreenshotRef    string // opaque storage reference, not the raw image
	ExpiresAt        time.Time

	HumanRequired bool
	FailureReason string

	// BrowserContextRef is an opaque reference to where the browser
	// context (cookies, local storage, etc.) is stored out-of-band —
	// never inlined into this record (CLAUDE.md "Never persist raw
	// browser cookies or credentials as ordinary application data.").
	BrowserContextRef string
}

// IsExpired reports whether the session has passed its expiration time.
func (s Session) IsExpired(now time.Time) bool {
	return !s.ExpiresAt.IsZero() && now.After(s.ExpiresAt)
}

// Checkpoint is a persisted point-in-time snapshot of session progress,
// saved after each meaningful application step
// (docs/BROWSER-RECOVERY.md "Checkpoints").
type Checkpoint struct {
	ID        string
	SessionID string

	Step          string // e.g. "navigation", "resume_upload", "form_section", "before_submission"
	CurrentURL    string
	ScreenshotRef string

	CreatedAt time.Time
}

// Store defines the persistence boundary for browser sessions and
// checkpoints, kept separate from application.Store
// (CLAUDE.md "Persist browser checkpoint metadata separately.").
type Store interface {
	GetSession(id string) (Session, error)
	SaveSession(s Session) (Session, error)

	SaveCheckpoint(c Checkpoint) (Checkpoint, error)
	ListCheckpoints(sessionID string) ([]Checkpoint, error)

	CountStartedSince(since time.Time) (int, error)
}
