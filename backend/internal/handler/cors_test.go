package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func corsHarness() http.Handler {
	allowed := map[string]struct{}{"http://allowed.example": {}}
	return withCORS(allowed)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
}

func TestCORSAllowedOriginEchoed(t *testing.T) {
	h := corsHarness()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://allowed.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "http://allowed.example" {
		t.Fatalf("allow-origin = %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("want Allow-Credentials true")
	}
	if rr.Header().Get("Vary") == "" {
		t.Fatal("want Vary: Origin")
	}
}

func TestCORSDisallowedOriginNoHeaders(t *testing.T) {
	h := corsHarness()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://evil.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("allow-origin = %q, want empty", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Vary") == "" {
		t.Fatal("want Vary: Origin even for disallowed origin")
	}
}

func TestCORSPreflightAllowed(t *testing.T) {
	h := corsHarness()
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	req.Header.Set("Origin", "http://allowed.example")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Fatal("want Allow-Methods on preflight")
	}
}

func TestCORSSameOriginNoOriginHeader(t *testing.T) {
	h := corsHarness()
	// No Origin header (same-origin / non-browser) → no CORS headers, request passes.
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Code != http.StatusNoContent {
		t.Fatalf("status = %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatalf("allow-origin = %q, want empty", rr.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestSessionCookieHelpers(t *testing.T) {
	rr := httptest.NewRecorder()
	setSessionCookie(rr, "jwt-token", CookieConfig{})
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	for _, c := range rr.Result().Cookies() {
		if c.Name == sessionCookieName {
			req.AddCookie(c)
		}
	}
	if got := sessionToken(req); got != "jwt-token" {
		t.Fatalf("sessionToken = %q", got)
	}

	// Missing cookie → empty.
	if got := sessionToken(httptest.NewRequest(http.MethodGet, "/", nil)); got != "" {
		t.Fatalf("sessionToken without cookie = %q", got)
	}
}
