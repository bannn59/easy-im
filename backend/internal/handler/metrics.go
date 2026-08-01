package handler

import (
	"bufio"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"easy-im/backend/internal/metrics"
)

// statusRecorder captures the response status code for metrics and forwards
// optional interfaces (Flusher, Hijacker) so wrapped handlers keep working —
// most importantly the WebSocket upgrade path, which requires http.Hijacker.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
	emitDone    bool
	record      func(status int)
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.wroteHeader = true
		r.status = code
	}
	r.ResponseWriter.WriteHeader(code)
}

// Flush forwards http.Flusher.
func (r *statusRecorder) Flush() {
	if f, ok := r.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// Hijack forwards http.Hijacker so WebSocket upgrades keep working behind the
// metrics middleware, and records the request at upgrade time so the HTTP
// duration histogram is not skewed by the long-lived connection.
func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("metrics: underlying ResponseWriter does not implement http.Hijacker")
	}
	conn, brw, err := hj.Hijack()
	if err == nil {
		r.wroteHeader = true
		r.status = http.StatusSwitchingProtocols
		r.emit()
	}
	return conn, brw, err
}

// Unwrap exposes the underlying writer for http.ResponseController.
func (r *statusRecorder) Unwrap() http.ResponseWriter { return r.ResponseWriter }

// emit invokes the recording callback exactly once.
func (r *statusRecorder) emit() {
	if r.emitDone {
		return
	}
	r.emitDone = true
	if r.record != nil {
		r.record(r.status)
	}
}

// uuidSeg matches a UUID path segment. Every dynamic route segment in this
// codebase (conversation/message/user ids) is a uuid.NewString() value, so
// normalizing these keeps metric label cardinality bounded.
var uuidSeg = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

// normalizePath replaces UUID path segments with {id} so metric labels stay
// bounded (e.g. /v1/conversations/{id}/messages).
func normalizePath(p string) string {
	segs := strings.Split(p, "/")
	for i, s := range segs {
		if uuidSeg.MatchString(s) {
			segs[i] = "{id}"
		}
	}
	return strings.Join(segs, "/")
}

// MetricsMiddleware records HTTP request count and duration. Dynamic path
// segments are normalized to {id} to keep label cardinality bounded.
func MetricsMiddleware(service string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			path := normalizePath(r.URL.Path)
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			rec.record = func(status int) {
				metrics.HTTPRequestsTotal.WithLabelValues(service, r.Method, path, strconv.Itoa(status)).Inc()
				metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
			}
			next.ServeHTTP(rec, r)
			rec.emit()
		})
	}
}
