package application

import "fmt"

// PlatformPolicy describes whether a job's source platform is known to
// permit automated submission. "unknown" must never be treated as
// permitted — CLAUDE.md "Platform compliance": "Before implementing
// browser automation for a platform, determine whether automation is
// permitted."
type PlatformPolicy string

const (
	PlatformPolicyUnknown PlatformPolicy = "unknown"
	PlatformPolicyAllowed PlatformPolicy = "allowed"
	PlatformPolicyDenied  PlatformPolicy = "denied"
)

// SafetyGateInput carries every fact the gate needs to decide READY
// eligibility (CLAUDE.md "Application Safety Gate").
type SafetyGateInput struct {
	Application Application

	JobVerified    bool
	ResumeVerified bool // true once ResumeVersionID's resume.Version.Status == APPROVED

	IsDuplicate bool

	Platform PlatformPolicy

	ApplicationsSubmittedToday int
	DailyApplicationLimit      int
}

// SafetyGateResult is the explainable outcome of evaluating the gate —
// mirroring job.Score's explainability requirement so a blocked
// application always states why.
type SafetyGateResult struct {
	Passed  bool
	Reasons []string // populated when Passed is false, one entry per failed check
}

// EvaluateSafetyGate runs every required check before an application may
// become READY (CLAUDE.md "Application Safety Gate" / "Before submission
// verify: ..."). It is pure and deterministic — no model call, no
// mutation; callers apply the result via Ready.
func EvaluateSafetyGate(in SafetyGateInput) SafetyGateResult {
	var reasons []string

	if !in.JobVerified {
		reasons = append(reasons, "job identity not verified")
	}
	if !in.ResumeVerified {
		reasons = append(reasons, "resume not verified/approved")
	}
	if HasUnresolvedReview(in.Application.Answers) {
		reasons = append(reasons, "unresolved REVIEW answers remain")
	}
	if in.IsDuplicate {
		reasons = append(reasons, "duplicate application detected for this job")
	}
	if in.Platform == PlatformPolicyUnknown {
		reasons = append(reasons, "platform automation policy is unknown")
	} else if in.Platform == PlatformPolicyDenied {
		reasons = append(reasons, "platform automation is denied")
	}
	if in.DailyApplicationLimit > 0 && in.ApplicationsSubmittedToday >= in.DailyApplicationLimit {
		reasons = append(reasons, fmt.Sprintf("daily application limit (%d) reached", in.DailyApplicationLimit))
	}

	return SafetyGateResult{
		Passed:  len(reasons) == 0,
		Reasons: reasons,
	}
}

// Ready transitions an application to READY only if the safety gate
// passed. It never allows a caller to bypass a failing gate.
func Ready(a Application, gate SafetyGateResult) (Application, error) {
	if !gate.Passed {
		return Application{}, fmt.Errorf("application: safety gate failed: %v", gate.Reasons)
	}
	a.SafetyGate = &gate
	return Transition(a, StatusReady)
}
