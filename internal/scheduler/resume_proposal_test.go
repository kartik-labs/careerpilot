package scheduler

import (
	"context"
	"testing"
)

type fakeChangeDetector struct {
	changes []DetectedChange
	err     error
}

func (d *fakeChangeDetector) DetectChanges(ctx context.Context, profileID string) ([]DetectedChange, error) {
	return d.changes, d.err
}

type fakeProposalStore struct {
	saved []ResumeProposal
}

func (s *fakeProposalStore) SaveProposal(p ResumeProposal) error {
	s.saved = append(s.saved, p)
	return nil
}

func TestResumeProposalTask_Run_NoChanges(t *testing.T) {
	detector := &fakeChangeDetector{}
	store := &fakeProposalStore{}
	task := &ResumeProposalTask{Detector: detector, Store: store, NewID: idGen("prop-"), Now: testClock()}

	proposal, err := task.Run(context.Background(), "profile-1", "software-engineer")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if proposal != nil {
		t.Errorf("proposal = %+v, want nil when no changes detected", proposal)
	}
	if len(store.saved) != 0 {
		t.Error("expected no proposal saved")
	}
}

func TestResumeProposalTask_Run_WithChanges(t *testing.T) {
	detector := &fakeChangeDetector{changes: []DetectedChange{
		{Description: "Added new project", Section: "projects", Risk: RiskLow},
		{Description: "Updated job title", Section: "employment", Risk: RiskHigh},
	}}
	store := &fakeProposalStore{}
	task := &ResumeProposalTask{Detector: detector, Store: store, NewID: idGen("prop-"), Now: testClock()}

	proposal, err := task.Run(context.Background(), "profile-1", "software-engineer")
	if err != nil {
		t.Fatalf("Run() error: %v", err)
	}
	if proposal == nil {
		t.Fatal("expected a proposal to be returned")
	}
	if len(proposal.ProposedChanges) != 2 {
		t.Errorf("ProposedChanges = %v, want 2 entries", proposal.ProposedChanges)
	}
	if proposal.Risk != RiskHigh {
		t.Errorf("Risk = %s, want high (max of detected risks)", proposal.Risk)
	}
	if len(store.saved) != 1 {
		t.Errorf("saved proposals = %d, want 1", len(store.saved))
	}
}

func TestResumeProposalTask_Run_DetectorError(t *testing.T) {
	detector := &fakeChangeDetector{err: context.DeadlineExceeded}
	store := &fakeProposalStore{}
	task := &ResumeProposalTask{Detector: detector, Store: store, NewID: idGen("prop-"), Now: testClock()}

	_, err := task.Run(context.Background(), "profile-1", "software-engineer")
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}
}

func TestRiskRank(t *testing.T) {
	if riskRank(RiskHigh) <= riskRank(RiskMedium) {
		t.Error("expected high > medium")
	}
	if riskRank(RiskMedium) <= riskRank(RiskLow) {
		t.Error("expected medium > low")
	}
}
