package application

import (
	"testing"
	"time"

	"github.com/kartik-labs/careerpilot/internal/job"
	"github.com/kartik-labs/careerpilot/internal/resume"
)

func fixedClock() (func() string, func() time.Time) {
	return func() string { return "app-1" }, func() time.Time { return time.Unix(0, 0) }
}

func TestFromJob(t *testing.T) {
	newID, now := fixedClock()
	j := job.Job{ID: "job-1"}

	a := FromJob(newID, now, j, "profile-1")

	if a.ID != "app-1" || a.JobID != "job-1" || a.ProfileID != "profile-1" {
		t.Errorf("a = %+v, unexpected fields", a)
	}
	if a.Status != StatusDiscovered {
		t.Errorf("Status = %s, want DISCOVERED", a.Status)
	}
}

func TestQualify_ApplyRecommendation(t *testing.T) {
	a := Application{Status: StatusDiscovered}
	score := job.Score{Recommendation: job.RecommendationApply}

	got, err := Qualify(a, score)
	if err != nil {
		t.Fatalf("Qualify() error: %v", err)
	}
	if got.Status != StatusQualified {
		t.Errorf("Status = %s, want QUALIFIED", got.Status)
	}
}

func TestQualify_RejectsNonApplyRecommendation(t *testing.T) {
	a := Application{Status: StatusDiscovered}

	for _, rec := range []job.Recommendation{job.RecommendationReview, job.RecommendationReject} {
		if _, err := Qualify(a, job.Score{Recommendation: rec}); err == nil {
			t.Errorf("Qualify() with recommendation %s expected error, got nil", rec)
		}
	}
}

func TestSelectResumeVariant(t *testing.T) {
	if got := SelectResumeVariant("fde"); got != resume.VariantFDE {
		t.Errorf("SelectResumeVariant(fde) = %s, want fde", got)
	}
	if got := SelectResumeVariant("software-engineer"); got != resume.VariantSoftwareEngineer {
		t.Errorf("SelectResumeVariant(software-engineer) = %s, want software-engineer", got)
	}
	if got := SelectResumeVariant("unrecognized"); got != resume.VariantSoftwareEngineer {
		t.Errorf("SelectResumeVariant(unrecognized) = %s, want default software-engineer", got)
	}
}

func TestStartPreparing(t *testing.T) {
	a := Application{Status: StatusQualified}
	got, err := StartPreparing(a)
	if err != nil {
		t.Fatalf("StartPreparing() error: %v", err)
	}
	if got.Status != StatusPreparing {
		t.Errorf("Status = %s, want PREPARING", got.Status)
	}
}

func TestAttachResume_RequiresValidatedOrApproved(t *testing.T) {
	a := Application{Status: StatusPreparing}

	_, err := AttachResume(a, resume.Version{ID: "v1", Status: resume.StatusDraft})
	if err == nil {
		t.Fatal("AttachResume() expected error for DRAFT resume, got nil")
	}

	got, err := AttachResume(a, resume.Version{ID: "v1", Status: resume.StatusValidated})
	if err != nil {
		t.Fatalf("AttachResume() error: %v", err)
	}
	if got.ResumeVersionID != "v1" {
		t.Errorf("ResumeVersionID = %q, want v1", got.ResumeVersionID)
	}
	if got.Status != StatusReview {
		t.Errorf("Status = %s, want REVIEW", got.Status)
	}
}
