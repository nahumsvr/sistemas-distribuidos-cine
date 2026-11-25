package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

type AdminService interface {
	ListAuditoriums(ctx context.Context) ([]domain.Auditorium, error)
	CreateMovie(ctx context.Context, in service.MovieInput) (domain.Movie, error)
	UpdateMovie(ctx context.Context, id int64, in service.MovieInput) (domain.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
	CreateShowtime(ctx context.Context, in service.ShowtimeInput) (domain.Showtime, error)
	UpdateShowtime(ctx context.Context, id int64, in service.ShowtimeInput) (domain.Showtime, error)
	DeleteShowtime(ctx context.Context, id int64) error
}

type auditoriumJSON struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Rows int    `json:"rows"`
	Cols int    `json:"cols"`
}

type adminShowtimeJSON struct {
	ID           int64     `json:"id"`
	MovieID      int64     `json:"movieId"`
	AuditoriumID int64     `json:"auditoriumId"`
	Auditorium   string    `json:"auditorium"`
	StartsAt     time.Time `json:"startsAt"`
	PriceCents   int       `json:"priceCents"`
}

func toAdminShowtimeJSON(s domain.Showtime) adminShowtimeJSON {
	return adminShowtimeJSON{
		ID:           s.ID,
		MovieID:      s.MovieID,
		AuditoriumID: s.AuditoriumID,
		Auditorium:   s.Auditorium,
		StartsAt:     s.StartsAt.UTC(),
		PriceCents:   s.PriceCents,
	}
}

type movieRequestBody struct {
	Title       string `json:"title"`
	Synopsis    string `json:"synopsis"`
	DurationMin int    `json:"durationMin"`
	Rating      string `json:"rating"`
	PosterURL   string `json:"posterUrl"`
}

func (b movieRequestBody) toInput() service.MovieInput {
	return service.MovieInput{
		Title:       b.Title,
		Synopsis:    b.Synopsis,
		DurationMin: b.DurationMin,
		Rating:      b.Rating,
		PosterURL:   b.PosterURL,
	}
}

type showtimeRequestBody struct {
	MovieID      int64     `json:"movieId"`
	AuditoriumID int64     `json:"auditoriumId"`
	StartsAt     time.Time `json:"startsAt"`
	PriceCents   int       `json:"priceCents"`
}

func (b showtimeRequestBody) toInput() service.ShowtimeInput {
	return service.ShowtimeInput{
		MovieID:      b.MovieID,
		AuditoriumID: b.AuditoriumID,
		StartsAt:     b.StartsAt,
		PriceCents:   b.PriceCents,
	}
}

func adminListAuditoriumsHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := svc.ListAuditoriums(r.Context())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		out := make([]auditoriumJSON, 0, len(list))
		for _, a := range list {
			out = append(out, auditoriumJSON{ID: a.ID, Name: a.Name, Rows: a.Rows, Cols: a.Cols})
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func adminCreateMovieHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body movieRequestBody
		if !decodeJSON(w, r, &body) {
			return
		}
		m, err := svc.CreateMovie(r.Context(), body.toInput())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusCreated, toMovieJSON(m))
	}
}

func adminUpdateMovieHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var body movieRequestBody
		if !decodeJSON(w, r, &body) {
			return
		}
		m, err := svc.UpdateMovie(r.Context(), id, body.toInput())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, toMovieJSON(m))
	}
}

func adminDeleteMovieHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := svc.DeleteMovie(r.Context(), id); err != nil {
			writeServiceError(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func adminCreateShowtimeHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body showtimeRequestBody
		if !decodeJSON(w, r, &body) {
			return
		}
		st, err := svc.CreateShowtime(r.Context(), body.toInput())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusCreated, toAdminShowtimeJSON(st))
	}
}

func adminUpdateShowtimeHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var body showtimeRequestBody
		if !decodeJSON(w, r, &body) {
			return
		}
		st, err := svc.UpdateShowtime(r.Context(), id, body.toInput())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, toAdminShowtimeJSON(st))
	}
}

func adminDeleteShowtimeHandler(svc AdminService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		if err := svc.DeleteShowtime(r.Context(), id); err != nil {
			writeServiceError(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
