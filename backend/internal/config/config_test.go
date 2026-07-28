package config

import (
	"testing"
)

func TestLoadDefaultPort(t *testing.T) {
	t.Setenv("PORT", "")
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Fatalf("Addr = %q, want :8080", cfg.Addr)
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
