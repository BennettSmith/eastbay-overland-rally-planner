package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Overland-East-Bay/trip-planner-api/internal/adapters/httpapi/oas"
)

func TestNewRouter_Healthz_NoAuthRequired(t *testing.T) {
	t.Parallel()

	h := NewRouter(StrictUnimplemented{})

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if rec.Body.String() != "ok" {
		t.Fatalf("body=%q", rec.Body.String())
	}
}

func TestNewDevAuthMiddleware_SubjectResolutionAndErrors(t *testing.T) {
	t.Parallel()

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sub, ok := SubjectFromContext(r.Context())
		if !ok {
			t.Fatalf("missing subject in context")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(sub))
	})

	t.Run("header wins", func(t *testing.T) {
		h := NewDevAuthMiddleware("default")(next)
		req := httptest.NewRequest(http.MethodGet, "/members/me", nil)
		req.Header.Set("X-Debug-Subject", "  sub-1  ")
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Body.String() != "sub-1" {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("falls back to default", func(t *testing.T) {
		h := NewDevAuthMiddleware("  sub-default ")(next)
		req := httptest.NewRequest(http.MethodGet, "/members/me", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK || rec.Body.String() != "sub-default" {
			t.Fatalf("status=%d body=%q", rec.Code, rec.Body.String())
		}
	})

	t.Run("missing subject returns 401", func(t *testing.T) {
		h := NewDevAuthMiddleware("")(next)
		req := httptest.NewRequest(http.MethodGet, "/members/me", nil)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var er oas.ErrorResponse
		_ = json.NewDecoder(rec.Body).Decode(&er)
		if er.Error.Code != "UNAUTHORIZED" {
			t.Fatalf("code=%q", er.Error.Code)
		}
	})
}
