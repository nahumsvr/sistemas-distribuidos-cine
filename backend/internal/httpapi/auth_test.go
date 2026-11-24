package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proyecto-cine/backend/internal/auth"
	"proyecto-cine/backend/internal/domain"
)

func request(t *testing.T, h http.Handler, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func bearer(tok string) map[string]string { return map[string]string{"Authorization": "Bearer " + tok} }

type authResponse struct {
	Token string         `json:"token"`
	User  map[string]any `json:"user"`
}

func decodeAuth(t *testing.T, rec *httptest.ResponseRecorder) authResponse {
	t.Helper()
	var res authResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid JSON %q: %v", rec.Body.String(), err)
	}
	return res
}

const anaJSON = `{"name":"Ana","email":"ana@mail.com","password":"secret123"}`

func TestRegister(t *testing.T) {
	t.Run("creates user and returns token", func(t *testing.T) {
		h, _ := newTestRouter(t)
		rec := request(t, h, "POST", "/api/auth/register", anaJSON, nil)

		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body %s", rec.Code, rec.Body)
		}
		res := decodeAuth(t, rec)
		if res.Token == "" || res.User["email"] != "ana@mail.com" || res.User["role"] != "user" {
			t.Errorf("unexpected body: %s", rec.Body)
		}
		if strings.Contains(strings.ToLower(rec.Body.String()), "password") {
			t.Errorf("response must never include the password or its hash: %s", rec.Body)
		}
	})

	cases := []struct {
		name, body string
		status     int
		code       string
	}{
		{"role cannot be chosen by the client", `{"name":"A","email":"a@mail.com","password":"secret123","role":"admin"}`, 400, "VALIDATION_ERROR"},
		{"invalid fields", `{"name":"","email":"nope","password":"1"}`, 400, "VALIDATION_ERROR"},
		{"malformed JSON", `{"name":`, 400, "VALIDATION_ERROR"},
		{"two JSON objects", anaJSON + anaJSON, 400, "VALIDATION_ERROR"},
		{"body over 1 MB", `{"name":"` + strings.Repeat("a", 1<<20) + `"}`, 413, "PAYLOAD_TOO_LARGE"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := newTestRouter(t)
			rec := request(t, h, "POST", "/api/auth/register", tc.body, nil)
			if rec.Code != tc.status {
				t.Fatalf("status = %d, want %d; body %s", rec.Code, tc.status, rec.Body)
			}
			if got := decodeError(t, rec).Error.Code; got != tc.code {
				t.Errorf("code = %q, want %q", got, tc.code)
			}
		})
	}

	t.Run("content type must be JSON", func(t *testing.T) {
		h, _ := newTestRouter(t)
		// text/plain is a CORS "simple request": other sites could send it
		// from a browser without a preflight, so it must be rejected.
		for _, ct := range []string{"text/plain", "application/x-www-form-urlencoded", ""} {
			rec := request(t, h, "POST", "/api/auth/register", anaJSON, map[string]string{"Content-Type": ct})
			if rec.Code != http.StatusUnsupportedMediaType || decodeError(t, rec).Error.Code != "UNSUPPORTED_MEDIA_TYPE" {
				t.Errorf("%q: got %d %s, want 415 UNSUPPORTED_MEDIA_TYPE", ct, rec.Code, rec.Body)
			}
		}
		rec := request(t, h, "POST", "/api/auth/register", anaJSON, map[string]string{"Content-Type": "application/json; charset=utf-8"})
		if rec.Code != http.StatusCreated {
			t.Errorf("JSON with charset: got %d %s, want 201", rec.Code, rec.Body)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		h, _ := newTestRouter(t)
		request(t, h, "POST", "/api/auth/register", anaJSON, nil)
		rec := request(t, h, "POST", "/api/auth/register", strings.Replace(anaJSON, "ana@", "ANA@", 1), nil)
		if rec.Code != http.StatusConflict || decodeError(t, rec).Error.Code != "EMAIL_TAKEN" {
			t.Fatalf("got %d %s, want 409 EMAIL_TAKEN", rec.Code, rec.Body)
		}
	})
}

func TestLogin(t *testing.T) {
	h, _ := newTestRouter(t)
	request(t, h, "POST", "/api/auth/register", anaJSON, nil)

	ok := request(t, h, "POST", "/api/auth/login", `{"email":"ana@mail.com","password":"secret123"}`, nil)
	if ok.Code != http.StatusOK || decodeAuth(t, ok).Token == "" {
		t.Fatalf("valid login: %d %s", ok.Code, ok.Body)
	}

	wrongPass := request(t, h, "POST", "/api/auth/login", `{"email":"ana@mail.com","password":"bad-pass"}`, nil)
	unknown := request(t, h, "POST", "/api/auth/login", `{"email":"ghost@mail.com","password":"secret123"}`, nil)
	for name, rec := range map[string]*httptest.ResponseRecorder{"wrong password": wrongPass, "unknown email": unknown} {
		if rec.Code != http.StatusUnauthorized || decodeError(t, rec).Error.Code != "UNAUTHORIZED" {
			t.Errorf("%s: got %d %s, want 401 UNAUTHORIZED", name, rec.Code, rec.Body)
		}
	}
	if wrongPass.Body.String() != unknown.Body.String() {
		t.Errorf("responses must be identical to avoid user enumeration:\n%s\n%s", wrongPass.Body, unknown.Body)
	}
}

func TestMe(t *testing.T) {
	h, deps := newTestRouter(t)
	token := decodeAuth(t, request(t, h, "POST", "/api/auth/register", anaJSON, nil)).Token

	t.Run("scheme is case-insensitive (RFC 7235)", func(t *testing.T) {
		rec := request(t, h, "GET", "/api/auth/me", "", map[string]string{"Authorization": "bearer " + token})
		if rec.Code != http.StatusOK {
			t.Fatalf("got %d %s, want 200", rec.Code, rec.Body)
		}
	})

	t.Run("valid token", func(t *testing.T) {
		rec := request(t, h, "GET", "/api/auth/me", "", bearer(token))
		var body struct{ User map[string]any }
		_ = json.Unmarshal(rec.Body.Bytes(), &body)
		if rec.Code != http.StatusOK || body.User["email"] != "ana@mail.com" {
			t.Fatalf("got %d %s", rec.Code, rec.Body)
		}
	})

	expired := auth.NewTokens(testSecret, time.Nanosecond)
	expiredTok, _ := expired.Issue(domain.User{ID: 1, Role: domain.RoleUser})
	time.Sleep(1100 * time.Millisecond) // exp has second precision; make sure it is in the past
	ghostTok, _ := deps.tokens.Issue(domain.User{ID: 999, Role: domain.RoleUser})

	unauthorized := map[string]map[string]string{
		"no header":           nil,
		"basic scheme":        {"Authorization": "Basic YWxhZGRpbjpvcGVu"},
		"empty bearer":        {"Authorization": "Bearer "},
		"garbage token":       bearer("garbage"),
		"expired token":       bearer(expiredTok),
		"user no longer here": bearer(ghostTok),
	}
	for name, headers := range unauthorized {
		t.Run(name, func(t *testing.T) {
			rec := request(t, h, "GET", "/api/auth/me", "", headers)
			if rec.Code != http.StatusUnauthorized || decodeError(t, rec).Error.Code != "UNAUTHORIZED" {
				t.Fatalf("got %d %s, want 401 UNAUTHORIZED", rec.Code, rec.Body)
			}
			if rec.Header().Get("WWW-Authenticate") != "Bearer" {
				t.Errorf("WWW-Authenticate = %q, want Bearer", rec.Header().Get("WWW-Authenticate"))
			}
		})
	}
}

func TestCORS(t *testing.T) {
	h, _ := newTestRouter(t)
	allowed := map[string]string{"Origin": testOrigin}
	other := map[string]string{"Origin": "https://evil.example"}

	t.Run("allowed origin gets CORS headers", func(t *testing.T) {
		rec := request(t, h, "GET", "/api/healthz", "", allowed)
		if rec.Header().Get("Access-Control-Allow-Origin") != testOrigin {
			t.Errorf("ACAO = %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
		if rec.Header().Get("Vary") != "Origin" {
			t.Errorf("Vary = %q, want Origin", rec.Header().Get("Vary"))
		}
		if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
			t.Error("credentials must not be allowed (token travels in a header, not a cookie)")
		}
	})
	t.Run("other origin gets none", func(t *testing.T) {
		rec := request(t, h, "GET", "/api/healthz", "", other)
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Errorf("ACAO must be empty for other origins, got %q", rec.Header().Get("Access-Control-Allow-Origin"))
		}
	})
	t.Run("preflight from allowed origin", func(t *testing.T) {
		rec := request(t, h, "OPTIONS", "/api/reservations", "", map[string]string{
			"Origin":                         testOrigin,
			"Access-Control-Request-Method":  "POST",
			"Access-Control-Request-Headers": "authorization, content-type",
		})
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "DELETE") ||
			!strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), "Authorization") ||
			rec.Header().Get("Access-Control-Allow-Origin") != testOrigin {
			t.Errorf("incomplete preflight headers: %v", rec.Header())
		}
	})
	t.Run("preflight from other origin", func(t *testing.T) {
		rec := request(t, h, "OPTIONS", "/api/reservations", "", map[string]string{
			"Origin": "https://evil.example", "Access-Control-Request-Method": "POST",
		})
		if rec.Header().Get("Access-Control-Allow-Origin") != "" {
			t.Error("preflight from other origin must not be allowed")
		}
	})
}

func TestAuthRateLimit(t *testing.T) {
	h, _ := newTestRouterWith(t, RateLimit{Every: time.Hour, Burst: 2})
	login := `{"email":"x@mail.com","password":"whatever1"}`
	fromIP := func(ip string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(login))
		req.RemoteAddr = ip + ":5555"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	fromIP("10.0.0.1")
	fromIP("10.0.0.1")
	blocked := fromIP("10.0.0.1")
	if blocked.Code != http.StatusTooManyRequests || decodeError(t, blocked).Error.Code != "RATE_LIMITED" {
		t.Fatalf("3rd attempt: got %d %s, want 429 RATE_LIMITED", blocked.Code, blocked.Body)
	}
	if blocked.Header().Get("Retry-After") == "" {
		t.Error("429 must include Retry-After")
	}
	if rec := fromIP("10.0.0.2"); rec.Code == http.StatusTooManyRequests {
		t.Error("another IP must not be limited")
	}
	// Spoofed forwarding headers must not bypass the limit (there is no proxy).
	req := httptest.NewRequest("POST", "/api/auth/login", strings.NewReader(login))
	req.RemoteAddr = "10.0.0.1:5555"
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Error("X-Forwarded-For must not reset the limit")
	}
	// Non-auth routes are not limited.
	for range 5 {
		if rec := do(t, h, "GET", "/api/healthz"); rec.Code != http.StatusOK {
			t.Fatalf("healthz limited: %d", rec.Code)
		}
	}
}
