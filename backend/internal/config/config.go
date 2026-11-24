// Package config loads and validates the API configuration from environment
// variables. The app fails fast at startup if something required is missing.
package config

import (
	"errors"
	"fmt"
	"time"
)

const minJWTSecretBytes = 32

type Config struct {
	DatabaseURL    string
	Port           string
	FrontendOrigin string
	JWTSecret      []byte
	JWTTTL         time.Duration
}

// Load reads the configuration using getenv (os.Getenv in production; a map in
// tests). Errors name the offending variable but never include secret values.
func Load(getenv func(string) string) (Config, error) {
	cfg := Config{
		DatabaseURL:    getenv("DATABASE_URL"),
		Port:           withDefault(getenv("API_PORT"), "8080"),
		FrontendOrigin: withDefault(getenv("FRONTEND_ORIGIN"), "http://localhost:3000"),
		JWTSecret:      []byte(getenv("JWT_SECRET")),
	}

	var errs []error
	if cfg.DatabaseURL == "" {
		errs = append(errs, errors.New("DATABASE_URL is required"))
	}
	if len(cfg.JWTSecret) < minJWTSecretBytes {
		errs = append(errs, fmt.Errorf("JWT_SECRET must be at least %d bytes", minJWTSecretBytes))
	}

	ttl, err := time.ParseDuration(withDefault(getenv("JWT_TTL"), "1h"))
	switch {
	case err != nil:
		errs = append(errs, errors.New("JWT_TTL must be a duration like 1h or 30m"))
	case ttl <= 0:
		errs = append(errs, errors.New("JWT_TTL must be positive"))
	default:
		cfg.JWTTTL = ttl
	}

	if len(errs) > 0 {
		return Config{}, fmt.Errorf("invalid configuration: %w", errors.Join(errs...))
	}
	return cfg, nil
}

func withDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
