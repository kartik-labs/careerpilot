package config

import "testing"

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("CAREERPILOT_ENV", "")
	t.Setenv("CAREERPILOT_HTTP_ADDR", "")
	t.Setenv("CAREERPILOT_DATABASE_URL", "")
	t.Setenv("CAREERPILOT_LOG_LEVEL", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Env != "development" {
		t.Errorf("Env = %q, want development", cfg.Env)
	}
	if cfg.HTTPAddr != ":8080" {
		t.Errorf("HTTPAddr = %q, want :8080", cfg.HTTPAddr)
	}
	if cfg.Limits.ApplicationsPerDay != 15 {
		t.Errorf("ApplicationsPerDay = %d, want 15", cfg.Limits.ApplicationsPerDay)
	}
}

func TestLoad_ProductionRequiresDatabaseURL(t *testing.T) {
	t.Setenv("CAREERPILOT_ENV", "production")
	t.Setenv("CAREERPILOT_DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for production without DATABASE_URL, got nil")
	}
}

func TestLoad_InvalidLimitInt(t *testing.T) {
	t.Setenv("CAREERPILOT_ENV", "development")
	t.Setenv("CAREERPILOT_LIMIT_APPLICATIONS_PER_DAY", "not-a-number")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() expected error for invalid integer limit, got nil")
	}
}

func TestLoad_CustomLimits(t *testing.T) {
	t.Setenv("CAREERPILOT_ENV", "development")
	t.Setenv("CAREERPILOT_LIMIT_APPLICATIONS_PER_DAY", "5")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg.Limits.ApplicationsPerDay != 5 {
		t.Errorf("ApplicationsPerDay = %d, want 5", cfg.Limits.ApplicationsPerDay)
	}
}
