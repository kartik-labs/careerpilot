package scheduler

import (
	"testing"
	"time"
)

func TestBuildAuditReport(t *testing.T) {
	since := time.Unix(0, 0)
	until := time.Unix(1000, 0)

	runs := []Run{
		{Task: TaskJobDiscovery, Status: RunSucceeded},
		{Task: TaskJobDiscovery, Status: RunSucceeded},
		{Task: TaskJobDiscovery, Status: RunFailed},
		{Task: TaskJobEvaluation, Status: RunDeadLetter},
		{Task: TaskJobEvaluation, Status: RunSkipped},
	}

	report := BuildAuditReport(since, until, runs)

	discovery := report.ByTask[TaskJobDiscovery]
	if discovery.Succeeded != 2 || discovery.Failed != 1 {
		t.Errorf("discovery summary = %+v, want Succeeded=2 Failed=1", discovery)
	}

	evaluation := report.ByTask[TaskJobEvaluation]
	if evaluation.DeadLetter != 1 || evaluation.Skipped != 1 {
		t.Errorf("evaluation summary = %+v, want DeadLetter=1 Skipped=1", evaluation)
	}
}

func TestBuildAuditReport_Empty(t *testing.T) {
	report := BuildAuditReport(time.Unix(0, 0), time.Unix(100, 0), nil)
	if len(report.ByTask) != 0 {
		t.Errorf("ByTask = %v, want empty", report.ByTask)
	}
}
