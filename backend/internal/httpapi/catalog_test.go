package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

func seedCatalog(d testDeps) {
	start := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	d.catalog.movies = []domain.Movie{
		{ID: 1, Title: "Dune", Synopsis: "Sand.", DurationMin: 155, Rating: "PG-13", PosterURL: "http://x/p.jpg"},
		{ID: 2, Title: "Alien", DurationMin: 117},
	}
	d.catalog.showtimes = []domain.Showtime{
		{ID: 7, MovieID: 1, Auditorium: "Sala 1", StartsAt: start, PriceCents: 8000},
		{ID: 8, MovieID: 1, Auditorium: "Sala 2", StartsAt: time.Now().Add(-time.Hour), PriceCents: 8000},
	}
	d.catalog.seats = map[int64][]domain.Seat{
		7: {{ID: 1, Row: "A", Number: 1, Available: true}, {ID: 2, Row: "A", Number: 2, Available: false}},
	}
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestCatalog_ListMovies_IsPublicAndCamelCase(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)

	rec := get(h, "/api/movies") // no Authorization header
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0]["title"] != "Dune" || got[0]["durationMin"] != float64(155) ||
		got[0]["posterUrl"] != "http://x/p.jpg" || got[0]["id"] != float64(1) {
		t.Fatalf("unexpected body: %s", rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("content-type = %q", ct)
	}
}

func TestCatalog_ListMovies_EmptyIsArrayNotNull(t *testing.T) {
	h, _ := newTestRouter(t)
	if body := strings.TrimSpace(get(h, "/api/movies").Body.String()); body != "[]" {
		t.Fatalf("body = %q, want []", body)
	}
}

func TestCatalog_GetMovie(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)

	rec := get(h, "/api/movies/2")
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"title":"Alien"`) {
		t.Fatalf("status %d body %s", rec.Code, rec.Body)
	}
}

func TestCatalog_GetMovie_NotFoundAndBadID(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)

	if rec := get(h, "/api/movies/99"); rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"NOT_FOUND"`) {
		t.Errorf("missing: status %d body %s", rec.Code, rec.Body)
	}
	for _, id := range []string{"abc", "0", "-1", "1.5"} {
		rec := get(h, "/api/movies/"+id)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"VALIDATION_ERROR"`) {
			t.Errorf("id %q: status %d body %s", id, rec.Code, rec.Body)
		}
	}
}

func TestCatalog_MovieShowtimes_OnlyFutureWithExpectedShape(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)

	rec := get(h, "/api/movies/1/showtimes")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got []map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &got)
	if len(got) != 1 {
		t.Fatalf("want 1 future showtime, got %s", rec.Body)
	}
	s := got[0]
	if s["id"] != float64(7) || s["movieId"] != float64(1) || s["auditorium"] != "Sala 1" || s["priceCents"] != float64(8000) {
		t.Errorf("unexpected showtime: %v", s)
	}
	if _, err := time.Parse(time.RFC3339, s["startsAt"].(string)); err != nil {
		t.Errorf("startsAt not RFC3339: %v", err)
	}
}

func TestCatalog_MovieShowtimes_UnknownMovie404(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)
	if rec := get(h, "/api/movies/99/showtimes"); rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestCatalog_ShowtimeSeats(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)

	rec := get(h, "/api/showtimes/7/seats")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", rec.Code, rec.Body)
	}
	var got struct {
		Showtime map[string]any   `json:"showtime"`
		Seats    []map[string]any `json:"seats"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Showtime["id"] != float64(7) || got.Showtime["auditorium"] != "Sala 1" {
		t.Errorf("showtime = %v", got.Showtime)
	}
	if len(got.Seats) != 2 || got.Seats[0]["row"] != "A" || got.Seats[0]["number"] != float64(1) ||
		got.Seats[0]["available"] != true || got.Seats[1]["available"] != false {
		t.Errorf("seats = %v", got.Seats)
	}
}

func TestCatalog_ShowtimeSeats_NotFoundAndBadID(t *testing.T) {
	h, d := newTestRouter(t)
	seedCatalog(d)
	if rec := get(h, "/api/showtimes/99/seats"); rec.Code != http.StatusNotFound {
		t.Errorf("missing: status %d", rec.Code)
	}
	if rec := get(h, "/api/showtimes/x/seats"); rec.Code != http.StatusBadRequest {
		t.Errorf("bad id: status %d", rec.Code)
	}
}

// Catalog routes are read-only: writes belong to the admin API (T6).
func TestCatalog_WriteMethodsAreNotRouted(t *testing.T) {
	h, _ := newTestRouter(t)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/movies", strings.NewReader("{}")))
	if rec.Code == http.StatusOK || rec.Code == http.StatusCreated {
		t.Fatalf("POST /api/movies must not succeed, got %d", rec.Code)
	}
}
