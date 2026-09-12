package outcome

import "sort"

// FunnelCounts is the application funnel: how many records reached each
// stage at least once. Counts are cumulative-by-stage, not mutually
// exclusive — a record that reached Interview also counted at Submitted.
type FunnelCounts struct {
	Discovered int
	Qualified  int
	Prepared   int
	Submitted  int
	Responded  int
	Interview  int
	Offer      int
	Rejected   int
	Withdrawn  int
}

// Funnel computes the application funnel over records
// (Phase 6 prompt: "1. Application funnel.").
func Funnel(records []Record) FunnelCounts {
	var f FunnelCounts
	for _, r := range records {
		if r.HasStage(StageDiscovered) {
			f.Discovered++
		}
		if r.HasStage(StageQualified) {
			f.Qualified++
		}
		if r.HasStage(StagePrepared) {
			f.Prepared++
		}
		if r.HasStage(StageSubmitted) {
			f.Submitted++
		}
		if r.HasStage(StageResponded) {
			f.Responded++
		}
		if r.HasStage(StageInterview) {
			f.Interview++
		}
		if r.HasStage(StageOffer) {
			f.Offer++
		}
		if r.HasStage(StageRejected) {
			f.Rejected++
		}
		if r.HasStage(StageWithdrawn) {
			f.Withdrawn++
		}
	}
	return f
}

// Rate is a ratio expressed as both a fraction and the counts behind it,
// so callers/UIs never have to re-derive the denominator.
type Rate struct {
	Numerator   int
	Denominator int
	Value       float64 // 0 when Denominator is 0, never NaN/Inf
}

func computeRate(numerator, denominator int) Rate {
	if denominator == 0 {
		return Rate{Numerator: numerator, Denominator: 0, Value: 0}
	}
	return Rate{Numerator: numerator, Denominator: denominator, Value: float64(numerator) / float64(denominator)}
}

// ResponseRate is recruiter responses as a fraction of submitted
// applications (Phase 6 prompt: "2. Response rate.").
func ResponseRate(records []Record) Rate {
	f := Funnel(records)
	return computeRate(f.Responded, f.Submitted)
}

// InterviewRate is interviews as a fraction of submitted applications
// (Phase 6 prompt: "3. Interview rate.").
func InterviewRate(records []Record) Rate {
	f := Funnel(records)
	return computeRate(f.Interview, f.Submitted)
}

// ResumeVersionPerformance aggregates outcomes per resume
// variant+version (Phase 6 prompt: "5. Resume-version performance.").
type ResumeVersionPerformance struct {
	Variant       string
	Version       int
	Submitted     int
	ResponseRate  Rate
	InterviewRate Rate
	OfferCount    int
}

// ByResumeVersion groups records by (ResumeVariant, ResumeVersion) and
// computes performance for each group. Results are sorted by
// variant then version for stable output.
func ByResumeVersion(records []Record) []ResumeVersionPerformance {
	type key struct {
		variant string
		version int
	}
	groups := map[key][]Record{}
	for _, r := range records {
		k := key{r.ResumeVariant, r.ResumeVersion}
		groups[k] = append(groups[k], r)
	}

	var out []ResumeVersionPerformance
	for k, group := range groups {
		f := Funnel(group)
		out = append(out, ResumeVersionPerformance{
			Variant:       k.variant,
			Version:       k.version,
			Submitted:     f.Submitted,
			ResponseRate:  computeRate(f.Responded, f.Submitted),
			InterviewRate: computeRate(f.Interview, f.Submitted),
			OfferCount:    f.Offer,
		})
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Variant != out[j].Variant {
			return out[i].Variant < out[j].Variant
		}
		return out[i].Version < out[j].Version
	})

	return out
}

// ScoreBucket is a job-score range used for calibration analysis.
type ScoreBucket struct {
	MinScore, MaxScore int // inclusive
	Count              int
	ResponseRate       Rate
	InterviewRate      Rate
	OfferCount         int
}

// defaultScoreBuckets splits the 0-100 score range into deciles.
func defaultScoreBuckets() []ScoreBucket {
	buckets := make([]ScoreBucket, 10)
	for i := range buckets {
		buckets[i] = ScoreBucket{MinScore: i * 10, MaxScore: i*10 + 9}
	}
	buckets[9].MaxScore = 100
	return buckets
}

// ScoreCalibration buckets records by job score decile and reports actual
// outcome rates per bucket — this is what lets a human see whether a
// score of "87" actually correlates with better outcomes than "60"
// (Phase 6 prompt: "6. Job-score calibration."). It never adjusts scoring
// itself; that would require the deliberate review step Phase 6 mandates.
func ScoreCalibration(records []Record) []ScoreBucket {
	buckets := defaultScoreBuckets()
	grouped := make([][]Record, len(buckets))

	for _, r := range records {
		idx := r.JobScore / 10
		if idx > 9 {
			idx = 9
		}
		if idx < 0 {
			idx = 0
		}
		grouped[idx] = append(grouped[idx], r)
	}

	for i, group := range grouped {
		f := Funnel(group)
		buckets[i].Count = len(group)
		buckets[i].ResponseRate = computeRate(f.Responded, f.Submitted)
		buckets[i].InterviewRate = computeRate(f.Interview, f.Submitted)
		buckets[i].OfferCount = f.Offer
	}

	return buckets
}

// SourcePerformance aggregates outcomes per job source
// (Phase 6 prompt: "7. Source performance.").
type SourcePerformance struct {
	Source        string
	Submitted     int
	ResponseRate  Rate
	InterviewRate Rate
	OfferCount    int
}

// BySource groups records by Source and computes performance for each,
// sorted alphabetically by source name.
func BySource(records []Record) []SourcePerformance {
	groups := map[string][]Record{}
	for _, r := range records {
		groups[r.Source] = append(groups[r.Source], r)
	}

	var out []SourcePerformance
	for source, group := range groups {
		f := Funnel(group)
		out = append(out, SourcePerformance{
			Source:        source,
			Submitted:     f.Submitted,
			ResponseRate:  computeRate(f.Responded, f.Submitted),
			InterviewRate: computeRate(f.Interview, f.Submitted),
			OfferCount:    f.Offer,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out
}

// RolePerformance aggregates outcomes per role
// (Phase 6 prompt: "7. Role performance." — listed alongside source
// performance in the spec).
type RolePerformance struct {
	Role          string
	Submitted     int
	ResponseRate  Rate
	InterviewRate Rate
	OfferCount    int
}

// ByRole groups records by Role and computes performance for each,
// sorted alphabetically by role name.
func ByRole(records []Record) []RolePerformance {
	groups := map[string][]Record{}
	for _, r := range records {
		groups[r.Role] = append(groups[r.Role], r)
	}

	var out []RolePerformance
	for role, group := range groups {
		f := Funnel(group)
		out = append(out, RolePerformance{
			Role:          role,
			Submitted:     f.Submitted,
			ResponseRate:  computeRate(f.Responded, f.Submitted),
			InterviewRate: computeRate(f.Interview, f.Submitted),
			OfferCount:    f.Offer,
		})
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Role < out[j].Role })
	return out
}
