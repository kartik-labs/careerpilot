package candidate

import "testing"

func TestIsForbidden(t *testing.T) {
	forbidden := []ForbiddenClaim{
		{ProfileID: "p1", Text: "Led a team of 50 engineers"},
		{ProfileID: "p1", Text: "Increased revenue by 300%"},
	}

	cases := []struct {
		name string
		text string
		want bool
	}{
		{"exact match", "Led a team of 50 engineers", true},
		{"no match", "Wrote backend services in Go", false},
		{"case sensitive mismatch", "led a team of 50 engineers", false},
		{"empty forbidden list", "anything", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			list := forbidden
			if tc.name == "empty forbidden list" {
				list = nil
			}
			if got := IsForbidden(tc.text, list); got != tc.want {
				t.Errorf("IsForbidden(%q) = %v, want %v", tc.text, got, tc.want)
			}
		})
	}
}
