package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds process configuration loaded from the environment.
type Config struct {
	// Addr is the HTTP listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a Postgres connection string. Empty means no DB.
	DatabaseURL string
	// AuthJWTSecret signs access tokens. Empty disables auth routes (503).
	AuthJWTSecret string
	// AuthTokenTTL is access token lifetime.
	AuthTokenTTL time.Duration
}

// Load reads configuration from environment variables.
func Load() Config {
	port := 8080
	if raw := os.Getenv("PORT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n < 65536 {
			port = n
		}
	}

	ttl := 168 * time.Hour // 7 days
	if raw := strings.TrimSpace(os.Getenv("AUTH_TOKEN_TTL")); raw != "" {
		if d, err := time.ParseDuration(raw); err == nil && d > 0 {
			ttl = d
		}
	}

	secret := strings.TrimSpace(os.Getenv("AUTH_JWT_SECRET"))
	// Dev convenience: allow insecure default only when explicitly opted in.
	if secret == "" && os.Getenv("AUTH_DEV_INSECURE") == "1" {
		secret = "easyim-dev-secret-change-me"
	}

	return Config{
		Addr:          ":" + strconv.Itoa(port),
		DatabaseURL:   strings.TrimSpace(os.Getenv("DATABASE_URL")),
		AuthJWTSecret: secret,
		AuthTokenTTL:  ttl,
	}
}
