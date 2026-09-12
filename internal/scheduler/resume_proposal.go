package scheduler

import (
	"context"
	"time"
)

// Risk classifies how significant a proposed resume change is.
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// ResumeProposal is a scheduled suggestion for updating a resume version.
// It is never auto-applied — docs/IMPLEMENTATION-PLAN.md section 1.3:
// "No automatic approval during MVP." A human must review and approve it
// through the existing resume.Approve workflow before it becomes a new
// resume.Version.
type ResumeProposal struct {
	ID                string
	ProfileID         string
	Variant           string
	ProposedChanges   []string
	Reason            string
	AffectedSections  []string
	PageCountEstimate int
	Risk              Risk

	CreatedAt time.Time
}

// ProposalStore persists resume proposals awaiting human review.
type ProposalStore interface {
	SaveProposal(p ResumeProposal) error
}

// ChangeDetector inspects recent candidate-profile activity and returns
// candidate change descriptions worth proposing. Kept as an interface so
// "what counts as a relevant change" stays outside the scheduler — this
// package only handles the scheduling/proposal-record shape, not the
// judgment of what's resume-worthy.
type ChangeDetector interface {
	DetectChanges(ctx context.Context, profileID string) ([]DetectedChange, error)
}

// DetectedChange is one candidate-profile change a ChangeDetector found.
type DetectedChange struct {
	Description string
	Section     string
	Risk        Risk
}

// ResumeProposalTask runs the scheduled resume-update-proposal flow
// (docs/IMPLEMENTATION-PLAN.md section 9): detect changes, produce a
// proposal, persist it for human review. It never compiles or approves a
// resume itself.
type ResumeProposalTask struct {
	Detector ChangeDetector
	Store    ProposalStore
	NewID    func() string
	Now      func() time.Time
}

// Run detects changes for profileID/variant and, if any are found,
// persists a single ResumeProposal summarizing them. Returns nil (no
// proposal) when there is nothing worth proposing — that is a normal,
// non-error outcome, not a failure.
func (t *ResumeProposalTask) Run(ctx context.Context, profileID, variant string) (*ResumeProposal, error) {
	changes, err := t.Detector.DetectChanges(ctx, profileID)
	if err != nil {
		return nil, err
	}
	if len(changes) == 0 {
		return nil, nil
	}

	proposal := ResumeProposal{
		ID:        t.NewID(),
		ProfileID: profileID,
		Variant:   variant,
		Reason:    "scheduled resume update proposal",
		CreatedAt: t.Now(),
		Risk:      RiskLow,
	}

	for _, c := range changes {
		proposal.ProposedChanges = append(proposal.ProposedChanges, c.Description)
		proposal.AffectedSections = append(proposal.AffectedSections, c.Section)
		if riskRank(c.Risk) > riskRank(proposal.Risk) {
			proposal.Risk = c.Risk
		}
	}

	if err := t.Store.SaveProposal(proposal); err != nil {
		return nil, err
	}

	return &proposal, nil
}

func riskRank(r Risk) int {
	switch r {
	case RiskHigh:
		return 3
	case RiskMedium:
		return 2
	default:
		return 1
	}
}
