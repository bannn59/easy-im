package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"easy-im/backend/internal/metrics"
)

func TestNormalizePath(t *testing.T) {
	cases := map[string]string{
		"/healthz":                       "/healthz",
		"/v1/conversations":              "/v1/conversations",
		"/v1/conversations/abc/messages": "/v1/conversations/abc/messages", // non-UUID untouched
		"/v1/conversations/u1/u2":        "/v1/conversations/u1/u2",
	}
	for in, want := range cases {
		if got := normalizePath(in); got != want {
			t.Errorf("normalizePath(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizePathUUID(t *testing.T) {
	id := "3f2504e0-4f89-11d3-9a0c-0305e82c3301"
	got := normalizePath("/v1/conversations/" + id + "/messages")
	want := "/v1/conversations/{id}/messages"
	if got != want {
		t.Errorf("normalizePath = %q, want %q", got, want)
	}
}

func TestMetricsMiddlewareRecords(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	h := MetricsMiddleware("api")(inner)

	req := httptest.NewRequest(http.MethodGet, "/v1/conversations/3f2504e0-4f89-11d3-9a0c-0305e82c3301/messages", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

// TestMetricsMiddlewarePreservesWSUpgrade guards the WebSocket upgrade path,
// which fails if the middleware's recorder does not forward http.Hijacker.
func TestMetricsMiddlewarePreservesWSUpgrade(t *testing.T) {
	metrics.HTTPRequestsTotal.Reset()
	upgrader := websocket.Upgrader{}
	h := MetricsMiddleware("api")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := upgrader.Upgrade(w, r, nil); err != nil {
			t.Fatalf("upgrade behind metrics middleware: %v", err)
		}
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	conn, _, err := websocket.DefaultDialer.Dial("ws"+srv.URL[len("http"):]+"/v1/ws", nil)
	if err != nil {
		t.Fatalf("websocket dial behind metrics middleware: %v", err)
	}
	conn.Close()

	// The request should be recorded at upgrade time with status 101.
	body := scrapeDefaultRegistry()
	if !strings.Contains(body, `easyim_http_requests_total{method="GET",path="/v1/ws",service="api",status="101"} 1`) {
		t.Errorf("ws request metric not recorded at upgrade time:\n%s", body)
	}
}

// scrapeDefaultRegistry returns the default registry's Prometheus text format.
func scrapeDefaultRegistry() string {
	rec := httptest.NewRecorder()
	promhttp.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	return rec.Body.String()
}
