package scheduler

import (
	"errors"
	"testing"
	"time"
)

func TestRetryPolicy_Decide(t *testing.T) {
	p := RetryPolicy{MaxRetries: 2}

	cases := []struct {
		attempt int
		want    Outcome
	}{
		{1, OutcomeRetry},
		{2, OutcomeRetry},
		{3, OutcomeDeadLetter},
		{4, OutcomeDeadLetter},
	}

	for _, tc := range cases {
		if got := p.Decide(tc.attempt); got != tc.want {
			t.Errorf("Decide(%d) = %s, want %s", tc.attempt, got, tc.want)
		}
	}
}

func testClock() func() time.Time {
	return func() time.Time { return time.Unix(0, 0) }
}

func idGen(prefix string) func() string {
	i := 0
	return func() string {
		i++
		return prefix + string(rune('0'+i))
	}
}

func TestExecute_SucceedsFirstTry(t *testing.T) {
	calls := 0
	run := Execute(RetryPolicy{MaxRetries: 2}, TaskJobDiscovery, idGen("run-"), testClock(), func(attempt int) error {
		calls++
		return nil
	})

	if run.Status != RunSucceeded {
		t.Errorf("Status = %s, want SUCCEEDED", run.Status)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestExecute_SucceedsAfterRetries(t *testing.T) {
	calls := 0
	run := Execute(RetryPolicy{MaxRetries: 2}, TaskJobDiscovery, idGen("run-"), testClock(), func(attempt int) error {
		calls++
		if attempt < 3 {
			return errors.New("transient failure")
		}
		return nil
	})

	if run.Status != RunSucceeded {
		t.Errorf("Status = %s, want SUCCEEDED", run.Status)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestExecute_DeadLettersAfterExhaustingRetries(t *testing.T) {
	calls := 0
	run := Execute(RetryPolicy{MaxRetries: 2}, TaskJobDiscovery, idGen("run-"), testClock(), func(attempt int) error {
		calls++
		return errors.New("permanent failure")
	})

	if run.Status != RunDeadLetter {
		t.Errorf("Status = %s, want DEAD_LETTER", run.Status)
	}
	if calls != 3 { // 1 initial + 2 retries
		t.Errorf("calls = %d, want 3 (bounded, never infinite)", calls)
	}
	if run.Detail == "" {
		t.Error("expected Detail to explain the dead-letter reason")
	}
}
