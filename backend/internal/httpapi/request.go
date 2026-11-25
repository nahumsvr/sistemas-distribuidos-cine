package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"

	"proyecto-cine/backend/internal/domain"
)

const maxBodyBytes = 1 << 20 // 1 MB

// decodeJSON reads exactly one JSON object into dst, rejecting unknown fields
// (so clients cannot sneak in e.g. "role") and bodies over 1 MB. On failure it
// writes the error response and returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	// Requiring application/json forces browsers to send a CORS preflight, so
	// other origins cannot submit these requests as "simple" form posts.
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", "Content-Type must be application/json")
		return false
	}

	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err == nil && dec.Decode(&struct{}{}) != io.EOF {
		err = errors.New("body must contain a single JSON object")
	}
	if err == nil {
		return true
	}

	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Request body must be at most 1 MB")
		return false
	}
	writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Invalid JSON body")
	return false
}

// writeServiceError maps business errors to HTTP. Anything unexpected is
// logged with details and answered with a generic 500.
func writeServiceError(w http.ResponseWriter, logger *slog.Logger, err error) {
	var ve *domain.ValidationError
	switch {
	case errors.As(err, &ve):
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", ve.Message)
	case errors.Is(err, domain.ErrEmailTaken):
		writeError(w, http.StatusConflict, "EMAIL_TAKEN", "Email is already registered")
	case errors.Is(err, domain.ErrSeatTaken):
		writeError(w, http.StatusConflict, "SEAT_TAKEN", "One or more seats are already reserved")
	case errors.Is(err, domain.ErrHasActiveReservations):
		writeError(w, http.StatusConflict, "HAS_ACTIVE_RESERVATIONS", "Cannot modify or delete a resource with active reservations")
	case errors.Is(err, domain.ErrShowtimeStarted):
		writeError(w, http.StatusUnprocessableEntity, "SHOWTIME_STARTED", "Showtime has already started")
	case errors.Is(err, domain.ErrInvalidCredentials):
		writeError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid email or password")
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Resource not found")
	default:
		logger.Error("unexpected error", "err", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL", "Internal server error")
	}
}
