package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func newTestRouter(p Pinger) http.Handler {
	return NewRouter(Deps{
		DB:     p,
		Logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func do(t *testing.T, h http.Handler, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func decodeError(t *testing.T, rec *httptest.ResponseRecorder) errorBody {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, want application/json", ct)
	}
	var body errorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("invalid JSON error body %q: %v", rec.Body.String(), err)
	}
	return body
}

func TestHealthz_OK(t *testing.T) {
	rec := do(t, newTestRouter(fakePinger{}), http.MethodGet, "/api/healthz")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil || body["status"] != "ok" {
		t.Fatalf("body = %q, want {\"status\":\"ok\"}", rec.Body.String())
	}
}

func TestHealthz_DatabaseDown(t *testing.T) {
	rec := do(t, newTestRouter(fakePinger{err: errors.New("connection refused")}), http.MethodGet, "/api/healthz")

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	body := decodeError(t, rec)
	if body.Error.Code != "UNAVAILABLE" {
		t.Errorf("code = %q, want UNAVAILABLE", body.Error.Code)
	}
	if body.Error.Message == "connection refused" {
		t.Error("internal error details must not leak to the client")
	}
}

func TestUnknownRoute_ReturnsJSON404(t *testing.T) {
	rec := do(t, newTestRouter(fakePinger{}), http.MethodGet, "/api/does-not-exist")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
	if body := decodeError(t, rec); body.Error.Code != "NOT_FOUND" {
		t.Errorf("code = %q, want NOT_FOUND", body.Error.Code)
	}
}

func TestSecurityHeaders_OnEveryResponse(t *testing.T) {
	h := newTestRouter(fakePinger{})
	for _, path := range []string{"/api/healthz", "/api/does-not-exist"} {
		rec := do(t, h, http.MethodGet, path)
		want := map[string]string{
			"X-Content-Type-Options":  "nosniff",
			"X-Frame-Options":         "DENY",
			"Referrer-Policy":         "strict-origin-when-cross-origin",
			"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
		}
		for k, v := range want {
			if got := rec.Header().Get(k); got != v {
				t.Errorf("%s: header %s = %q, want %q", path, k, got, v)
			}
		}
	}
}

func TestRecover_PanicBecomesJSON500(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := Recover(logger)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		panic("boom")
	}))

	rec := do(t, h, http.MethodGet, "/")

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
	if body := decodeError(t, rec); body.Error.Code != "INTERNAL" {
		t.Errorf("code = %q, want INTERNAL", body.Error.Code)
	}
}
