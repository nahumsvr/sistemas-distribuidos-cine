package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

var testNow = time.Date(2026, 12, 1, 12, 0, 0, 0, time.UTC)

type memCatalog struct {
	movies    []domain.Movie
	showtimes []domain.Showtime
	seats     map[int64][]domain.Seat
}

func (m *memCatalog) ListMovies(context.Context) ([]domain.Movie, error) { return m.movies, nil }

func (m *memCatalog) FindMovie(_ context.Context, id int64) (domain.Movie, error) {
	for _, mv := range m.movies {
		if mv.ID == id {
			return mv, nil
		}
	}
	return domain.Movie{}, domain.ErrNotFound
}

func (m *memCatalog) UpcomingShowtimes(_ context.Context, movieID int64, after time.Time) ([]domain.Showtime, error) {
	var out []domain.Showtime
	for _, s := range m.showtimes {
		if s.MovieID == movieID && s.StartsAt.After(after) {
			out = append(out, s)
		}
	}
	return out, nil
}

func (m *memCatalog) FindShowtime(_ context.Context, id int64) (domain.Showtime, error) {
	for _, s := range m.showtimes {
		if s.ID == id {
			return s, nil
		}
	}
	return domain.Showtime{}, domain.ErrNotFound
}

func (m *memCatalog) SeatsWithAvailability(_ context.Context, showtimeID int64) ([]domain.Seat, error) {
	return m.seats[showtimeID], nil
}

func newTestCatalog() *CatalogService {
	repo := &memCatalog{
		movies: []domain.Movie{{ID: 1, Title: "Dune"}, {ID: 2, Title: "Alien"}},
		showtimes: []domain.Showtime{
			{ID: 10, MovieID: 1, StartsAt: testNow.Add(-time.Hour)},
			{ID: 11, MovieID: 1, StartsAt: testNow.Add(time.Hour)},
		},
		seats: map[int64][]domain.Seat{11: {{ID: 1, Row: "A", Number: 1, Available: true}}},
	}
	svc := NewCatalogService(repo)
	svc.now = func() time.Time { return testNow }
	return svc
}

func TestCatalog_MovieShowtimes_OnlyFuture(t *testing.T) {
	got, err := newTestCatalog().MovieShowtimes(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != 11 {
		t.Fatalf("showtimes = %+v, want only showtime 11", got)
	}
}

func TestCatalog_MovieShowtimes_UnknownMovieIsNotFound(t *testing.T) {
	if _, err := newTestCatalog().MovieShowtimes(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestCatalog_MovieWithoutShowtimes_IsNotAnError(t *testing.T) {
	got, err := newTestCatalog().MovieShowtimes(context.Background(), 2)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %v, %v; want empty and nil", got, err)
	}
}

func TestCatalog_ShowtimeSeats(t *testing.T) {
	st, seats, err := newTestCatalog().ShowtimeSeats(context.Background(), 11)
	if err != nil {
		t.Fatal(err)
	}
	if st.ID != 11 || len(seats) != 1 || !seats[0].Available {
		t.Fatalf("got %+v %+v", st, seats)
	}
}

func TestCatalog_ShowtimeSeats_UnknownShowtimeIsNotFound(t *testing.T) {
	if _, _, err := newTestCatalog().ShowtimeSeats(context.Background(), 99); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}
