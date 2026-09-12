package scheduler

import "time"

// AuditReport summarizes scheduled-task activity over a window, for
// operator visibility (Phase 5 prompt: "Audit reporting.").
type AuditReport struct {
	Since time.Time
	Until time.Time

	ByTask map[TaskType]TaskSummary
}

// TaskSummary aggregates run outcomes for one TaskType.
type TaskSummary struct {
	Succeeded  int
	Failed     int
	DeadLetter int
	Skipped    int
}

// BuildAuditReport aggregates a list of Run records into a report. It is
// pure aggregation over already-fetched data — callers fetch runs via
// RunStore.ListRuns for each TaskType they care about and pass them here.
func BuildAuditReport(since, until time.Time, runs []Run) AuditReport {
	report := AuditReport{
		Since:  since,
		Until:  until,
		ByTask: map[TaskType]TaskSummary{},
	}

	for _, r := range runs {
		summary := report.ByTask[r.Task]
		switch r.Status {
		case RunSucceeded:
			summary.Succeeded++
		case RunFailed:
			summary.Failed++
		case RunDeadLetter:
			summary.DeadLetter++
		case RunSkipped:
			summary.Skipped++
		}
		report.ByTask[r.Task] = summary
	}

	return report
}
