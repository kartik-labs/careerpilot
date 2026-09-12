package job

// Recommendation is the outcome of scoring/evaluating a job.
type Recommendation string

const (
	RecommendationApply  Recommendation = "APPLY"
	RecommendationReview Recommendation = "REVIEW"
	RecommendationReject Recommendation = "REJECT"
)

// Score is the explainable output of job evaluation. Every field that
// drives a recommendation must be visible here — a score without visible
// reasons is not acceptable (CLAUDE.md/docs/IMPLEMENTATION-PLAN.md
// section 5 "The score must be explainable.").
type Score struct {
	Value          int // 0-100
	Recommendation Recommendation

	Reasons  []string
	Concerns []string

	MatchedSkills []string
	MissingSkills []string

	// Confidence reflects how certain the evaluator is in this score,
	// independent of the score value itself (e.g. a 90 with low confidence
	// should not auto-apply).
	Confidence string // "high" | "medium" | "low"

	// EvaluatedBy identifies which stage produced this score:
	// "triage" (Gemini, cheap/high-volume) or "deep_evaluation" (Claude).
	EvaluatedBy string
}
