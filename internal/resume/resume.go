// Package resume defines the resume variant/version domain model and the
// generation pipeline: Candidate Profile -> Base Resume -> Tailoring ->
// LaTeX -> PDF -> Page Validation -> Human Approval. See
// docs/RESUME-MANAGEMENT.md and docs/IMPLEMENTATION-PLAN.md section 1.
package resume

import "time"

// VariantType identifies which resume positioning a version was generated
// for. CareerPilot initially supports exactly these two (CLAUDE.md
// "Resume Rules").
type VariantType string

const (
	VariantSoftwareEngineer VariantType = "software-engineer"
	VariantFDE              VariantType = "fde"
)

// Status is the lifecycle state of a resume version. Transitions are
// enforced by ValidTransition, not left implicit.
type Status string

const (
	StatusDraft     Status = "DRAFT"
	StatusGenerated Status = "GENERATED"
	StatusValidated Status = "VALIDATED"
	StatusReview    Status = "REVIEW"
	StatusApproved  Status = "APPROVED"
	StatusRejected  Status = "REJECTED"
)

// validTransitions enumerates the only allowed Status -> Status edges.
// See docs/IMPLEMENTATION-PLAN.md section 1.4 (LaTeX pipeline) and
// docs/RESUME-MANAGEMENT.md (versioning, one-page requirement).
var validTransitions = map[Status][]Status{
	StatusDraft:     {StatusGenerated},
	StatusGenerated: {StatusValidated, StatusReview},
	StatusValidated: {StatusReview, StatusApproved},
	StatusReview:    {StatusApproved, StatusRejected, StatusGenerated},
	StatusApproved:  {},
	StatusRejected:  {},
}

// ValidTransition reports whether moving from `from` to `to` is allowed.
func ValidTransition(from, to Status) bool {
	for _, allowed := range validTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Change records a single edit applied when producing a new version, so
// version history stays auditable (CLAUDE.md "A resume version must
// retain ... changes").
type Change struct {
	Description string
	ClaimIDs    []string // candidate.Claim IDs this change draws from
}

// Version is a single generated/approved resume artifact.
//
// Never mutate an approved historical Version in place — create a new
// Version instead (docs/RESUME-MANAGEMENT.md "Versioning").
type Version struct {
	ID             string
	ProfileID      string
	ProfileVersion int

	Variant       VariantType
	VersionNumber int

	TargetRole string // empty for a base (non-job-specific) version
	JobID      string // empty for a base version

	SourceTeX    string
	GeneratedPDF []byte
	PageCount    int

	Changes []Change
	Status  Status

	Validation *ValidationResult

	CreatedAt  time.Time
	ApprovedAt time.Time
}

// Store defines the persistence boundary for resume versions.
type Store interface {
	GetVersion(id string) (Version, error)
	ListVersions(profileID string, variant VariantType) ([]Version, error)
	LatestApproved(profileID string, variant VariantType) (Version, error)
	SaveVersion(v Version) (Version, error)
}
