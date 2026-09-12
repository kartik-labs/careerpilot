package scheduler

import (
	"context"
	"fmt"

	"github.com/kartik-labs/careerpilot/internal/job"
)

// PendingJobLister returns jobs awaiting triage/evaluation. Kept separate
// from job.Store so the scheduler can query "what needs evaluation"
// without every job.Store implementation needing that query.
type PendingJobLister interface {
	ListPendingEvaluation(limit int) ([]job.Job, error)
}

// EvaluationTask runs the job.Pipeline over jobs awaiting triage/deep
// evaluation, respecting the pipeline's own daily limits
// (job.Limits — CLAUDE.md "Initial Resource Limits").
type EvaluationTask struct {
	Lister    PendingJobLister
	Pipeline  *job.Pipeline
	BatchSize int
}

// EvaluationResult summarizes one evaluation run.
type EvaluationResult struct {
	Processed int
	Applied   int // recommendation == APPLY
	Reviewed  int
	Rejected  int
}

// Run evaluates up to BatchSize pending jobs. It stops early (without
// error) once job.Pipeline itself reports the daily discovery/deep-eval
// limit has been reached — that is an expected, non-exceptional stop
// condition, not a failure.
func (t *EvaluationTask) Run(ctx context.Context) (EvaluationResult, error) {
	var result EvaluationResult

	jobs, err := t.Lister.ListPendingEvaluation(t.BatchSize)
	if err != nil {
		return result, fmt.Errorf("scheduler: list pending jobs: %w", err)
	}

	for _, j := range jobs {
		score, err := t.Pipeline.Process(ctx, j)
		if err != nil {
			// A limit-reached error stops this run cleanly; any other
			// error is reported so it can be retried/dead-lettered by the
			// caller's RetryPolicy, but does not corrupt already-processed
			// results.
			return result, fmt.Errorf("scheduler: evaluate job %s: %w", j.ID, err)
		}

		result.Processed++
		switch score.Recommendation {
		case job.RecommendationApply:
			result.Applied++
		case job.RecommendationReject:
			result.Rejected++
		default:
			result.Reviewed++
		}
	}

	return result, nil
}
