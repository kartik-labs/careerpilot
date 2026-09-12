package resume

import "testing"

func TestValidTransition(t *testing.T) {
	cases := []struct {
		from, to Status
		want     bool
	}{
		{StatusDraft, StatusGenerated, true},
		{StatusGenerated, StatusValidated, true},
		{StatusGenerated, StatusReview, true},
		{StatusValidated, StatusApproved, true},
		{StatusReview, StatusApproved, true},
		{StatusReview, StatusRejected, true},
		{StatusReview, StatusGenerated, true},
		{StatusDraft, StatusApproved, false},
		{StatusApproved, StatusDraft, false},
		{StatusRejected, StatusApproved, false},
		{StatusApproved, StatusRejected, false},
	}

	for _, tc := range cases {
		if got := ValidTransition(tc.from, tc.to); got != tc.want {
			t.Errorf("ValidTransition(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}
