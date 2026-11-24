package httpapi

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"proyecto-cine/backend/internal/auth"
	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

type AuthService interface {
	Register(ctx context.Context, in service.RegisterInput) (service.AuthResult, error)
	Login(ctx context.Context, email, password string) (service.AuthResult, error)
	Me(ctx context.Context, userID int64) (domain.User, error)
}

type TokenParser interface {
	Parse(token string) (auth.Claims, error)
}

// userJSON is the public view of a user: never includes the password hash.
type userJSON struct {
	ID        int64       `json:"id"`
	Name      string      `json:"name"`
	Email     string      `json:"email"`
	Role      domain.Role `json:"role"`
	CreatedAt time.Time   `json:"createdAt"`
}

func toUserJSON(u domain.User) userJSON {
	return userJSON{ID: u.ID, Name: u.Name, Email: u.Email, Role: u.Role, CreatedAt: u.CreatedAt}
}

type authJSON struct {
	Token string   `json:"token"`
	User  userJSON `json:"user"`
}

func registerHandler(svc AuthService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		res, err := svc.Register(r.Context(), service.RegisterInput{Name: body.Name, Email: body.Email, Password: body.Password})
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusCreated, authJSON{Token: res.Token, User: toUserJSON(res.User)})
	}
}

func loginHandler(svc AuthService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Email    string `json:"email"`
			Password string `json:"password"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		res, err := svc.Login(r.Context(), body.Email, body.Password)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, authJSON{Token: res.Token, User: toUserJSON(res.User)})
	}
}

func meHandler(svc AuthService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFrom(r.Context())
		user, err := svc.Me(r.Context(), claims.UserID)
		if errors.Is(err, domain.ErrNotFound) {
			// Valid token but the account no longer exists.
			unauthorized(w)
			return
		}
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]userJSON{"user": toUserJSON(user)})
	}
}

type claimsKey struct{}

// ClaimsFrom returns the authenticated caller set by RequireAuth.
func ClaimsFrom(ctx context.Context) (auth.Claims, bool) {
	c, ok := ctx.Value(claimsKey{}).(auth.Claims)
	return c, ok
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Missing, invalid or expired token")
}

// RequireAuth accepts only "Authorization: Bearer <valid JWT>" (the scheme is
// case-insensitive, RFC 7235) and stores the caller's claims in the context.
func RequireAuth(tokens TokenParser) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			scheme, raw, _ := strings.Cut(r.Header.Get("Authorization"), " ")
			if !strings.EqualFold(scheme, "Bearer") || strings.TrimSpace(raw) == "" {
				unauthorized(w)
				return
			}
			claims, err := tokens.Parse(strings.TrimSpace(raw))
			if err != nil {
				unauthorized(w)
				return
			}
			next(w, r.WithContext(context.WithValue(r.Context(), claimsKey{}, claims)))
		}
	}
}
