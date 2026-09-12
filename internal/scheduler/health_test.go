package scheduler

import (
	"context"
	"errors"
	"testing"
)

func TestHealthChecker_AllHealthy(t *testing.T) {
	checker := NewHealthChecker([]HealthCheck{
		{Name: "database", CheckFn: func(ctx context.Context) error { return nil }},
		{Name: "gemini", CheckFn: func(ctx context.Context) error { return nil }},
	})

	report := checker.Run(context.Background())
	if report.Status != HealthOK {
		t.Errorf("Status = %s, want ok", report.Status)
	}
	if report.Checks["database"] != "ok" || report.Checks["gemini"] != "ok" {
		t.Errorf("Checks = %v, want all ok", report.Checks)
	}
}

func TestHealthChecker_OneFailureMakesReportUnhealthy(t *testing.T) {
	checker := NewHealthChecker([]HealthCheck{
		{Name: "database", CheckFn: func(ctx context.Context) error { return nil }},
		{Name: "gemini", CheckFn: func(ctx context.Context) error { return errors.New("timeout") }},
	})

	report := checker.Run(context.Background())
	if report.Status != HealthUnhealthy {
		t.Errorf("Status = %s, want unhealthy", report.Status)
	}
	if report.Checks["gemini"] != "timeout" {
		t.Errorf("Checks[gemini] = %q, want timeout", report.Checks["gemini"])
	}
	if report.Checks["database"] != "ok" {
		t.Errorf("Checks[database] = %q, want ok (one failure shouldn't hide other results)", report.Checks["database"])
	}
}
