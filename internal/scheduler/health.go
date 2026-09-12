package scheduler

import (
	"context"
	"time"
)

// HealthStatus is the outcome of one dependency health check.
type HealthStatus string

const (
	HealthOK        HealthStatus = "ok"
	HealthUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck is a single named dependency check (database, a model
// provider, etc). CheckFn must be fast and side-effect free — health
// checks are read-only observations, never operations that mutate state.
type HealthCheck struct {
	Name    string
	CheckFn func(ctx context.Context) error
}

// HealthReport is the result of running every configured HealthCheck.
type HealthReport struct {
	Status    HealthStatus
	Checks    map[string]string // name -> "ok" or an error message
	CheckedAt time.Time
}

// HealthChecker runs a fixed set of dependency checks and aggregates them
// into one report (Phase 5 prompt: "Operational health checks.").
type HealthChecker struct {
	Checks []HealthCheck
	Now    func() time.Time
}

// NewHealthChecker builds a HealthChecker over the given checks.
func NewHealthChecker(checks []HealthCheck) *HealthChecker {
	return &HealthChecker{Checks: checks, Now: time.Now}
}

// Run executes every check and aggregates the worst status observed:
// any failure makes the overall report unhealthy — there is no silent
// partial-success reporting.
func (h *HealthChecker) Run(ctx context.Context) HealthReport {
	report := HealthReport{
		Status:    HealthOK,
		Checks:    map[string]string{},
		CheckedAt: h.Now(),
	}

	for _, c := range h.Checks {
		if err := c.CheckFn(ctx); err != nil {
			report.Checks[c.Name] = err.Error()
			report.Status = HealthUnhealthy
			continue
		}
		report.Checks[c.Name] = "ok"
	}

	return report
}
