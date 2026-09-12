package outcome

import "strconv"

// Recommendation is an analytics-derived suggestion for a human to
// consider. It is never applied automatically — Phase 6 prompt: "Analytics
// may recommend changes. Changes to the candidate profile or resume
// require review." Nothing in this package writes to
// internal/candidate.Profile or internal/resume.Version; producing a
// Recommendation is the full extent of this package's influence on
// those systems.
type Recommendation struct {
	Subject        string // e.g. "resume:software-engineer:v12", "source:linkedin"
	Finding        string // what the data shows
	Suggestion     string
	SupportingRate Rate
}

// RecommendLowPerformingResumeVersions flags resume versions with a
// response rate meaningfully below the overall average, provided they
// have enough submissions to be statistically meaningful. minSubmissions
// guards against flagging a version that just had bad luck on 1-2
// applications.
func RecommendLowPerformingResumeVersions(records []Record, minSubmissions int) []Recommendation {
	overall := ResponseRate(records)
	perVersion := ByResumeVersion(records)

	var out []Recommendation
	for _, v := range perVersion {
		if v.Submitted < minSubmissions {
			continue
		}
		if v.ResponseRate.Value < overall.Value {
			out = append(out, Recommendation{
				Subject:        "resume:" + v.Variant + ":v" + strconv.Itoa(v.Version),
				Finding:        "response rate below overall average",
				Suggestion:     "review this resume version's content for this role/variant before continuing to use it as-is",
				SupportingRate: v.ResponseRate,
			})
		}
	}
	return out
}
