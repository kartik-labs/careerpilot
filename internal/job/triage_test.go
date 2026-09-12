package job

import (
	"context"
	"testing"

	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

type fakeRunStore struct{}

func (fakeRunStore) SaveRun(run modelprovider.Run) error { return nil }

func newTestRecorder() *modelprovider.Recorder {
	i := 0
	return modelprovider.NewRecorder(fakeRunStore{}, func() string {
		i++
		return "run"
	})
}

func TestTriager_Triage_ParsesJSON(t *testing.T) {
	provider := &modelprovider.GeminiProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		return modelprovider.Response{Text: `{
			"role_type": "backend",
			"fit_score": 85,
			"hard_fail": false,
			"matched_skills": ["Go"],
			"missing_skills": ["Kubernetes"],
			"risk_flags": [],
			"recommended_resume": "software-engineer",
			"recommendation": "apply"
		}`}, nil
	}}

	triager := NewTriager(newTestRecorder(), provider)
	out, err := triager.Triage(context.Background(), "job-1", "prompt text")
	if err != nil {
		t.Fatalf("Triage() error: %v", err)
	}
	if out.FitScore != 85 || out.RoleType != "backend" {
		t.Errorf("out = %+v, unexpected fields", out)
	}
}

func TestTriager_Triage_InvalidJSON(t *testing.T) {
	provider := &modelprovider.GeminiProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		return modelprovider.Response{Text: "not json"}, nil
	}}

	triager := NewTriager(newTestRecorder(), provider)
	if _, err := triager.Triage(context.Background(), "job-1", "prompt"); err == nil {
		t.Fatal("Triage() expected error for invalid JSON, got nil")
	}
}

func TestApplyTriage_HardFail(t *testing.T) {
	score := ApplyTriage(TriageOutput{FitScore: 90, HardFail: true})
	if score.Recommendation != RecommendationReject {
		t.Errorf("Recommendation = %s, want REJECT", score.Recommendation)
	}
}

func TestApplyTriage_Apply(t *testing.T) {
	score := ApplyTriage(TriageOutput{FitScore: 80, Recommendation: "apply"})
	if score.Recommendation != RecommendationApply {
		t.Errorf("Recommendation = %s, want APPLY", score.Recommendation)
	}
}

func TestApplyTriage_UnrecognizedRecommendationDefaultsToReview(t *testing.T) {
	score := ApplyTriage(TriageOutput{FitScore: 50, Recommendation: "maybe"})
	if score.Recommendation != RecommendationReview {
		t.Errorf("Recommendation = %s, want REVIEW", score.Recommendation)
	}
}

func TestApplyTriage_ScoreClamped(t *testing.T) {
	score := ApplyTriage(TriageOutput{FitScore: 500, Recommendation: "apply"})
	if score.Value != 100 {
		t.Errorf("Value = %d, want clamped to 100", score.Value)
	}

	score = ApplyTriage(TriageOutput{FitScore: -20, Recommendation: "reject"})
	if score.Value != 0 {
		t.Errorf("Value = %d, want clamped to 0", score.Value)
	}
}
