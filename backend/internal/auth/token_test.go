package auth

import (
	"strconv"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"proyecto-cine/backend/internal/domain"
)

var secret = []byte("0123456789abcdef0123456789abcdef")

func fixedNow() time.Time { return time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC) }

func newTokens() *Tokens {
	t := NewTokens(secret, time.Hour)
	t.now = fixedNow
	return t
}

// sign builds a token by hand to simulate tampered or foreign tokens.
func sign(t *testing.T, method jwt.SigningMethod, key any, c claims) string {
	t.Helper()
	s, err := jwt.NewWithClaims(method, c).SignedString(key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return s
}

func validClaims() claims {
	return claims{
		Role: "user",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "7",
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(fixedNow()),
			ExpiresAt: jwt.NewNumericDate(fixedNow().Add(time.Hour)),
		},
	}
}

func TestIssueAndParse_RoundTrip(t *testing.T) {
	tokens := newTokens()
	tok, err := tokens.Issue(domain.User{ID: 42, Role: domain.RoleAdmin})
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}

	got, err := tokens.Parse(tok)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if got.UserID != 42 || got.Role != domain.RoleAdmin {
		t.Errorf("claims = %+v, want UserID 42 role admin", got)
	}
}

func TestParse_RejectsExpiredToken(t *testing.T) {
	tokens := newTokens()
	tok, _ := tokens.Issue(domain.User{ID: 1, Role: domain.RoleUser})

	tokens.now = func() time.Time { return fixedNow().Add(time.Hour + time.Second) }
	if _, err := tokens.Parse(tok); err == nil {
		t.Fatal("expired token was accepted")
	}
}

func TestParse_RejectsTamperedOrForeignTokens(t *testing.T) {
	noExp := validClaims()
	noExp.ExpiresAt = nil
	otherIssuer := validClaims()
	otherIssuer.Issuer = "someone-else"
	badRole := validClaims()
	badRole.Role = "superuser"
	badSubject := validClaims()
	badSubject.Subject = "abc"

	cases := map[string]string{
		"wrong secret":    sign(t, jwt.SigningMethodHS256, []byte("another-secret-another-secret-123"), validClaims()),
		"alg none":        sign(t, jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, validClaims()),
		"alg HS384":       sign(t, jwt.SigningMethodHS384, secret, validClaims()),
		"missing exp":     sign(t, jwt.SigningMethodHS256, secret, noExp),
		"other issuer":    sign(t, jwt.SigningMethodHS256, secret, otherIssuer),
		"unknown role":    sign(t, jwt.SigningMethodHS256, secret, badRole),
		"non-numeric sub": sign(t, jwt.SigningMethodHS256, secret, badSubject),
		"garbage":         "not.a.jwt",
		"empty":           "",
	}
	tokens := newTokens()
	for name, tok := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := tokens.Parse(tok); err == nil {
				t.Fatal("token was accepted")
			}
		})
	}

	// Sanity check: the hand-built valid token IS accepted, so the cases above
	// fail for the reason they name and not because sign() is broken.
	if c, err := tokens.Parse(sign(t, jwt.SigningMethodHS256, secret, validClaims())); err != nil || strconv.FormatInt(c.UserID, 10) != "7" {
		t.Fatalf("valid hand-built token rejected: %v", err)
	}
}
