// Package browser implements CareerPilot's browser runner — an isolated
// component that performs navigation, form filling, and screenshotting on
// behalf of an in-progress application. See docs/ARCHITECTURE.md
// "Browser Layer" and docs/BROWSER-RECOVERY.md.
//
// The browser runner never owns career/business state (CLAUDE.md
// "Architecture Principles"): it reports what it observes (PageState,
// screenshots, navigation results) and executes deterministic field
// fills the caller supplies. Decisions about what those observations mean
// for an Application belong to internal/application.
package browser

import "context"

// FieldValue is one deterministic field to fill, addressed by a selector
// the caller already resolved (e.g. from a platform-specific form
// mapping). The Driver does not infer what a field means.
type FieldValue struct {
	Selector string
	Value    string
}

// PageState is what the Driver observed after navigating or filling.
// Detection is deliberately conservative: anything the driver cannot
// positively classify must come back as StateUnknown, which callers must
// treat as HUMAN_REQUIRED, never as an implicit pass
// (CLAUDE.md "Never guess").
type PageState string

const (
	StateNormal      PageState = "normal"
	StateCaptcha     PageState = "captcha"
	StateLoginWall   PageState = "login_wall"
	StateUnknown     PageState = "unknown"
	StateSubmitted   PageState = "submitted" // driver observed a clear success signal
	StateNavigateErr PageState = "navigate_error"
)

// Screenshot is a captured image plus a reference for where it was
// stored; the Driver does not decide long-term storage, it just captures.
type Screenshot struct {
	Data     []byte
	MimeType string
}

// Driver is the isolation boundary around actual browser automation. All
// business logic (internal/browserrunner, internal/application) depends
// only on this interface — never on Playwright types directly
// (docs/ARCHITECTURE.md "The rest of the system should not depend
// directly on a specific provider" applied to browser automation too).
//
// Implementations must never:
//   - solve or bypass CAPTCHA
//   - rotate proxies or spoof fingerprints to evade detection
//   - retry a failed action indefinitely
//
// See CLAUDE.md "CAPTCHA and anti-bot handling".
type Driver interface {
	// Launch starts a new browser context for a session.
	Launch(ctx context.Context) error
	// Navigate goes to url and reports the resulting PageState.
	Navigate(ctx context.Context, url string) (PageState, error)
	// Fill deterministically fills the given fields. It does not submit.
	Fill(ctx context.Context, fields []FieldValue) error
	// DetectState re-inspects the current page without navigating.
	DetectState(ctx context.Context) (PageState, error)
	// Screenshot captures the current page.
	Screenshot(ctx context.Context) (Screenshot, error)
	// CurrentURL returns the page's current URL.
	CurrentURL(ctx context.Context) (string, error)
	// Submit performs the final submission action (e.g. clicking a submit
	// button) identified by selector. Callers must only invoke this after
	// the application safety gate has passed.
	Submit(ctx context.Context, selector string) (PageState, error)
	// Close releases the browser context. Safe to call multiple times.
	Close(ctx context.Context) error
}
