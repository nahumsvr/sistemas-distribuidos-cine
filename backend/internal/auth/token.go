// Package auth issues and validates the JWT access tokens (see ADR-001).
package auth

import (
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"proyecto-cine/backend/internal/domain"
)

const issuer = "cine-api"

var ErrInvalidToken = errors.New("invalid token")

// Claims is what the rest of the app needs to know about the caller.
type Claims struct {
	UserID int64
	Role   domain.Role
}

type claims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

func NewTokens(secret []byte, ttl time.Duration) *Tokens {
	return &Tokens{secret: secret, ttl: ttl, now: time.Now}
}

// Issue signs an HS256 token with the user's id (sub), role and expiration.
func (t *Tokens) Issue(u domain.User) (string, error) {
	now := t.now()
	c := claims{
		Role: u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(u.ID, 10),
			Issuer:    issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(t.secret)
}

// Parse validates signature, algorithm (HS256 only: blocks "none" and
// algorithm-confusion attacks), issuer and expiration, then the claims shape.
func (t *Tokens) Parse(token string) (Claims, error) {
	var c claims
	_, err := jwt.ParseWithClaims(token, &c,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	id, err := strconv.ParseInt(c.Subject, 10, 64)
	if err != nil || id <= 0 || !c.Role.Valid() {
		return Claims{}, ErrInvalidToken
	}
	return Claims{UserID: id, Role: c.Role}, nil
}
