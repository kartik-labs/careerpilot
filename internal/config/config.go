// Package config loads CareerPilot runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration for the CareerPilot service.
type Config struct {
	Env         string
	HTTPAddr    string
	DatabaseURL string
	LogLevel    string

	Limits Limits
}

// Limits holds configurable daily/per-application resource limits.
// See CLAUDE.md "Initial Resource Limits".
type Limits struct {
	JobsDiscoveredPerDay  int
	DeepEvaluationsPerDay int
	ApplicationsPerDay    int
	BrowserSessionsPerDay int
	MaxRetriesPerApp      int
	MaxSessionMinutes     int
}

// Load reads configuration from environment variables, applying defaults
// documented in .env.example.
func Load() (Config, error) {
	cfg := Config{
		Env:         getEnv("CAREERPILOT_ENV", "development"),
		HTTPAddr:    getEnv("CAREERPILOT_HTTP_ADDR", ":8080"),
		DatabaseURL: getEnv("CAREERPILOT_DATABASE_URL", ""),
		LogLevel:    getEnv("CAREERPILOT_LOG_LEVEL", "info"),
	}

	limits, err := loadLimits()
	if err != nil {
		return Config{}, err
	}
	cfg.Limits = limits

	if cfg.Env == "production" && cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: CAREERPILOT_DATABASE_URL is required when CAREERPILOT_ENV=production")
	}

	return cfg, nil
}

func loadLimits() (Limits, error) {
	var (
		l   Limits
		err error
	)

	if l.JobsDiscoveredPerDay, err = getEnvInt("CAREERPILOT_LIMIT_JOBS_DISCOVERED_PER_DAY", 100); err != nil {
		return Limits{}, err
	}
	if l.DeepEvaluationsPerDay, err = getEnvInt("CAREERPILOT_LIMIT_DEEP_EVALUATIONS_PER_DAY", 30); err != nil {
		return Limits{}, err
	}
	if l.ApplicationsPerDay, err = getEnvInt("CAREERPILOT_LIMIT_APPLICATIONS_PER_DAY", 15); err != nil {
		return Limits{}, err
	}
	if l.BrowserSessionsPerDay, err = getEnvInt("CAREERPILOT_LIMIT_BROWSER_SESSIONS_PER_DAY", 10); err != nil {
		return Limits{}, err
	}
	if l.MaxRetriesPerApp, err = getEnvInt("CAREERPILOT_LIMIT_MAX_RETRIES_PER_APPLICATION", 2); err != nil {
		return Limits{}, err
	}
	if l.MaxSessionMinutes, err = getEnvInt("CAREERPILOT_LIMIT_MAX_SESSION_MINUTES", 30); err != nil {
		return Limits{}, err
	}

	return l, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) (int, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: invalid integer for %s: %w", key, err)
	}
	return n, nil
}
