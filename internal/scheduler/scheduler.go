// Package scheduler implements CareerPilot Phase 5: scheduled operations
// (job discovery, job evaluation, resume update proposals) and the
// operational scaffolding around them — retry policy, dead-letter
// handling, health checks, audit reporting. See
// docs/ARCHITECTURE.md "Orchestrator" (Cloudflare Worker: cron, API,
// job dispatch) and CLAUDE.md "Initial Resource Limits".
//
// This package defines *what* each scheduled run does and how failures
// are classified; it does not implement its own timer/loop. The actual
// trigger (Cloudflare Worker cron, a CLI invocation, a test) calls
// RunX once per invocation — there is no goroutine that loops forever
// (CLAUDE.md "Avoid: ... uncontrolled model loops"; Phase 5 prompt:
// "Never allow an infinite loop.").
package scheduler

import "time"

// TaskType names a schedulable operation, used for run records, retry
// policy lookup, and audit reporting.
type TaskType string

const (
	TaskJobDiscovery   TaskType = "job_discovery"
	TaskJobEvaluation  TaskType = "job_evaluation"
	TaskResumeProposal TaskType = "resume_proposal"
)

// RunStatus is the outcome of one scheduled task invocation.
type RunStatus string

const (
	RunSucceeded  RunStatus = "SUCCEEDED"
	RunFailed     RunStatus = "FAILED"
	RunDeadLetter RunStatus = "DEAD_LETTER" // exhausted retries; needs human attention
	RunSkipped    RunStatus = "SKIPPED"     // e.g. daily limit already reached
)

// Run is an audit record of a single scheduled task invocation
// (CLAUDE.md "Every important state transition should be auditable.").
type Run struct {
	ID      string
	Task    TaskType
	Status  RunStatus
	Attempt int
	Detail  string // human-readable summary or failure reason

	StartedAt  time.Time
	FinishedAt time.Time
}

// RunStore persists scheduler Run records.
type RunStore interface {
	SaveRun(r Run) error
	ListRuns(task TaskType, since time.Time) ([]Run, error)
}
