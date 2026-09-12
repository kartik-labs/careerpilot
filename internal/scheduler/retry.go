package scheduler

import (
	"fmt"
	"time"
)

// RetryPolicy bounds how many times a failed task attempt may be retried
// before it becomes a dead letter requiring human attention
// (CLAUDE.md "per_application: max_retries: 2"; Phase 5 prompt:
// "A failed operation should become an explicit failure state.").
type RetryPolicy struct {
	MaxRetries int
}

// Outcome classifies what should happen after a task attempt fails,
// given how many attempts have already been made.
type Outcome string

const (
	OutcomeRetry      Outcome = "retry"
	OutcomeDeadLetter Outcome = "dead_letter"
)

// Decide returns whether attemptNumber (1-indexed) should be retried or
// treated as a dead letter. It never recommends unbounded retry —
// attemptNumber > MaxRetries+1 always dead-letters
// (Phase 5 prompt: "Never allow an infinite loop.").
func (p RetryPolicy) Decide(attemptNumber int) Outcome {
	if attemptNumber <= p.MaxRetries {
		return OutcomeRetry
	}
	return OutcomeDeadLetter
}

// Execute runs fn up to policy.MaxRetries+1 times, stopping at the first
// success. If every attempt fails, it returns a Run with Status
// DEAD_LETTER — never an infinite loop, and never a silently swallowed
// failure.
func Execute(policy RetryPolicy, task TaskType, newID func() string, now func() time.Time, fn func(attempt int) error) Run {
	var lastErr error
	attempt := 0

	for {
		attempt++
		start := now()
		err := fn(attempt)
		finish := now()

		if err == nil {
			return Run{
				ID:         newID(),
				Task:       task,
				Status:     RunSucceeded,
				Attempt:    attempt,
				StartedAt:  start,
				FinishedAt: finish,
			}
		}

		lastErr = err

		if policy.Decide(attempt) == OutcomeDeadLetter {
			return Run{
				ID:         newID(),
				Task:       task,
				Status:     RunDeadLetter,
				Attempt:    attempt,
				Detail:     fmt.Sprintf("exhausted retries: %v", lastErr),
				StartedAt:  start,
				FinishedAt: finish,
			}
		}
	}
}
