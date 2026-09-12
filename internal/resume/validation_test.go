package resume

import "testing"

func TestValidatePageCount(t *testing.T) {
	cases := []struct {
		name      string
		pageCount int
		wantPass  bool
	}{
		{"one page passes", 1, true},
		{"two pages fails", 2, false},
		{"zero pages fails", 0, false},
		{"negative pages fails", -1, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := ValidatePageCount(tc.pageCount)
			if result.Passed != tc.wantPass {
				t.Errorf("Passed = %v, want %v", result.Passed, tc.wantPass)
			}
			if !tc.wantPass && len(result.Reasons) == 0 {
				t.Error("expected structured reasons on failure, got none")
			}
		})
	}
}

func TestValidatePageCount_NeverSuggestsFontShrink(t *testing.T) {
	result := ValidatePageCount(2)
	for _, r := range result.Reasons {
		if r == "shrink font" || r == "reduce font size" {
			t.Errorf("validator must never suggest shrinking font, got reason: %q", r)
		}
	}
}
