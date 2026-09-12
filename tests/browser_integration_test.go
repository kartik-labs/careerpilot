// Package tests holds cross-package integration tests that exercise more
// than one internal package together (see README.md "tests/").
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/application"
	"github.com/kartik-labs/careerpilot/internal/browser"
)

type fakeApplicationStore struct {
	byJobID map[string]application.Application
}

func newFakeApplicationStore() *fakeApplicationStore {
	return &fakeApplicationStore{byJobID: map[string]application.Application{}}
}

func (s *fakeApplicationStore) GetApplication(id string) (application.Application, error) {
	return application.Application{}, nil
}

func (s *fakeApplicationStore) FindByJobID(jobID string) (application.Application, bool, error) {
	a, ok := s.byJobID[jobID]
	return a, ok, nil
}

func (s *fakeApplicationStore) SaveApplication(a application.Application) (application.Application, error) {
	s.byJobID[a.JobID] = a
	return a, nil
}

func (s *fakeApplicationStore) CountSubmittedSince(since time.Time) (int, error) {
	return 0, nil
}

type fakeSessionStore struct {
	sessions map[string]browser.Session
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{sessions: map[string]browser.Session{}}
}

func (s *fakeSessionStore) GetSession(id string) (browser.Session, error) {
	return s.sessions[id], nil
}
func (s *fakeSessionStore) SaveSession(sess browser.Session) (browser.Session, error) {
	s.sessions[sess.ID] = sess
	return sess, nil
}
func (s *fakeSessionStore) SaveCheckpoint(c browser.Checkpoint) (browser.Checkpoint, error) {
	return c, nil
}
func (s *fakeSessionStore) ListCheckpoints(sessionID string) ([]browser.Checkpoint, error) {
	return nil, nil
}
func (s *fakeSessionStore) CountStartedSince(since time.Time) (int, error) { return 0, nil }

type fakeTokenStore struct {
	hashes map[string]string
}

func (s *fakeTokenStore) SaveTokenHash(hash, sessionID string, expiresAt time.Time) error {
	if s.hashes == nil {
		s.hashes = map[string]string{}
	}
	s.hashes[hash] = sessionID
	return nil
}
func (s *fakeTokenStore) LookupByHash(hash string, now time.Time) (string, bool, error) {
	id, ok := s.hashes[hash]
	return id, ok, nil
}
func (s *fakeTokenStore) RevokeByHash(hash string) error {
	delete(s.hashes, hash)
	return nil
}

type fakeScreenshotStore struct{}

func (fakeScreenshotStore) Save(ctx context.Context, sessionID string, shot browser.Screenshot) (string, error) {
	return "ref", nil
}

type fakeNotifier struct{ notified int }

func (n *fakeNotifier) NotifyHumanRequired(ctx context.Context, sessionID, applicationID, reason, recoveryURL string) error {
	n.notified++
	return nil
}

// TestDuplicatePrevention_BlocksBeforeBrowserRunnerEverLaunches verifies
// the full path: an application already exists for a job, so the safety
// gate must fail on duplicate, and the browser runner must refuse Submit
// without ever invoking the driver — duplicate prevention happens at the
// application layer, and the browser layer enforces it never gets bypassed
// (CLAUDE.md "Application Safety Gate": "duplicate application status").
func TestDuplicatePrevention_BlocksBeforeBrowserRunnerEverLaunches(t *testing.T) {
	appStore := newFakeApplicationStore()
	existing := application.Application{ID: "app-existing", JobID: "job-1", Status: application.StatusSubmitted}
	appStore.SaveApplication(existing)

	_, isDup, err := application.CheckDuplicate(appStore, "job-1")
	if err != nil {
		t.Fatalf("CheckDuplicate() error: %v", err)
	}
	if !isDup {
		t.Fatal("expected duplicate to be detected")
	}

	gate := application.EvaluateSafetyGate(application.SafetyGateInput{
		JobVerified:    true,
		ResumeVerified: true,
		IsDuplicate:    isDup,
		Platform:       application.PlatformPolicyAllowed,
	})
	if gate.Passed {
		t.Fatal("expected safety gate to fail for duplicate application")
	}

	driver := &browser.FakeDriver{SubmitState: browser.StateSubmitted}
	sessions := newFakeSessionStore()
	tokens := &fakeTokenStore{}
	notifier := &fakeNotifier{}
	runner := browser.NewRunner(driver, sessions, tokens, fakeScreenshotStore{}, notifier, browser.Limits{BrowserSessionsPerDay: 10, MaxSessionMinutes: 30}, func() string { return "id-1" })

	session := browser.Session{ID: "sess-1", ApplicationID: "app-new", ExpiresAt: time.Now().Add(time.Hour)}

	_, err = runner.Submit(context.Background(), browser.SubmitInput{
		Session:        session,
		GateResult:     gate,
		SubmitSelector: "#submit",
	})
	if err == nil {
		t.Fatal("Submit() expected error for duplicate application, got nil")
	}
	if driver.CurrentURLValue != "" || len(driver.NavigatedURLs) != 0 {
		t.Error("driver must never be invoked when the safety gate already failed on duplicate")
	}
}

// TestUnknownPlatformPolicy_DefaultsToReview verifies CLAUDE.md's rule
// that an unknown platform automation policy must default to human
// review, never to permitted-by-default.
func TestUnknownPlatformPolicy_DefaultsToReview(t *testing.T) {
	gate := application.EvaluateSafetyGate(application.SafetyGateInput{
		JobVerified:    true,
		ResumeVerified: true,
		IsDuplicate:    false,
		Platform:       application.PlatformPolicyUnknown,
	})
	if gate.Passed {
		t.Fatal("expected safety gate to fail when platform policy is unknown")
	}
}
