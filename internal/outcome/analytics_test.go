package outcome

import (
	"testing"
	"time"
)

func rec(role, company, source, variant string, version, score int, stages ...FunnelStage) Record {
	r := Record{Role: role, Company: company, Source: source, ResumeVariant: variant, ResumeVersion: version, JobScore: score}
	for _, s := range stages {
		r = AppendStage(r, s, time.Unix(0, 0))
	}
	return r
}

func TestFunnel_CountsEachStageIndependently(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 80, StageDiscovered, StageQualified, StageSubmitted, StageResponded),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 60, StageDiscovered, StageQualified, StageSubmitted),
		rec("FDE", "Gamma", "greenhouse", "fde", 2, 90, StageDiscovered),
	}

	f := Funnel(records)
	if f.Discovered != 3 {
		t.Errorf("Discovered = %d, want 3", f.Discovered)
	}
	if f.Submitted != 2 {
		t.Errorf("Submitted = %d, want 2", f.Submitted)
	}
	if f.Responded != 1 {
		t.Errorf("Responded = %d, want 1", f.Responded)
	}
}

func TestResponseRate(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 80, StageSubmitted, StageResponded),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 60, StageSubmitted),
	}

	rate := ResponseRate(records)
	if rate.Numerator != 1 || rate.Denominator != 2 {
		t.Errorf("rate = %+v, want 1/2", rate)
	}
	if rate.Value != 0.5 {
		t.Errorf("Value = %f, want 0.5", rate.Value)
	}
}

func TestResponseRate_NoSubmissionsIsZeroNotNaN(t *testing.T) {
	rate := ResponseRate(nil)
	if rate.Value != 0 {
		t.Errorf("Value = %f, want 0", rate.Value)
	}
	if rate.Denominator != 0 {
		t.Errorf("Denominator = %d, want 0", rate.Denominator)
	}
}

func TestInterviewRate(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 80, StageSubmitted, StageInterview),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 60, StageSubmitted),
	}

	rate := InterviewRate(records)
	if rate.Numerator != 1 || rate.Denominator != 2 {
		t.Errorf("rate = %+v, want 1/2", rate)
	}
}

func TestByResumeVersion_GroupsCorrectly(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 80, StageSubmitted, StageResponded),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 60, StageSubmitted),
		rec("FDE", "Gamma", "greenhouse", "fde", 2, 90, StageSubmitted, StageOffer),
	}

	perf := ByResumeVersion(records)
	if len(perf) != 2 {
		t.Fatalf("perf = %d groups, want 2", len(perf))
	}

	// Sorted alphabetically: "fde" before "software-engineer".
	if perf[0].Variant != "fde" || perf[0].Version != 2 {
		t.Errorf("perf[0] = %+v, want fde/v2", perf[0])
	}
	if perf[0].OfferCount != 1 {
		t.Errorf("perf[0].OfferCount = %d, want 1", perf[0].OfferCount)
	}
	if perf[1].Variant != "software-engineer" {
		t.Errorf("perf[1].Variant = %q, want software-engineer", perf[1].Variant)
	}
	if perf[1].Submitted != 2 {
		t.Errorf("perf[1].Submitted = %d, want 2", perf[1].Submitted)
	}
}

func TestScoreCalibration_BucketsByDecile(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 85, StageSubmitted, StageResponded),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 25, StageSubmitted),
		rec("SWE", "Gamma", "greenhouse", "software-engineer", 1, 100, StageSubmitted, StageOffer),
	}

	buckets := ScoreCalibration(records)
	if len(buckets) != 10 {
		t.Fatalf("buckets = %d, want 10", len(buckets))
	}

	// score 85 -> bucket 8 (80-89)
	if buckets[8].Count != 1 {
		t.Errorf("buckets[8].Count = %d, want 1", buckets[8].Count)
	}
	// score 25 -> bucket 2 (20-29)
	if buckets[2].Count != 1 {
		t.Errorf("buckets[2].Count = %d, want 1", buckets[2].Count)
	}
	// score 100 -> clamped into bucket 9 (90-100)
	if buckets[9].Count != 1 {
		t.Errorf("buckets[9].Count = %d, want 1 (score 100 clamped into last bucket)", buckets[9].Count)
	}
	if buckets[9].OfferCount != 1 {
		t.Errorf("buckets[9].OfferCount = %d, want 1", buckets[9].OfferCount)
	}
}

func TestBySource(t *testing.T) {
	records := []Record{
		rec("SWE", "Acme", "greenhouse", "software-engineer", 1, 80, StageSubmitted, StageResponded),
		rec("SWE", "Beta", "linkedin", "software-engineer", 1, 60, StageSubmitted),
		rec("SWE", "Gamma", "greenhouse", "software-engineer", 1, 70, StageSubmitted),
	}

	perf := BySource(records)
	if len(perf) != 2 {
		t.Fatalf("perf = %d, want 2 sources", len(perf))
	}
	if perf[0].Source != "greenhouse" || perf[0].Submitted != 2 {
		t.Errorf("perf[0] = %+v, want greenhouse/2", perf[0])
	}
	if perf[1].Source != "linkedin" || perf[1].Submitted != 1 {
		t.Errorf("perf[1] = %+v, want linkedin/1", perf[1])
	}
}

func TestByRole(t *testing.T) {
	records := []Record{
		rec("Backend Engineer", "Acme", "greenhouse", "software-engineer", 1, 80, StageSubmitted),
		rec("Forward Deployed Engineer", "Beta", "linkedin", "fde", 1, 60, StageSubmitted, StageInterview),
	}

	perf := ByRole(records)
	if len(perf) != 2 {
		t.Fatalf("perf = %d, want 2 roles", len(perf))
	}
	if perf[0].Role != "Backend Engineer" {
		t.Errorf("perf[0].Role = %q, want Backend Engineer", perf[0].Role)
	}
	if perf[1].InterviewRate.Numerator != 1 {
		t.Errorf("perf[1].InterviewRate.Numerator = %d, want 1", perf[1].InterviewRate.Numerator)
	}
}

func TestRecommendLowPerformingResumeVersions(t *testing.T) {
	records := []Record{
		// High performer: 3/3 responded.
		rec("SWE", "A", "gh", "software-engineer", 2, 80, StageSubmitted, StageResponded),
		rec("SWE", "B", "gh", "software-engineer", 2, 80, StageSubmitted, StageResponded),
		rec("SWE", "C", "gh", "software-engineer", 2, 80, StageSubmitted, StageResponded),
		// Low performer: 0/3 responded.
		rec("SWE", "D", "gh", "software-engineer", 1, 80, StageSubmitted),
		rec("SWE", "E", "gh", "software-engineer", 1, 80, StageSubmitted),
		rec("SWE", "F", "gh", "software-engineer", 1, 80, StageSubmitted),
	}

	recs := RecommendLowPerformingResumeVersions(records, 2)
	if len(recs) != 1 {
		t.Fatalf("recommendations = %d, want 1", len(recs))
	}
	if recs[0].Subject != "resume:software-engineer:v1" {
		t.Errorf("Subject = %q, want resume:software-engineer:v1", recs[0].Subject)
	}
}

func TestRecommendLowPerformingResumeVersions_RespectsMinSubmissions(t *testing.T) {
	records := []Record{
		rec("SWE", "A", "gh", "software-engineer", 2, 80, StageSubmitted, StageResponded),
		rec("SWE", "D", "gh", "software-engineer", 1, 80, StageSubmitted), // only 1 submission, below threshold
	}

	recs := RecommendLowPerformingResumeVersions(records, 2)
	if len(recs) != 0 {
		t.Errorf("recommendations = %d, want 0 (insufficient sample size)", len(recs))
	}
}

func TestRecommendation_NeverMutatesInput(t *testing.T) {
	records := []Record{
		rec("SWE", "A", "gh", "software-engineer", 1, 80, StageSubmitted),
	}
	before := len(records[0].Stages)

	RecommendLowPerformingResumeVersions(records, 1)

	if len(records[0].Stages) != before {
		t.Error("expected recommendation computation to never mutate input records")
	}
}
