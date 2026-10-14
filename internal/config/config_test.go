package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Env != EnvDevelopment || cfg.HTTP.Port != 8080 || cfg.Log.Format != "text" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
	if cfg.HTTP.ShutdownTimeout != 15*time.Second {
		t.Errorf("ShutdownTimeout = %v", cfg.HTTP.ShutdownTimeout)
	}
}

func TestLoad_ProductionDefaultsToJSONLogs(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("APP_ENV", "production")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Log.Format != "json" || !cfg.IsProduction() {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestLoad_CollectsErrors(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("HTTP_PORT", "abc")
	t.Setenv("APP_ENV", "staging")

	if _, err := Load(); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLoad_ParsesLists(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://u:p@localhost:5432/db")
	t.Setenv("CORS_ALLOWED_ORIGINS", " https://a.com, https://b.com ,,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(cfg.CORS.AllowedOrigins) != 2 || cfg.CORS.AllowedOrigins[1] != "https://b.com" {
		t.Errorf("AllowedOrigins = %v", cfg.CORS.AllowedOrigins)
	}
}
