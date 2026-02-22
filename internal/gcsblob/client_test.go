package gcsblob

import (
	"os"
	"testing"
	"time"
)

func TestDefaultConfigUsesExpectedPrecedence(t *testing.T) {
	t.Setenv("GOEX_GCS_EMULATOR_HOST", "")
	t.Setenv("STORAGE_EMULATOR_HOST", "")
	t.Setenv("GOEX_GCS_PROJECT_ID", "")
	t.Setenv("GOEX_GCS_REQUEST_TIMEOUT", "")

	cfg := DefaultConfig()
	if cfg.Endpoint != "http://127.0.0.1:4443" {
		t.Fatalf("unexpected default endpoint: %q", cfg.Endpoint)
	}
	if cfg.ProjectID != "goex" {
		t.Fatalf("unexpected default project id: %q", cfg.ProjectID)
	}
	if cfg.RequestTimeout != 30*time.Second {
		t.Fatalf("unexpected default request timeout: %s", cfg.RequestTimeout)
	}

	t.Setenv("STORAGE_EMULATOR_HOST", "http://127.0.0.1:1111")
	cfg = DefaultConfig()
	if cfg.Endpoint != "http://127.0.0.1:1111" {
		t.Fatalf("expected STORAGE_EMULATOR_HOST endpoint, got %q", cfg.Endpoint)
	}

	t.Setenv("GOEX_GCS_EMULATOR_HOST", "http://127.0.0.1:2222")
	cfg = DefaultConfig()
	if cfg.Endpoint != "http://127.0.0.1:2222" {
		t.Fatalf("expected GOEX_GCS_EMULATOR_HOST endpoint, got %q", cfg.Endpoint)
	}

	t.Setenv("GOEX_GCS_PROJECT_ID", "project-123")
	t.Setenv("GOEX_GCS_REQUEST_TIMEOUT", "45s")
	cfg = DefaultConfig()
	if cfg.ProjectID != "project-123" {
		t.Fatalf("unexpected project id override: %q", cfg.ProjectID)
	}
	if cfg.RequestTimeout != 45*time.Second {
		t.Fatalf("unexpected request timeout override: %s", cfg.RequestTimeout)
	}
}

func TestEnvDurationOrFallsBackOnInvalid(t *testing.T) {
	name := "GOEX_GCS_REQUEST_TIMEOUT"
	_ = os.Unsetenv(name)

	fallback := 12 * time.Second
	if got := envDurationOr(name, fallback); got != fallback {
		t.Fatalf("expected fallback when env empty, got %s", got)
	}

	t.Setenv(name, "invalid")
	if got := envDurationOr(name, fallback); got != fallback {
		t.Fatalf("expected fallback when env invalid, got %s", got)
	}
}
