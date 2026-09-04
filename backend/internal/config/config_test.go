package config

import (
	"path/filepath"
	"testing"
)

func TestLoadFromEnvironment(t *testing.T) {
	t.Setenv("OPSHUB_DATABASE_DSN", "host=db.example.internal user=opshub password=test-only dbname=opshub")
	t.Setenv("OPSHUB_ENCRYPT_KEY", "0123456789abcdef0123456789abcdef")
	t.Setenv("OPSHUB_AUTH_SECRET", "abcdef0123456789abcdef0123456789")
	t.Setenv("OPSHUB_ADMIN_USERNAME", "test-admin")
	t.Setenv("OPSHUB_ADMIN_PASSWORD", "test-password-only")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.Security.AdminUsername != "test-admin" {
		t.Fatalf("AdminUsername = %q, want test-admin", cfg.Security.AdminUsername)
	}
}

func TestLoadRejectsMissingSecrets(t *testing.T) {
	t.Setenv("OPSHUB_DATABASE_DSN", "")
	t.Setenv("OPSHUB_ENCRYPT_KEY", "")
	t.Setenv("OPSHUB_AUTH_SECRET", "")
	t.Setenv("OPSHUB_ADMIN_PASSWORD", "")

	if _, err := Load(filepath.Join(t.TempDir(), "missing.yaml")); err == nil {
		t.Fatal("Load() error = nil, want validation error")
	}
}
