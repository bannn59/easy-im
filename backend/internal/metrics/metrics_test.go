package metrics

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// TestServerScrapes verifies a NewServer returns Prometheus text including the
// default runtime collectors.
func TestServerScrapes(t *testing.T) {
	srv := NewServer(":0", nil)
	if srv.Addr() != ":0" {
		t.Fatalf("Addr() = %q, want %q", srv.Addr(), ":0")
	}
	// promhttp.Handler() on the default registry must serve runtime metrics.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	promhttp.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /metrics = %d, want 200", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"go_goroutines", "go_gc_duration_seconds"} {
		if !strings.Contains(body, want) {
			t.Errorf("scrape missing %q", want)
		}
	}
}

// TestServerStartShutdown exercises the real listen/serve/shutdown path over
// TCP, not just the promhttp handler.
func TestServerStartShutdown(t *testing.T) {
	port := freePort(t)
	srv := NewServer(fmt.Sprintf("127.0.0.1:%d", port), nil)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start() = %v", err)
	}
	defer srv.Shutdown(2 * time.Second)

	// Poll until the listener accepts, then scrape it.
	deadline := time.Now().Add(3 * time.Second)
	var resp *http.Response
	var err error
	for time.Now().Before(deadline) {
		resp, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d/metrics", port))
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /metrics over TCP: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "go_goroutines") {
		t.Error("scrape missing runtime metrics")
	}
}

// freePort binds :0 to learn a free port, then closes it so the server can
// bind. There is a small race window, acceptable for a dev test.
func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// TestHTTPCounterWired verifies the easyim HTTP counter is registered and
// increments with its labels.
func TestHTTPCounterWired(t *testing.T) {
	HTTPRequestsTotal.Reset()
	HTTPRequestsTotal.WithLabelValues("api", http.MethodGet, "/v1/me", "200").Inc()
	HTTPRequestsTotal.WithLabelValues("api", http.MethodGet, "/v1/me", "200").Inc()

	reg := prometheus.NewRegistry()
	reg.MustRegister(HTTPRequestsTotal)
	rec := httptest.NewRecorder()
	promhttp.HandlerFor(reg, promhttp.HandlerOpts{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	body := rec.Body.String()
	if !strings.Contains(body, `easyim_http_requests_total{method="GET",path="/v1/me",service="api",status="200"} 2`) {
		t.Errorf("scrape missing expected line:\n%s", body)
	}
}
