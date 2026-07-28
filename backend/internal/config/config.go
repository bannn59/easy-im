package config

import (
	"os"
	"strconv"
	"strings"
)

// Config holds process configuration loaded from the environment.
type Config struct {
	// Addr is the HTTP listen address, e.g. ":8080".
	Addr string
	// DatabaseURL is a Postgres connection string. Empty means no DB.
	DatabaseURL string
}

// Load reads configuration from environment variables.
// PORT defaults to 8080 when unset or invalid.
// DATABASE_URL is optional; when empty the API still serves /healthz.
func Load() Config {
	port := 8080
	if raw := os.Getenv("PORT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n < 65536 {
			port = n
		}
	}
	return Config{
		Addr:        ":" + strconv.Itoa(port),
		DatabaseURL: strings.TrimSpace(os.Getenv("DATABASE_URL")),
	}
}
