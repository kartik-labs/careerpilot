package application

import "testing"

func passingGateInput() SafetyGateInput {
	return SafetyGateInput{
		Application:                Application{},
		JobVerified:                true,
		ResumeVerified:             true,
		IsDuplicate:                false,
		Platform:                   PlatformPolicyAllowed,
		ApplicationsSubmittedToday: 2,
		DailyApplicationLimit:      15,
	}
}

func TestEvaluateSafetyGate_AllPass(t *testing.T) {
	result := EvaluateSafetyGate(passingGateInput())
	if !result.Passed {
		t.Errorf("Passed = false, reasons = %v, want true", result.Reasons)
	}
}

func TestEvaluateSafetyGate_JobNotVerified(t *testing.T) {
	in := passingGateInput()
	in.JobVerified = false
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail when job not verified")
	}
}

func TestEvaluateSafetyGate_ResumeNotVerified(t *testing.T) {
	in := passingGateInput()
	in.ResumeVerified = false
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail when resume not verified")
	}
}

func TestEvaluateSafetyGate_UnresolvedReview(t *testing.T) {
	in := passingGateInput()
	in.Application.Answers = []Answer{{Status: AnswerStatusReview}}
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail with unresolved review answers")
	}
}

func TestEvaluateSafetyGate_Duplicate(t *testing.T) {
	in := passingGateInput()
	in.IsDuplicate = true
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail for duplicate application")
	}
}

func TestEvaluateSafetyGate_PlatformUnknown(t *testing.T) {
	in := passingGateInput()
	in.Platform = PlatformPolicyUnknown
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail when platform policy is unknown")
	}
}

func TestEvaluateSafetyGate_PlatformDenied(t *testing.T) {
	in := passingGateInput()
	in.Platform = PlatformPolicyDenied
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail when platform automation is denied")
	}
}

func TestEvaluateSafetyGate_DailyLimitReached(t *testing.T) {
	in := passingGateInput()
	in.ApplicationsSubmittedToday = 15
	in.DailyApplicationLimit = 15
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Error("expected gate to fail when daily limit reached")
	}
}

func TestEvaluateSafetyGate_MultipleFailureReasons(t *testing.T) {
	in := SafetyGateInput{Platform: PlatformPolicyUnknown}
	result := EvaluateSafetyGate(in)
	if result.Passed {
		t.Fatal("expected gate to fail")
	}
	if len(result.Reasons) < 3 {
		t.Errorf("Reasons = %v, expected multiple failure reasons listed", result.Reasons)
	}
}

func TestReady_PassingGate(t *testing.T) {
	a := Application{Status: StatusReview}
	gate := EvaluateSafetyGate(passingGateInput())

	got, err := Ready(a, gate)
	if err != nil {
		t.Fatalf("Ready() error: %v", err)
	}
	if got.Status != StatusReady {
		t.Errorf("Status = %s, want READY", got.Status)
	}
	if got.SafetyGate == nil || !got.SafetyGate.Passed {
		t.Error("expected SafetyGate to be recorded and passed")
	}
}

func TestReady_FailingGateBlocksTransition(t *testing.T) {
	a := Application{Status: StatusReview}
	gate := SafetyGateResult{Passed: false, Reasons: []string{"job identity not verified"}}

	_, err := Ready(a, gate)
	if err == nil {
		t.Fatal("Ready() expected error when gate fails, got nil")
	}
}
