package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

// EvaluationOutput is the structured shape the Claude deep-evaluation
// prompt must return, matching the explainable score example in
// docs/IMPLEMENTATION-PLAN.md section 5.
type EvaluationOutput struct {
	Score          int      `json:"score"`
	Recommendation string   `json:"recommendation"` // "apply" | "review" | "reject"
	Reasons        []string `json:"reasons"`
	Concerns       []string `json:"concerns"`
	MatchedSkills  []string `json:"matched_skills"`
	MissingSkills  []string `json:"missing_skills"`
	Confidence     string   `json:"confidence"` // "high" | "medium" | "low"
}

// Evaluator runs deep evaluation over triage-qualified jobs using the
// Claude provider (CLAUDE.md: "Use Claude for: deep job analysis").
type Evaluator struct {
	Recorder *modelprovider.Recorder
	Provider modelprovider.Provider
}

// NewEvaluator builds an Evaluator.
func NewEvaluator(recorder *modelprovider.Recorder, provider modelprovider.Provider) *Evaluator {
	return &Evaluator{Recorder: recorder, Provider: provider}
}

// Evaluate sends the rendered prompt to the provider and parses its
// response into an EvaluationOutput.
func (e *Evaluator) Evaluate(ctx context.Context, jobID, prompt string) (EvaluationOutput, error) {
	resp, err := e.Recorder.Generate(ctx, e.Provider, "job_deep_evaluation", modelprovider.Request{
		Prompt:   prompt,
		Metadata: map[string]string{"job_id": jobID},
	})
	if err != nil {
		return EvaluationOutput{}, fmt.Errorf("job: evaluate generate: %w", err)
	}

	var out EvaluationOutput
	if err := json.Unmarshal([]byte(resp.Text), &out); err != nil {
		return EvaluationOutput{}, fmt.Errorf("job: evaluation response is not valid EvaluationOutput JSON: %w", err)
	}

	return out, nil
}

// ApplyEvaluation deterministically converts an EvaluationOutput into a
// Score. As with triage, the recommendation mapping is Go code, not model
// output taken at face value — an unrecognized recommendation string
// always degrades to REVIEW, never APPLY.
func ApplyEvaluation(out EvaluationOutput) Score {
	score := Score{
		Value:         clampScore(out.Score),
		Reasons:       out.Reasons,
		Concerns:      out.Concerns,
		MatchedSkills: out.MatchedSkills,
		MissingSkills: out.MissingSkills,
		Confidence:    normalizeConfidence(out.Confidence),
		EvaluatedBy:   "deep_evaluation",
	}

	switch normalizeRecommendation(out.Recommendation) {
	case RecommendationApply:
		score.Recommendation = RecommendationApply
	case RecommendationReject:
		score.Recommendation = RecommendationReject
	default:
		score.Recommendation = RecommendationReview
		if len(score.Reasons) == 0 {
			score.Reasons = []string{"deep evaluation recommendation ambiguous or unrecognized"}
		}
	}

	return score
}

func normalizeConfidence(c string) string {
	switch c {
	case "high", "medium", "low":
		return c
	default:
		return "low"
	}
}
