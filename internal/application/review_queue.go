package application

// ReviewItem is one unit of work a human needs to resolve, surfaced from
// an application's unresolved answers or its own REVIEW status.
type ReviewItem struct {
	ApplicationID string
	JobID         string
	AnswerID      string // empty when the item is the application itself, not a specific answer
	Reason        string
}

// BuildReviewQueue collects every outstanding REVIEW item across the given
// applications: answers needing review, plus applications sitting in
// StatusReview with no other reason recorded (e.g. awaiting resume
// approval). Deterministic and read-only — no model call involved.
func BuildReviewQueue(apps []Application) []ReviewItem {
	var items []ReviewItem

	for _, a := range apps {
		foundAnswerReview := false
		for _, ans := range a.Answers {
			if ans.Status == AnswerStatusReview {
				items = append(items, ReviewItem{
					ApplicationID: a.ID,
					JobID:         a.JobID,
					AnswerID:      ans.ID,
					Reason:        ans.Reason,
				})
				foundAnswerReview = true
			}
		}

		if a.Status == StatusReview && !foundAnswerReview {
			items = append(items, ReviewItem{
				ApplicationID: a.ID,
				JobID:         a.JobID,
				Reason:        "application awaiting review (e.g. resume approval)",
			})
		}
	}

	return items
}
