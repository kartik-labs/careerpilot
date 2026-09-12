package application

import (
	"testing"

	"github.com/kartik-labs/careerpilot/internal/candidate"
)

func TestClassify_AlwaysReviewCategories(t *testing.T) {
	categories := []Category{
		CategoryAuthorization,
		CategorySponsorship,
		CategoryRelocation,
		CategoryCompensation,
		CategoryDemographic,
		CategoryLegal,
	}

	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: true}}

	for _, cat := range categories {
		t.Run(string(cat), func(t *testing.T) {
			a := Answer{
				Category:   cat,
				Confidence: AnswerConfidenceHigh,
				ClaimIDs:   []string{"c1"},
			}
			got := Classify(a, allowed)
			if got.Status != AnswerStatusReview {
				t.Errorf("Status = %s, want REVIEW for category %s even with high confidence", got.Status, cat)
			}
		})
	}
}

func TestClassify_AutoWhenDeterministicAndVerified(t *testing.T) {
	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: true}}
	a := Answer{
		Category:   CategoryEmployment,
		Confidence: AnswerConfidenceHigh,
		ClaimIDs:   []string{"c1"},
	}

	got := Classify(a, allowed)
	if got.Status != AnswerStatusAuto {
		t.Errorf("Status = %s, want AUTO", got.Status)
	}
	if got.Reason != "" {
		t.Errorf("Reason = %q, want empty for AUTO", got.Reason)
	}
}

func TestClassify_LowConfidenceGoesToReview(t *testing.T) {
	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: true}}
	a := Answer{
		Category:   CategoryEmployment,
		Confidence: AnswerConfidenceMedium,
		ClaimIDs:   []string{"c1"},
	}

	got := Classify(a, allowed)
	if got.Status != AnswerStatusReview {
		t.Errorf("Status = %s, want REVIEW", got.Status)
	}
}

func TestClassify_NoClaimsGoesToReview(t *testing.T) {
	a := Answer{Category: CategoryEmployment, Confidence: AnswerConfidenceHigh}
	got := Classify(a, nil)
	if got.Status != AnswerStatusReview {
		t.Errorf("Status = %s, want REVIEW", got.Status)
	}
}

func TestClassify_UnverifiedClaimGoesToReview(t *testing.T) {
	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: false}}
	a := Answer{
		Category:   CategoryEmployment,
		Confidence: AnswerConfidenceHigh,
		ClaimIDs:   []string{"c1"},
	}

	got := Classify(a, allowed)
	if got.Status != AnswerStatusReview {
		t.Errorf("Status = %s, want REVIEW for unverified claim", got.Status)
	}
}

func TestClassify_MissingClaimGoesToReview(t *testing.T) {
	allowed := []candidate.Claim{{ID: "c1", AllowedForResume: true}}
	a := Answer{
		Category:   CategoryEmployment,
		Confidence: AnswerConfidenceHigh,
		ClaimIDs:   []string{"nonexistent"},
	}

	got := Classify(a, allowed)
	if got.Status != AnswerStatusReview {
		t.Errorf("Status = %s, want REVIEW for missing claim", got.Status)
	}
}

func TestHasUnresolvedReview(t *testing.T) {
	answers := []Answer{
		{Status: AnswerStatusAuto},
		{Status: AnswerStatusReview},
	}
	if !HasUnresolvedReview(answers) {
		t.Error("expected unresolved review to be detected")
	}

	allAuto := []Answer{{Status: AnswerStatusAuto}, {Status: AnswerStatusAuto}}
	if HasUnresolvedReview(allAuto) {
		t.Error("expected no unresolved review when all answers are AUTO")
	}
}
