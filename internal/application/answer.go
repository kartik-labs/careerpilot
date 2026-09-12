package application

import "github.com/kartik-labs/careerpilot/internal/candidate"

// AnswerConfidence mirrors candidate.Confidence but is kept distinct: an
// answer's confidence reflects how certain we are THIS answer is correct
// for THIS question, not just how certain the underlying claim is.
type AnswerConfidence string

const (
	AnswerConfidenceHigh   AnswerConfidence = "high"
	AnswerConfidenceMedium AnswerConfidence = "medium"
	AnswerConfidenceLow    AnswerConfidence = "low"
)

// AnswerStatus is whether an answer is ready to submit or needs a human.
type AnswerStatus string

const (
	AnswerStatusAuto   AnswerStatus = "AUTO"
	AnswerStatusReview AnswerStatus = "REVIEW"
)

// Category classifies a question so sensitive categories can be routed to
// REVIEW regardless of how confident a model is
// (docs/AGENT-POLICY.md "Forbidden": "guess work authorization",
// "guess compensation expectations").
type Category string

const (
	CategoryPersonal      Category = "personal"
	CategoryEducation     Category = "education"
	CategoryEmployment    Category = "employment"
	CategoryAuthorization Category = "work_authorization"
	CategorySponsorship   Category = "sponsorship"
	CategoryRelocation    Category = "relocation"
	CategoryCompensation  Category = "compensation"
	CategoryDemographic   Category = "demographic"
	CategoryLegal         Category = "legal_declaration"
	CategoryExperience    Category = "experience"
	CategoryPreferences   Category = "preferences"
	CategoryOther         Category = "other"
)

// alwaysReview lists categories that must never be auto-answered, no
// matter how confident the source data is — these require a human's
// explicit, current decision every time (docs/AGENT-POLICY.md, and
// CLAUDE.md non-negotiable principle 1: never fabricate/guess these).
var alwaysReview = map[Category]bool{
	CategoryAuthorization: true,
	CategorySponsorship:   true,
	CategoryRelocation:    true,
	CategoryCompensation:  true,
	CategoryDemographic:   true,
	CategoryLegal:         true,
}

// Answer is a single application question/response pair. Text must trace
// to ClaimIDs when SourceClaim is set — an answer with no claim backing
// and no explicit human override is not permitted to reach AUTO status.
type Answer struct {
	ID           string
	QuestionText string
	Category     Category

	AnswerText string
	ClaimIDs   []string

	Confidence AnswerConfidence
	Status     AnswerStatus
	Reason     string // populated when Status == REVIEW, explaining why
}

// Classify decides AUTO vs REVIEW for a candidate answer. An answer may
// only be AUTO when all of:
//   - its category is not in the always-review list
//   - confidence is high
//   - every referenced claim exists and is AllowedForResume
//
// Any failure produces REVIEW with a specific reason — never a silent
// downgrade to a guessed answer (CLAUDE.md "Never guess").
func Classify(a Answer, allowedClaims []candidate.Claim) Answer {
	if alwaysReview[a.Category] {
		a.Status = AnswerStatusReview
		a.Reason = "category " + string(a.Category) + " always requires human review"
		return a
	}

	if a.Confidence != AnswerConfidenceHigh {
		a.Status = AnswerStatusReview
		a.Reason = "confidence is not high"
		return a
	}

	if len(a.ClaimIDs) == 0 {
		a.Status = AnswerStatusReview
		a.Reason = "answer has no supporting claim"
		return a
	}

	allowedSet := make(map[string]bool, len(allowedClaims))
	for _, c := range allowedClaims {
		if c.AllowedForResume {
			allowedSet[c.ID] = true
		}
	}
	for _, id := range a.ClaimIDs {
		if !allowedSet[id] {
			a.Status = AnswerStatusReview
			a.Reason = "claim " + id + " is missing or not allowed"
			return a
		}
	}

	a.Status = AnswerStatusAuto
	a.Reason = ""
	return a
}

// HasUnresolvedReview reports whether any answer in the slice still needs
// human review — used by the safety gate before READY.
func HasUnresolvedReview(answers []Answer) bool {
	for _, a := range answers {
		if a.Status == AnswerStatusReview {
			return true
		}
	}
	return false
}
