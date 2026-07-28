package config

import (
	"os"
	"strconv"
)

// Config holds process configuration loaded from the environment.
type Config struct {
	// Addr is the HTTP listen address, e.g. ":8080".
	Addr string
}

// Load reads configuration from environment variables.
// PORT defaults to 8080 when unset or invalid.
func Load() Config {
	port := 8080
	if raw := os.Getenv("PORT"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 && n < 65536 {
			port = n
		}
	}
	return Config{Addr: ":" + strconv.Itoa(port)}
}
