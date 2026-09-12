package resume

import (
	"context"
	"fmt"
	"time"

	"github.com/kartik-labs/careerpilot/internal/candidate"
	"github.com/kartik-labs/careerpilot/internal/resume/latex"
)

// GenerateInput carries everything the pipeline needs to produce one
// candidate resume Version. Data is assumed already validated against
// candidate.Claim provenance by the caller — this pipeline does not
// re-derive claim text, only orchestrates render -> compile -> validate.
type GenerateInput struct {
	ProfileID      string
	ProfileVersion int
	Variant        VariantType
	VersionNumber  int
	TargetRole     string
	JobID          string

	TemplateName   string
	TemplateSource string
	Data           latex.Data
	Changes        []Change
}

// Pipeline orchestrates resume generation:
// render LaTeX -> compile PDF -> validate page count -> await approval.
// It never itself decides to approve; APPROVED/REJECTED only happens via
// an explicit human decision recorded through Approve/Reject.
type Pipeline struct {
	Compiler latex.Compiler
}

// NewPipeline builds a Pipeline using the given LaTeX compiler.
func NewPipeline(compiler latex.Compiler) *Pipeline {
	return &Pipeline{Compiler: compiler}
}

// Generate renders and compiles a new resume Version. On success the
// version's Status is VALIDATED if the page count passes, or REVIEW if it
// does not — generation never fails the whole pipeline just because a
// resume needs shortening; it surfaces that as a REVIEW state with
// structured reasons instead.
func (p *Pipeline) Generate(ctx context.Context, in GenerateInput) (Version, error) {
	texSource, err := latex.Render(in.TemplateName, in.TemplateSource, in.Data)
	if err != nil {
		return Version{}, fmt.Errorf("resume: render: %w", err)
	}

	v := Version{
		ProfileID:      in.ProfileID,
		ProfileVersion: in.ProfileVersion,
		Variant:        in.Variant,
		VersionNumber:  in.VersionNumber,
		TargetRole:     in.TargetRole,
		JobID:          in.JobID,
		SourceTeX:      texSource,
		Changes:        in.Changes,
		Status:         StatusDraft,
		CreatedAt:      time.Now(),
	}

	result, err := p.Compiler.Compile(ctx, texSource)
	if err != nil {
		return Version{}, fmt.Errorf("resume: compile: %w", err)
	}

	v.GeneratedPDF = result.PDF
	v.PageCount = result.PageCount
	v.Status = StatusGenerated

	validation := ValidatePageCount(result.PageCount)
	v.Validation = &validation

	if validation.Passed {
		v.Status = StatusValidated
	} else {
		v.Status = StatusReview
	}

	return v, nil
}

// Approve transitions a version to APPROVED. It is the only path by which
// a resume version becomes eligible for use in an application
// (CLAUDE.md "Human-in-the-loop").
func Approve(v Version) (Version, error) {
	if !ValidTransition(v.Status, StatusApproved) {
		return Version{}, fmt.Errorf("resume: cannot approve version in status %s", v.Status)
	}
	v.Status = StatusApproved
	v.ApprovedAt = time.Now()
	return v, nil
}

// Reject transitions a version to REJECTED.
func Reject(v Version) (Version, error) {
	if !ValidTransition(v.Status, StatusRejected) {
		return Version{}, fmt.Errorf("resume: cannot reject version in status %s", v.Status)
	}
	v.Status = StatusRejected
	return v, nil
}

// ValidateClaimProvenance checks that every claimID referenced by changes
// exists among allowed claims. If any claim is missing or not allowed for
// resume use, it returns those offending claim IDs so the caller can route
// to REVIEW rather than silently dropping or inventing content
// (docs/AGENT-POLICY.md "Resume Rules").
func ValidateClaimProvenance(changes []Change, allowed []candidate.Claim) (missing []string) {
	allowedSet := make(map[string]bool, len(allowed))
	for _, c := range allowed {
		if c.AllowedForResume {
			allowedSet[c.ID] = true
		}
	}

	for _, change := range changes {
		for _, claimID := range change.ClaimIDs {
			if !allowedSet[claimID] {
				missing = append(missing, claimID)
			}
		}
	}

	return missing
}
