package config

import (
	"strings"
	"testing"
	"time"
)

const validSecret = "0123456789abcdef0123456789abcdef" // 32 bytes

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoad_AppliesDefaults(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"DATABASE_URL": "postgres://u:p@db:5432/cine",
		"JWT_SECRET":   validSecret,
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.JWTTTL != time.Hour {
		t.Errorf("JWTTTL = %v, want 1h", cfg.JWTTTL)
	}
	if cfg.FrontendOrigin != "http://localhost:3000" {
		t.Errorf("FrontendOrigin = %q, want http://localhost:3000", cfg.FrontendOrigin)
	}
	if cfg.DatabaseURL != "postgres://u:p@db:5432/cine" {
		t.Errorf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoad_ReadsOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"DATABASE_URL":    "postgres://x",
		"JWT_SECRET":      validSecret,
		"API_PORT":        "9090",
		"JWT_TTL":         "30m",
		"FRONTEND_ORIGIN": "http://localhost:5173",
	}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != "9090" || cfg.JWTTTL != 30*time.Minute || cfg.FrontendOrigin != "http://localhost:5173" {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_RejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{"missing database url", map[string]string{"JWT_SECRET": validSecret}, "DATABASE_URL"},
		{"missing jwt secret", map[string]string{"DATABASE_URL": "postgres://x"}, "JWT_SECRET"},
		{"short jwt secret", map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "short"}, "JWT_SECRET"},
		{"invalid ttl", map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": validSecret, "JWT_TTL": "soon"}, "JWT_TTL"},
		{"non-positive ttl", map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": validSecret, "JWT_TTL": "0s"}, "JWT_TTL"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Load(env(tt.env))
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error %q does not mention %s", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_ErrorNeverLeaksSecret(t *testing.T) {
	_, err := Load(env(map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "tooshort-secret"}))
	if err == nil || strings.Contains(err.Error(), "tooshort-secret") {
		t.Fatalf("error must not contain the secret value: %v", err)
	}
}
