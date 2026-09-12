// Package application implements CareerPilot Phase 3: Application
// Preparation — the domain model, state machine, resume/answer
// preparation, and safety gate that produce a complete application
// package ready for human submission or the future browser runner. See
// CLAUDE.md "Application Safety Gate" and
// docs/IMPLEMENTATION-PLAN.md section 4.
package application

import "time"

// Status is the application's position in its lifecycle. This is the
// minimum state set required by the phase spec; browser-execution states
// (STARTING..SUBMITTED, HUMAN_REQUIRED) are modeled now so the state
// machine is complete end-to-end, even though this phase does not
// implement the browser runner that drives them.
type Status string

const (
	StatusDiscovered Status = "DISCOVERED"
	StatusQualified  Status = "QUALIFIED"
	StatusPreparing  Status = "PREPARING"
	StatusReview     Status = "REVIEW"
	StatusReady      Status = "READY"
	StatusStarting   Status = "STARTING"
	StatusNavigating Status = "NAVIGATING"
	StatusFilling    Status = "FILLING"
	StatusVerifying  Status = "VERIFYING"
	StatusSubmitting Status = "SUBMITTING"
	StatusSubmitted  Status = "SUBMITTED"
	StatusHumanReq   Status = "HUMAN_REQUIRED"
	StatusFailed     Status = "FAILED"
	StatusExpired    Status = "EXPIRED"
	StatusCancelled  Status = "CANCELLED"
)

// Application is a single job application in progress. Fields beyond
// PREPARING/REVIEW/READY (resume, answers, safety gate result) are the
// "complete application package" this phase must produce; STARTING onward
// is populated by the future browser runner, not this package.
type Application struct {
	ID    string
	JobID string

	ProfileID       string
	ResumeVersionID string

	Answers []Answer

	Status Status

	SafetyGate *SafetyGateResult

	DuplicateOfID string // set if this application was blocked as a duplicate

	CreatedAt time.Time
	UpdatedAt time.Time
}

// Store defines the persistence boundary for applications.
type Store interface {
	GetApplication(id string) (Application, error)
	FindByJobID(jobID string) (Application, bool, error)
	SaveApplication(a Application) (Application, error)
	CountSubmittedSince(since time.Time) (int, error)
}
