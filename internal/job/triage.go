package job

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/kartik-labs/careerpilot/internal/modelprovider"
)

// TriageOutput is the structured shape the Gemini triage prompt must
// return. Parsing is strict: malformed output is a hard error, never
// silently coerced into a guessed score (docs/IMPLEMENTATION-PLAN.md
// section 2.2).
type TriageOutput struct {
	RoleType          string   `json:"role_type"`
	FitScore          int      `json:"fit_score"`
	HardFail          bool     `json:"hard_fail"`
	MatchedSkills     []string `json:"matched_skills"`
	MissingSkills     []string `json:"missing_skills"`
	RiskFlags         []string `json:"risk_flags"`
	RecommendedResume string   `json:"recommended_resume"`
	Recommendation    string   `json:"recommendation"`
}

// Triager runs the cheap, high-volume first pass over a normalized Job
// using the Gemini provider. Its output is a proposal only — the LLM never
// writes Job/Score state directly; ApplyTriage (deterministic Go) is what
// turns TriageOutput into a Score.
type Triager struct {
	Recorder *modelprovider.Recorder
	Provider modelprovider.Provider
}

// NewTriager builds a Triager. provider is expected to be a
// modelprovider.GeminiProvider (or any Provider) — CLAUDE.md routes triage
// to Gemini for cost reasons, but this stays provider-agnostic per the
// ModelProvider abstraction.
func NewTriager(recorder *modelprovider.Recorder, provider modelprovider.Provider) *Triager {
	return &Triager{Recorder: recorder, Provider: provider}
}

// Triage sends the rendered prompt to the provider and parses its
// response into a TriageOutput. prompt construction (from Job + candidate
// profile facts) is the caller's responsibility, keeping this package
// focused on orchestration, not prompt templates.
func (t *Triager) Triage(ctx context.Context, jobID, prompt string) (TriageOutput, error) {
	resp, err := t.Recorder.Generate(ctx, t.Provider, "job_triage", modelprovider.Request{
		Prompt:   prompt,
		Metadata: map[string]string{"job_id": jobID},
	})
	if err != nil {
		return TriageOutput{}, fmt.Errorf("job: triage generate: %w", err)
	}

	var out TriageOutput
	if err := json.Unmarshal([]byte(resp.Text), &out); err != nil {
		return TriageOutput{}, fmt.Errorf("job: triage response is not valid TriageOutput JSON: %w", err)
	}

	return out, nil
}

// ApplyTriage deterministically converts a TriageOutput into a Score.
// Hard filters and recommendation mapping happen here in Go, not inside
// the model call, so a malformed or adversarial model response cannot
// silently produce an APPLY recommendation.
func ApplyTriage(out TriageOutput) Score {
	score := Score{
		Value:         clampScore(out.FitScore),
		MatchedSkills: out.MatchedSkills,
		MissingSkills: out.MissingSkills,
		Concerns:      out.RiskFlags,
		EvaluatedBy:   "triage",
	}

	if out.HardFail {
		score.Recommendation = RecommendationReject
		score.Reasons = []string{"triage hard filter failed"}
		score.Confidence = "high"
		return score
	}

	switch normalizeRecommendation(out.Recommendation) {
	case RecommendationApply:
		score.Recommendation = RecommendationApply
		score.Reasons = []string{fmt.Sprintf("triage fit score %d", score.Value)}
	case RecommendationReject:
		score.Recommendation = RecommendationReject
		score.Reasons = []string{fmt.Sprintf("triage fit score %d below threshold", score.Value)}
	default:
		score.Recommendation = RecommendationReview
		score.Reasons = []string{"triage recommendation ambiguous or unrecognized"}
	}

	score.Confidence = "medium" // triage is a cheap first pass; deep evaluation refines this
	return score
}

func normalizeRecommendation(s string) Recommendation {
	switch s {
	case "apply", "APPLY":
		return RecommendationApply
	case "reject", "REJECT":
		return RecommendationReject
	case "review", "REVIEW":
		return RecommendationReview
	default:
		return ""
	}
}

func clampScore(v int) int {
	if v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
