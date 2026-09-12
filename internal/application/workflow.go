package application

import (
	"fmt"
	"time"

	"github.com/kartik-labs/careerpilot/internal/job"
	"github.com/kartik-labs/careerpilot/internal/resume"
)

// FromJob creates a new Application in DISCOVERED status for a job. It
// does not itself decide qualification — that is driven by the job
// package's Score/Recommendation and applied via Qualify.
func FromJob(newID func() string, now func() time.Time, j job.Job, profileID string) Application {
	return Application{
		ID:        newID(),
		JobID:     j.ID,
		ProfileID: profileID,
		Status:    StatusDiscovered,
		CreatedAt: now(),
		UpdatedAt: now(),
	}
}

// Qualify transitions an application to QUALIFIED, but only if the job's
// score actually recommends applying. A REVIEW or REJECT score must never
// be silently treated as qualified (CLAUDE.md "Never convert REVIEW or
// STOP into AUTO merely to make the workflow continue").
func Qualify(a Application, score job.Score) (Application, error) {
	if score.Recommendation != job.RecommendationApply {
		return Application{}, fmt.Errorf("application: cannot qualify job with recommendation %s", score.Recommendation)
	}
	return Transition(a, StatusQualified)
}

// SelectResumeVariant maps a job's evaluated skill signals to one of
// CareerPilot's two supported resume positionings
// (docs/RESUME-MANAGEMENT.md "Resume Types"). This is deterministic Go
// logic, not a model call — a triage/evaluation stage may *suggest* a
// variant (job.TriageOutput.RecommendedResume), but the final selection
// here validates that suggestion against a known variant rather than
// trusting it blindly.
func SelectResumeVariant(recommended string) resume.VariantType {
	switch recommended {
	case string(resume.VariantFDE):
		return resume.VariantFDE
	default:
		return resume.VariantSoftwareEngineer
	}
}

// StartPreparing transitions an application to PREPARING. Actual resume
// generation is performed by the caller via resume.Pipeline.Generate
// (this package does not depend on internal/resume/latex directly to
// avoid a wide import surface); AttachResume then records the result.
func StartPreparing(a Application) (Application, error) {
	return Transition(a, StatusPreparing)
}

// AttachResume records which resume version an application will use and
// moves it to REVIEW — a generated resume is never used automatically
// without a review step, matching resume.Version's own VALIDATED/REVIEW
// gating (docs/RESUME-MANAGEMENT.md).
func AttachResume(a Application, version resume.Version) (Application, error) {
	if version.Status != resume.StatusValidated && version.Status != resume.StatusApproved {
		return Application{}, fmt.Errorf("application: resume version must be VALIDATED or APPROVED, got %s", version.Status)
	}
	a.ResumeVersionID = version.ID
	return Transition(a, StatusReview)
}
