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
	DB     Pinger
	Logger *slog.Logger
}

func NewRouter(d Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/healthz", healthz(d.DB, d.Logger))

	// Fallback so unknown routes also answer with the JSON error shape.
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	})

	return chain(mux,
		Recover(d.Logger),
		RequestLogger(d.Logger),
		SecurityHeaders,
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
