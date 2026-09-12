// Package job implements job ingestion, normalization, deduplication, and
// scoring — CareerPilot Phase 2 "Job Intelligence". See
// docs/IMPLEMENTATION-PLAN.md section 2 and docs/ARCHITECTURE.md's
// "Job Ingestion -> Job Intelligence" pipeline.
package job

import "time"

// Status is the job's position in the intelligence pipeline.
type Status string

const (
	StatusDiscovered Status = "DISCOVERED"
	StatusNormalized Status = "NORMALIZED"
	StatusTriaged    Status = "TRIAGED"
	StatusEvaluated  Status = "EVALUATED"
	StatusQualified  Status = "QUALIFIED"
	StatusRejected   Status = "REJECTED"
	StatusDuplicate  Status = "DUPLICATE"
)

// Job is the canonical internal representation of a job posting. Raw
// source data is retained verbatim in RawData for auditability
// (docs/IMPLEMENTATION-PLAN.md section 4: "retain the original job data").
type Job struct {
	ID           string
	Source       string
	SourceJobID  string // stable identity from the source, if the source provides one
	URL          string
	Company      string
	Title        string
	Location     string
	Description  string
	RemotePolicy string

	RawData map[string]string // verbatim source fields, for audit

	NormalizedHash string // deterministic fingerprint, used when SourceJobID is unavailable
	DuplicateOfID  string // set when Status == StatusDuplicate

	Status Status

	DiscoveredAt time.Time
	UpdatedAt    time.Time
}

// Store defines the persistence boundary for jobs.
type Store interface {
	GetJob(id string) (Job, error)
	FindBySourceJobID(source, sourceJobID string) (Job, bool, error)
	FindByNormalizedHash(hash string) (Job, bool, error)
	SaveJob(j Job) (Job, error)
	CountDiscoveredSince(since time.Time) (int, error)
}
