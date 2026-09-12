package resume

// ValidationResult is the structured outcome of validating a compiled
// resume PDF. A resume exceeding one page must fail with reasons — never
// be silently forced to fit by shrinking typography
// (docs/RESUME-MANAGEMENT.md "One-Page Requirement").
type ValidationResult struct {
	Passed    bool
	PageCount int
	Reasons   []string
}

const maxPageCount = 1

// ValidatePageCount enforces the one-page requirement. It never modifies
// the document; callers wanting to attempt content reduction must do so
// before re-invoking this validator on a newly compiled PDF, and any such
// reduction must be a content/spacing change, never a typography shrink
// below the template's established minimum (see docs/RESUME-MANAGEMENT.md).
func ValidatePageCount(pageCount int) ValidationResult {
	if pageCount <= 0 {
		return ValidationResult{
			Passed:    false,
			PageCount: pageCount,
			Reasons:   []string{"page count could not be determined"},
		}
	}

	if pageCount > maxPageCount {
		return ValidationResult{
			Passed:    false,
			PageCount: pageCount,
			Reasons: []string{
				"resume exceeds one page",
				"reduce content (remove low-value bullets, shorten wording, tighten spacing) and recompile",
				"do not reduce font size below the template's established minimum",
			},
		}
	}

	return ValidationResult{
		Passed:    true,
		PageCount: pageCount,
	}
}
