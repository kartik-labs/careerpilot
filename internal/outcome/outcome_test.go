package outcome

import (
	"testing"
	"time"
)

func TestAppendStage_AddsToHistory(t *testing.T) {
	r := Record{ID: "rec-1"}
	now := time.Unix(1000, 0)

	r = AppendStage(r, StageDiscovered, now)
	r = AppendStage(r, StageQualified, now.Add(time.Hour))

	if len(r.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2", len(r.Stages))
	}
	if r.Stages[0].Stage != StageDiscovered || r.Stages[1].Stage != StageQualified {
		t.Errorf("Stages = %+v, unexpected order/content", r.Stages)
	}
	if r.FinalOutcome != "" {
		t.Errorf("FinalOutcome = %s, want empty (not yet terminal)", r.FinalOutcome)
	}
}

func TestAppendStage_TerminalStagesSetFinalOutcome(t *testing.T) {
	cases := []FunnelStage{StageOffer, StageRejected, StageWithdrawn}

	for _, stage := range cases {
		r := Record{}
		r = AppendStage(r, stage, time.Unix(0, 0))
		if r.FinalOutcome != stage {
			t.Errorf("FinalOutcome = %s, want %s", r.FinalOutcome, stage)
		}
	}
}

func TestAppendStage_NonTerminalDoesNotSetFinalOutcome(t *testing.T) {
	r := Record{}
	r = AppendStage(r, StageInterview, time.Unix(0, 0))
	if r.FinalOutcome != "" {
		t.Errorf("FinalOutcome = %s, want empty", r.FinalOutcome)
	}
}

func TestAppendStage_PreservesPriorHistory(t *testing.T) {
	r := Record{Stages: []StageEvent{{Stage: StageDiscovered, Timestamp: time.Unix(0, 0)}}}
	r = AppendStage(r, StageQualified, time.Unix(100, 0))

	if len(r.Stages) != 2 {
		t.Fatalf("Stages = %d, want 2 (prior history preserved)", len(r.Stages))
	}
	if r.Stages[0].Stage != StageDiscovered {
		t.Error("expected first stage to remain StageDiscovered")
	}
}

func TestHasStage(t *testing.T) {
	r := Record{Stages: []StageEvent{{Stage: StageDiscovered}, {Stage: StageQualified}}}

	if !r.HasStage(StageQualified) {
		t.Error("expected HasStage(StageQualified) = true")
	}
	if r.HasStage(StageInterview) {
		t.Error("expected HasStage(StageInterview) = false")
	}
}
