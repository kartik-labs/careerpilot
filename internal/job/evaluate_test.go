package job

import (
	"context"
	"testing"

	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

func TestEvaluator_Evaluate_ParsesJSON(t *testing.T) {
	provider := &modelprovider.ClaudeProvider{GenerateFunc: func(ctx context.Context, req modelprovider.Request) (modelprovider.Response, error) {
		return modelprovider.Response{Text: `{
			"score": 87,
			"recommendation": "apply",
			"reasons": ["Strong Go backend alignment", "Microservices experience matches"],
			"concerns": ["Production Kubernetes ownership is limited"],
			"matched_skills": ["Go", "GraphQL"],
			"missing_skills": ["Kubernetes"],
			"confidence": "high"
		}`}, nil
	}}

	evaluator := NewEvaluator(newTestRecorder(), provider)
	out, err := evaluator.Evaluate(context.Background(), "job-1", "prompt")
	if err != nil {
		t.Fatalf("Evaluate() error: %v", err)
	}
	if out.Score != 87 || out.Confidence != "high" {
		t.Errorf("out = %+v, unexpected fields", out)
	}
}

func TestApplyEvaluation_ExampleFromSpec(t *testing.T) {
	out := EvaluationOutput{
		Score:          87,
		Recommendation: "apply",
		Reasons: []string{
			"Strong Go backend alignment",
			"Microservices experience matches",
			"GraphQL experience matches",
			"Seniority appropriate",
		},
		Concerns:   []string{"Production Kubernetes ownership is limited"},
		Confidence: "high",
	}

	score := ApplyEvaluation(out)

	if score.Value != 87 {
		t.Errorf("Value = %d, want 87", score.Value)
	}
	if score.Recommendation != RecommendationApply {
		t.Errorf("Recommendation = %s, want APPLY", score.Recommendation)
	}
	if len(score.Reasons) != 4 {
		t.Errorf("Reasons = %v, want 4 entries", score.Reasons)
	}
	if len(score.Concerns) != 1 {
		t.Errorf("Concerns = %v, want 1 entry", score.Concerns)
	}
	if score.Confidence != "high" {
		t.Errorf("Confidence = %s, want high", score.Confidence)
	}
	if score.EvaluatedBy != "deep_evaluation" {
		t.Errorf("EvaluatedBy = %s, want deep_evaluation", score.EvaluatedBy)
	}
}

func TestApplyEvaluation_UnrecognizedRecommendationDefaultsToReview(t *testing.T) {
	score := ApplyEvaluation(EvaluationOutput{Score: 50, Recommendation: "unclear"})
	if score.Recommendation != RecommendationReview {
		t.Errorf("Recommendation = %s, want REVIEW", score.Recommendation)
	}
	if len(score.Reasons) == 0 {
		t.Error("expected a default reason to be attached")
	}
}

func TestApplyEvaluation_InvalidConfidenceDefaultsToLow(t *testing.T) {
	score := ApplyEvaluation(EvaluationOutput{Score: 50, Recommendation: "review", Confidence: "very sure"})
	if score.Confidence != "low" {
		t.Errorf("Confidence = %s, want low for unrecognized input", score.Confidence)
	}
}
