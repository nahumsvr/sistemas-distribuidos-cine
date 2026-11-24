package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"proyecto-cine/backend/internal/domain"
)

type CatalogService interface {
	ListMovies(ctx context.Context) ([]domain.Movie, error)
	GetMovie(ctx context.Context, id int64) (domain.Movie, error)
	MovieShowtimes(ctx context.Context, movieID int64) ([]domain.Showtime, error)
	ShowtimeSeats(ctx context.Context, showtimeID int64) (domain.Showtime, []domain.Seat, error)
}

type movieJSON struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Synopsis    string `json:"synopsis"`
	DurationMin int    `json:"durationMin"`
	Rating      string `json:"rating"`
	PosterURL   string `json:"posterUrl"`
}

type showtimeJSON struct {
	ID         int64     `json:"id"`
	MovieID    int64     `json:"movieId"`
	Auditorium string    `json:"auditorium"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
}

type seatJSON struct {
	ID        int64  `json:"id"`
	Row       string `json:"row"`
	Number    int    `json:"number"`
	Available bool   `json:"available"`
}

func toMovieJSON(m domain.Movie) movieJSON {
	return movieJSON{ID: m.ID, Title: m.Title, Synopsis: m.Synopsis, DurationMin: m.DurationMin, Rating: m.Rating, PosterURL: m.PosterURL}
}

func toShowtimeJSON(s domain.Showtime) showtimeJSON {
	return showtimeJSON{ID: s.ID, MovieID: s.MovieID, Auditorium: s.Auditorium, StartsAt: s.StartsAt.UTC(), PriceCents: s.PriceCents}
}

// pathID parses a positive integer path parameter. On failure it writes the
// 400 response and returns false.
func pathID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "id must be a positive integer")
		return 0, false
	}
	return id, true
}

func listMoviesHandler(svc CatalogService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		movies, err := svc.ListMovies(r.Context())
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		out := make([]movieJSON, 0, len(movies)) // non-nil: "[]", never "null"
		for _, m := range movies {
			out = append(out, toMovieJSON(m))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func getMovieHandler(svc CatalogService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		m, err := svc.GetMovie(r.Context(), id)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, toMovieJSON(m))
	}
}

func movieShowtimesHandler(svc CatalogService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		showtimes, err := svc.MovieShowtimes(r.Context(), id)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		out := make([]showtimeJSON, 0, len(showtimes))
		for _, s := range showtimes {
			out = append(out, toShowtimeJSON(s))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func showtimeSeatsHandler(svc CatalogService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		st, seats, err := svc.ShowtimeSeats(r.Context(), id)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		out := make([]seatJSON, 0, len(seats))
		for _, s := range seats {
			out = append(out, seatJSON{ID: s.ID, Row: s.Row, Number: s.Number, Available: s.Available})
		}
		writeJSON(w, http.StatusOK, struct {
			Showtime showtimeJSON `json:"showtime"`
			Seats    []seatJSON   `json:"seats"`
		}{toShowtimeJSON(st), out})
	}
}
