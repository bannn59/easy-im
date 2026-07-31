package config

import (
	"testing"
	"time"
)

func TestLoadDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DATABASE_URL", "")
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("AUTH_DEV_INSECURE", "")
	t.Setenv("AUTH_TOKEN_TTL", "")
	t.Setenv("CORS_ALLOWED_ORIGINS", "")
	t.Setenv("COOKIE_SECURE", "")
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
	if cfg.DatabaseURL != "" {
		t.Fatalf("DatabaseURL = %q, want empty", cfg.DatabaseURL)
	}
	if cfg.AuthJWTSecret != "" {
		t.Fatalf("AuthJWTSecret should be empty without insecure flag")
	}
	if cfg.AuthTokenTTL != 24*time.Hour {
		t.Fatalf("AuthTokenTTL = %v", cfg.AuthTokenTTL)
	}
	if len(cfg.CORSAllowedOrigins) != 1 || cfg.CORSAllowedOrigins[0] != "http://localhost:5173" {
		t.Fatalf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if cfg.CookieSecure {
		t.Fatal("CookieSecure should default false")
	}
}

func TestLoadCORSAndCookie(t *testing.T) {
	t.Setenv("CORS_ALLOWED_ORIGINS", "https://a.example.com, https://b.example.com")
	t.Setenv("COOKIE_SECURE", "1")
	t.Setenv("COOKIE_DOMAIN", ".example.com")
	cfg := Load()
	if len(cfg.CORSAllowedOrigins) != 2 {
		t.Fatalf("CORSAllowedOrigins = %v", cfg.CORSAllowedOrigins)
	}
	if !cfg.CookieSecure {
		t.Fatal("CookieSecure should be true")
	}
	if cfg.CookieDomain != ".example.com" {
		t.Fatalf("CookieDomain = %q", cfg.CookieDomain)
	}
}

func TestLoadCustomPort(t *testing.T) {
	t.Setenv("PORT", "9090")
	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Fatalf("Addr = %q, want :9090", cfg.Addr)
	}
}

func TestLoadInvalidPortFallsBack(t *testing.T) {
	t.Setenv("PORT", "nope")
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
	}
}

func TestLoadDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "  postgres://u:p@localhost:5432/db?sslmode=disable  ")
	cfg := Load()
	if cfg.DatabaseURL != "postgres://u:p@localhost:5432/db?sslmode=disable" {
		t.Fatalf("DatabaseURL = %q", cfg.DatabaseURL)
	}
}

func TestLoadAuthDevInsecure(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "")
	t.Setenv("AUTH_DEV_INSECURE", "1")
	cfg := Load()
	if cfg.AuthJWTSecret != "easyim-dev-secret-change-me" {
		t.Fatalf("secret = %q", cfg.AuthJWTSecret)
	}
}

func TestLoadAuthTokenTTL(t *testing.T) {
	t.Setenv("AUTH_TOKEN_TTL", "1h")
	cfg := Load()
	if cfg.AuthTokenTTL != time.Hour {
		t.Fatalf("ttl = %v", cfg.AuthTokenTTL)
	}
}
