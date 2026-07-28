package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"easy-im/backend/internal/apperr"
)

func TestWriteErrorNotFound(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withRequestID(req.Context(), "rid-1"))
	rr := httptest.NewRecorder()
	WriteError(rr, req, apperr.NotFound("user not found"))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rr.Code)
	}
	var body errorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != "not_found" || body.Error.RequestID != "rid-1" {
		t.Fatalf("body = %+v", body.Error)
	}
	if body.Error.Message != "user not found" {
		t.Fatalf("message = %q", body.Error.Message)
	}
}

func TestWriteErrorInternalHidesCause(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(withRequestID(req.Context(), "rid-2"))
	rr := httptest.NewRecorder()
	WriteError(rr, req, apperr.Internal("should not leak", apperr.ErrInternal))

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rr.Code)
	}
	var body errorBody
	if err := json.NewDecoder(rr.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Message != "internal server error" || body.Error.Code != "internal" {
		t.Fatalf("body = %+v", body.Error)
	}
}

func TestRequestIDMiddlewareGenerates(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) == "" {
			t.Fatal("missing request id in context")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
	if rr.Header().Get(headerRequestID) == "" {
		t.Fatal("missing response X-Request-ID")
	}
}

func TestRequestIDMiddlewarePropagates(t *testing.T) {
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RequestIDFromContext(r.Context()) != "client-rid" {
			t.Fatalf("id = %q", RequestIDFromContext(r.Context()))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(headerRequestID, "client-rid")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Header().Get(headerRequestID) != "client-rid" {
		t.Fatalf("response id = %q", rr.Header().Get(headerRequestID))
	}
}

func TestHTTPStatusMapping(t *testing.T) {
	cases := []struct {
		err  error
		code int
	}{
		{apperr.Invalid("x"), http.StatusBadRequest},
		{apperr.Unauthorized("x"), http.StatusUnauthorized},
		{apperr.Forbidden("x"), http.StatusForbidden},
		{apperr.NotFound("x"), http.StatusNotFound},
		{apperr.Conflict("x"), http.StatusConflict},
		{apperr.Unavailable("x"), http.StatusServiceUnavailable},
		{apperr.Internal("x", nil), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		if got := HTTPStatus(tc.err); got != tc.code {
			t.Fatalf("%v: status = %d, want %d", tc.err, got, tc.code)
		}
	}
}
