package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

type memAdminRepo struct {
	nextMovieID    int64
	nextShowtimeID int64
	auditoriums    map[int64]domain.Auditorium
	movies         map[int64]domain.Movie
	showtimes      map[int64]domain.Showtime
	activeShowtime map[int64]bool
}

func newMemAdminRepo() *memAdminRepo {
	return &memAdminRepo{
		nextMovieID:    10,
		nextShowtimeID: 100,
		auditoriums: map[int64]domain.Auditorium{
			1: {ID: 1, Name: "Sala 1", Rows: 8, Cols: 10},
			2: {ID: 2, Name: "Sala 2", Rows: 6, Cols: 8},
		},
		movies: map[int64]domain.Movie{
			1: {ID: 1, Title: "Dune", DurationMin: 155, Rating: "B"},
		},
		showtimes:      map[int64]domain.Showtime{},
		activeShowtime: map[int64]bool{},
	}
}

func (m *memAdminRepo) ListAuditoriums(context.Context) ([]domain.Auditorium, error) {
	return []domain.Auditorium{m.auditoriums[1], m.auditoriums[2]}, nil
}

func (m *memAdminRepo) CreateMovie(_ context.Context, mv domain.Movie) (domain.Movie, error) {
	m.nextMovieID++
	mv.ID = m.nextMovieID
	m.movies[mv.ID] = mv
	return mv, nil
}

func (m *memAdminRepo) UpdateMovie(_ context.Context, mv domain.Movie) (domain.Movie, error) {
	if _, ok := m.movies[mv.ID]; !ok {
		return domain.Movie{}, domain.ErrNotFound
	}
	m.movies[mv.ID] = mv
	return mv, nil
}

func (m *memAdminRepo) DeleteMovie(_ context.Context, id int64) error {
	if _, ok := m.movies[id]; !ok {
		return domain.ErrNotFound
	}
	for sid, st := range m.showtimes {
		if st.MovieID == id && m.activeShowtime[sid] {
			return domain.ErrHasActiveReservations
		}
	}
	for sid, st := range m.showtimes {
		if st.MovieID == id {
			delete(m.showtimes, sid)
		}
	}
	delete(m.movies, id)
	return nil
}

func (m *memAdminRepo) CreateShowtime(_ context.Context, st domain.Showtime) (domain.Showtime, error) {
	if _, ok := m.movies[st.MovieID]; !ok {
		return domain.Showtime{}, domain.ErrNotFound
	}
	aud, ok := m.auditoriums[st.AuditoriumID]
	if !ok {
		return domain.Showtime{}, domain.ErrNotFound
	}
	m.nextShowtimeID++
	st.ID = m.nextShowtimeID
	st.Auditorium = aud.Name
	m.showtimes[st.ID] = st
	return st, nil
}

func (m *memAdminRepo) UpdateShowtime(_ context.Context, st domain.Showtime) (domain.Showtime, error) {
	cur, ok := m.showtimes[st.ID]
	if !ok {
		return domain.Showtime{}, domain.ErrNotFound
	}
	if _, ok := m.movies[st.MovieID]; !ok {
		return domain.Showtime{}, domain.ErrNotFound
	}
	aud, ok := m.auditoriums[st.AuditoriumID]
	if !ok {
		return domain.Showtime{}, domain.ErrNotFound
	}
	if cur.AuditoriumID != st.AuditoriumID && m.activeShowtime[st.ID] {
		return domain.Showtime{}, domain.ErrHasActiveReservations
	}
	st.Auditorium = aud.Name
	m.showtimes[st.ID] = st
	return st, nil
}

func (m *memAdminRepo) DeleteShowtime(_ context.Context, id int64) error {
	if _, ok := m.showtimes[id]; !ok {
		return domain.ErrNotFound
	}
	if m.activeShowtime[id] {
		return domain.ErrHasActiveReservations
	}
	delete(m.showtimes, id)
	return nil
}

func newTestAdminService() (*AdminService, *memAdminRepo) {
	repo := newMemAdminRepo()
	svc := NewAdminService(repo)
	svc.now = func() time.Time { return testNow }
	return svc, repo
}

func TestAdminService_Movies(t *testing.T) {
	t.Run("creates, updates and trims fields", func(t *testing.T) {
		svc, _ := newTestAdminService()
		created, err := svc.CreateMovie(context.Background(), MovieInput{
			Title:       "  Arrival  ",
			Synopsis:    "  Linguistics and time.  ",
			DurationMin: 116,
			Rating:      " B ",
			PosterURL:   "https://cdn.example.com/arrival.jpg",
		})
		if err != nil {
			t.Fatalf("CreateMovie: %v", err)
		}
		if created.ID == 0 || created.Title != "Arrival" || created.Rating != "B" {
			t.Fatalf("unexpected movie: %+v", created)
		}

		updated, err := svc.UpdateMovie(context.Background(), created.ID, MovieInput{
			Title:       "Arrival (2016)",
			DurationMin: 116,
		})
		if err != nil || updated.Title != "Arrival (2016)" || updated.PosterURL != "" {
			t.Fatalf("UpdateMovie = %+v, %v", updated, err)
		}
	})

	invalidMovies := []struct {
		name string
		in   MovieInput
	}{
		{"empty title", MovieInput{Title: "   ", DurationMin: 90}},
		{"title over 200 chars", MovieInput{Title: strings.Repeat("a", 201), DurationMin: 90}},
		{"zero duration", MovieInput{Title: "Ok", DurationMin: 0}},
		{"negative duration", MovieInput{Title: "Ok", DurationMin: -10}},
		{"synopsis too long", MovieInput{Title: "Ok", DurationMin: 90, Synopsis: strings.Repeat("x", 2001)}},
		{"rating too long", MovieInput{Title: "Ok", DurationMin: 90, Rating: strings.Repeat("R", 21)}},
		{"javascript posterUrl", MovieInput{Title: "Ok", DurationMin: 90, PosterURL: "javascript:alert(1)"}},
		{"relative posterUrl", MovieInput{Title: "Ok", DurationMin: 90, PosterURL: "/img.png"}},
	}
	for _, tc := range invalidMovies {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestAdminService()
			_, err := svc.CreateMovie(context.Background(), tc.in)
			var ve *domain.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *domain.ValidationError", err)
			}
		})
	}

	t.Run("delete blocks when movie has active reservations", func(t *testing.T) {
		svc, repo := newTestAdminService()
		st, err := svc.CreateShowtime(context.Background(), ShowtimeInput{
			MovieID: 1, AuditoriumID: 1, StartsAt: testNow.Add(2 * time.Hour), PriceCents: 8000,
		})
		if err != nil {
			t.Fatal(err)
		}
		repo.activeShowtime[st.ID] = true

		if err := svc.DeleteMovie(context.Background(), 1); !errors.Is(err, domain.ErrHasActiveReservations) {
			t.Fatalf("err = %v, want ErrHasActiveReservations", err)
		}
		repo.activeShowtime[st.ID] = false
		if err := svc.DeleteMovie(context.Background(), 1); err != nil {
			t.Fatalf("DeleteMovie without active reservations: %v", err)
		}
	})

	t.Run("unknown movie returns ErrNotFound", func(t *testing.T) {
		svc, _ := newTestAdminService()
		if _, err := svc.UpdateMovie(context.Background(), 999, MovieInput{Title: "X", DurationMin: 90}); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("UpdateMovie err = %v, want ErrNotFound", err)
		}
		if err := svc.DeleteMovie(context.Background(), 999); !errors.Is(err, domain.ErrNotFound) {
			t.Errorf("DeleteMovie err = %v, want ErrNotFound", err)
		}
	})
}

func TestAdminService_Showtimes(t *testing.T) {
	future := testNow.Add(3 * time.Hour)

	t.Run("creates, updates and deletes showtime", func(t *testing.T) {
		svc, _ := newTestAdminService()
		st, err := svc.CreateShowtime(context.Background(), ShowtimeInput{
			MovieID: 1, AuditoriumID: 1, StartsAt: future, PriceCents: 0,
		})
		if err != nil {
			t.Fatalf("CreateShowtime: %v", err)
		}
		if st.ID == 0 || st.Auditorium != "Sala 1" || st.PriceCents != 0 {
			t.Fatalf("unexpected showtime: %+v", st)
		}

		upd, err := svc.UpdateShowtime(context.Background(), st.ID, ShowtimeInput{
			MovieID: 1, AuditoriumID: 2, StartsAt: future.Add(time.Hour), PriceCents: 9500,
		})
		if err != nil || upd.Auditorium != "Sala 2" || upd.PriceCents != 9500 {
			t.Fatalf("UpdateShowtime = %+v, %v", upd, err)
		}

		if err := svc.DeleteShowtime(context.Background(), st.ID); err != nil {
			t.Fatalf("DeleteShowtime: %v", err)
		}
	})

	invalidShowtimes := []struct {
		name string
		in   ShowtimeInput
	}{
		{"non-positive movieId", ShowtimeInput{MovieID: 0, AuditoriumID: 1, StartsAt: future, PriceCents: 100}},
		{"non-positive auditoriumId", ShowtimeInput{MovieID: 1, AuditoriumID: 0, StartsAt: future, PriceCents: 100}},
		{"missing startsAt", ShowtimeInput{MovieID: 1, AuditoriumID: 1, PriceCents: 100}},
		{"past startsAt", ShowtimeInput{MovieID: 1, AuditoriumID: 1, StartsAt: testNow.Add(-time.Minute), PriceCents: 100}},
		{"now startsAt", ShowtimeInput{MovieID: 1, AuditoriumID: 1, StartsAt: testNow, PriceCents: 100}},
		{"negative priceCents", ShowtimeInput{MovieID: 1, AuditoriumID: 1, StartsAt: future, PriceCents: -1}},
	}
	for _, tc := range invalidShowtimes {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestAdminService()
			_, err := svc.CreateShowtime(context.Background(), tc.in)
			var ve *domain.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *domain.ValidationError", err)
			}
		})
	}

	t.Run("active reservations block auditorium change and deletion", func(t *testing.T) {
		svc, repo := newTestAdminService()
		st, _ := svc.CreateShowtime(context.Background(), ShowtimeInput{
			MovieID: 1, AuditoriumID: 1, StartsAt: future, PriceCents: 8000,
		})
		repo.activeShowtime[st.ID] = true

		if _, err := svc.UpdateShowtime(context.Background(), st.ID, ShowtimeInput{
			MovieID: 1, AuditoriumID: 2, StartsAt: future, PriceCents: 8000,
		}); !errors.Is(err, domain.ErrHasActiveReservations) {
			t.Errorf("auditorium change err = %v, want ErrHasActiveReservations", err)
		}
		if err := svc.DeleteShowtime(context.Background(), st.ID); !errors.Is(err, domain.ErrHasActiveReservations) {
			t.Errorf("delete err = %v, want ErrHasActiveReservations", err)
		}
	})
}
