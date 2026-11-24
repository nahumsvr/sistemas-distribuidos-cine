// Package httpapi is the HTTP layer: routing, middleware, request/response
// mapping. Business rules live in the service layer, not here.
package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

// Pinger is the only thing the health check needs from the database.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Deps struct {
	DB            Pinger
	Logger        *slog.Logger
	Auth          AuthService
	Tokens        TokenParser
	Catalog       CatalogService
	AllowedOrigin string
	// AuthRateLimit applies to login and register. Zero value means
	// DefaultAuthRateLimit.
	AuthRateLimit RateLimit
}

func NewRouter(d Deps) http.Handler {
	rl := d.AuthRateLimit
	if rl.Burst == 0 {
		rl = DefaultAuthRateLimit
	}
	limited := newIPLimiter(rl).wrap
	authed := RequireAuth(d.Tokens)

	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", healthz(d.DB, d.Logger))

	mux.HandleFunc("POST /api/auth/register", limited(registerHandler(d.Auth, d.Logger)))
	mux.HandleFunc("POST /api/auth/login", limited(loginHandler(d.Auth, d.Logger)))
	mux.HandleFunc("GET /api/auth/me", authed(meHandler(d.Auth, d.Logger)))

	// Public catalog (read-only).
	mux.HandleFunc("GET /api/movies", listMoviesHandler(d.Catalog, d.Logger))
	mux.HandleFunc("GET /api/movies/{id}", getMovieHandler(d.Catalog, d.Logger))
	mux.HandleFunc("GET /api/movies/{id}/showtimes", movieShowtimesHandler(d.Catalog, d.Logger))
	mux.HandleFunc("GET /api/showtimes/{id}/seats", showtimeSeatsHandler(d.Catalog, d.Logger))

	// Fallback so unknown routes also answer with the JSON error shape.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	})

	return chain(mux,
		Recover(d.Logger),
		RequestLogger(d.Logger),
		SecurityHeaders,
		CORS(d.AllowedOrigin),
	)
}

func healthz(db Pinger, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.Ping(ctx); err != nil {
			logger.Warn("health check failed", "err", err)
			writeError(w, http.StatusServiceUnavailable, "UNAVAILABLE", "Database unavailable")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}
