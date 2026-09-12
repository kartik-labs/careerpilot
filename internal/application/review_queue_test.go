package application

import "testing"

func TestBuildReviewQueue_CollectsAnswerReviews(t *testing.T) {
	apps := []Application{
		{
			ID:    "app-1",
			JobID: "job-1",
			Answers: []Answer{
				{ID: "ans-1", Status: AnswerStatusReview, Reason: "confidence is not high"},
				{ID: "ans-2", Status: AnswerStatusAuto},
			},
		},
	}

	items := BuildReviewQueue(apps)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].AnswerID != "ans-1" {
		t.Errorf("AnswerID = %q, want ans-1", items[0].AnswerID)
	}
}

func TestBuildReviewQueue_ApplicationLevelReview(t *testing.T) {
	apps := []Application{
		{ID: "app-1", JobID: "job-1", Status: StatusReview},
	}

	items := BuildReviewQueue(apps)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].AnswerID != "" {
		t.Errorf("AnswerID = %q, want empty for application-level review", items[0].AnswerID)
	}
}

func TestBuildReviewQueue_NoDuplicateWhenAnswerAndAppBothReview(t *testing.T) {
	apps := []Application{
		{
			ID:     "app-1",
			JobID:  "job-1",
			Status: StatusReview,
			Answers: []Answer{
				{ID: "ans-1", Status: AnswerStatusReview, Reason: "x"},
			},
		},
	}

	items := BuildReviewQueue(apps)
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1 (answer review only, no duplicate app-level entry)", len(items))
	}
}

func TestBuildReviewQueue_EmptyWhenNothingPending(t *testing.T) {
	apps := []Application{
		{ID: "app-1", Status: StatusReady, Answers: []Answer{{Status: AnswerStatusAuto}}},
	}
	if items := BuildReviewQueue(apps); len(items) != 0 {
		t.Errorf("items = %v, want empty", items)
	}
}
