package httpapi

import (
	"context"
	"io"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"proyecto-cine/backend/internal/auth"
	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

const testOrigin = "http://localhost:3000"

var testSecret = []byte("0123456789abcdef0123456789abcdef")

type testDeps struct {
	tokens       *auth.Tokens
	users        *memUsers
	catalog      *memCatalog
	reservations *memReservations
	admin        *memAdmin
}

// memCatalog is an in-memory service.CatalogRepository. Tests fill it in.
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

func (m *memCatalog) SeatsWithAvailability(_ context.Context, id int64) ([]domain.Seat, error) {
	return m.seats[id], nil
}

// newTestRouterFull wires the real router, service and token code with an
// in-memory user store, so tests exercise the public HTTP interface.
func newTestRouterFull(t *testing.T, db Pinger, rl RateLimit) (http.Handler, testDeps) {
	t.Helper()
	users := &memUsers{byID: map[int64]domain.User{}}
	catalog := &memCatalog{}
	reservations := newMemReservations()
	admin := newMemAdmin()
	tokens := auth.NewTokens(testSecret, time.Hour)
	h := NewRouter(Deps{
		DB:            db,
		Logger:        slog.New(slog.NewTextHandler(io.Discard, nil)),
		Auth:          service.NewAuthService(users, tokens, bcrypt.MinCost),
		Catalog:       service.NewCatalogService(catalog),
		Reservations:  service.NewReservationService(reservations),
		Admin:         service.NewAdminService(admin),
		Tokens:        tokens,
		AllowedOrigin: testOrigin,
		AuthRateLimit: rl,
	})
	return h, testDeps{tokens: tokens, users: users, catalog: catalog, reservations: reservations, admin: admin}
}

// generous keeps the limiter out of the way in tests that are not about it.
var generous = RateLimit{Every: time.Millisecond, Burst: 1000}

func newTestRouter(t *testing.T) (http.Handler, testDeps) {
	return newTestRouterFull(t, fakePinger{}, generous)
}

func newTestRouterWith(t *testing.T, rl RateLimit) (http.Handler, testDeps) {
	return newTestRouterFull(t, fakePinger{}, rl)
}

func newTestRouterDB(t *testing.T, p Pinger) http.Handler {
	h, _ := newTestRouterFull(t, p, generous)
	return h
}

type memUsers struct {
	mu     sync.Mutex
	nextID int64
	byID   map[int64]domain.User
}

func (m *memUsers) Create(_ context.Context, u domain.User) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, e := range m.byID {
		if strings.EqualFold(e.Email, u.Email) {
			return domain.User{}, domain.ErrEmailTaken
		}
	}
	m.nextID++
	u.ID = m.nextID
	m.byID[u.ID] = u
	return u, nil
}

func (m *memUsers) FindByEmail(_ context.Context, email string) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, u := range m.byID {
		if strings.EqualFold(u.Email, email) {
			return u, nil
		}
	}
	return domain.User{}, domain.ErrNotFound
}

func (m *memUsers) FindByID(_ context.Context, id int64) (domain.User, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return domain.User{}, domain.ErrNotFound
}

type resSeatKey struct {
	showtimeID int64
	seatID     int64
}

type memReservations struct {
	mu           sync.Mutex
	nextID       int64
	showtimes    map[int64]domain.ReservationShowtime
	auditoriums  map[int64][]domain.ReservedSeat
	reservations map[int64]domain.Reservation
	taken        map[resSeatKey]int64
}

func newMemReservations() *memReservations {
	return &memReservations{
		showtimes:    map[int64]domain.ReservationShowtime{},
		auditoriums:  map[int64][]domain.ReservedSeat{},
		reservations: map[int64]domain.Reservation{},
		taken:        map[resSeatKey]int64{},
	}
}

func (m *memReservations) ListByUser(_ context.Context, userID int64) ([]domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Reservation
	for _, r := range m.reservations {
		if r.UserID == userID {
			r.Seats = slices.Clone(r.Seats)
			out = append(out, r)
		}
	}
	slices.SortFunc(out, func(a, b domain.Reservation) int {
		if a.ID > b.ID {
			return -1
		}
		if a.ID < b.ID {
			return 1
		}
		return 0
	})
	return out, nil
}

func (m *memReservations) FindByID(_ context.Context, id int64) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, domain.ErrNotFound
	}
	r.Seats = slices.Clone(r.Seats)
	return r, nil
}

func (m *memReservations) WithTx(_ context.Context, fn func(service.ReservationTx) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	resCopy := make(map[int64]domain.Reservation, len(m.reservations))
	for k, v := range m.reservations {
		v.Seats = slices.Clone(v.Seats)
		resCopy[k] = v
	}
	tx := &memResTx{
		nextID:       m.nextID,
		showtimes:    m.showtimes,
		auditoriums:  m.auditoriums,
		reservations: resCopy,
		taken:        maps.Clone(m.taken),
	}
	if err := fn(tx); err != nil {
		return err
	}
	m.nextID = tx.nextID
	m.reservations = tx.reservations
	m.taken = tx.taken
	return nil
}

type memResTx struct {
	nextID       int64
	showtimes    map[int64]domain.ReservationShowtime
	auditoriums  map[int64][]domain.ReservedSeat
	reservations map[int64]domain.Reservation
	taken        map[resSeatKey]int64
}

func (t *memResTx) FindShowtime(_ context.Context, id int64) (domain.ReservationShowtime, error) {
	st, ok := t.showtimes[id]
	if !ok {
		return domain.ReservationShowtime{}, domain.ErrNotFound
	}
	return st, nil
}

func (t *memResTx) FindSeatsInAuditorium(_ context.Context, auditoriumID int64, seatIDs []int64) ([]domain.ReservedSeat, error) {
	want := make(map[int64]struct{}, len(seatIDs))
	for _, id := range seatIDs {
		want[id] = struct{}{}
	}
	var out []domain.ReservedSeat
	for _, s := range t.auditoriums[auditoriumID] {
		if _, ok := want[s.ID]; ok {
			out = append(out, s)
		}
	}
	return out, nil
}

func (t *memResTx) FindReservationForUpdate(_ context.Context, id int64) (domain.Reservation, error) {
	r, ok := t.reservations[id]
	if !ok {
		return domain.Reservation{}, domain.ErrNotFound
	}
	r.Seats = slices.Clone(r.Seats)
	return r, nil
}

func (t *memResTx) InsertReservation(_ context.Context, userID, showtimeID int64, seatIDs []int64) (int64, time.Time, error) {
	t.nextID++
	id := t.nextID
	for _, sid := range seatIDs {
		k := resSeatKey{showtimeID: showtimeID, seatID: sid}
		if _, busy := t.taken[k]; busy {
			return 0, time.Time{}, domain.ErrSeatTaken
		}
		t.taken[k] = id
	}
	st := t.showtimes[showtimeID]
	seats, _ := t.FindSeatsInAuditorium(context.Background(), st.AuditoriumID, seatIDs)
	now := time.Now().UTC().Truncate(time.Second)
	t.reservations[id] = domain.Reservation{
		ID:         id,
		UserID:     userID,
		Status:     domain.ReservationActive,
		Showtime:   st,
		Seats:      seats,
		TotalCents: len(seats) * st.PriceCents,
		CreatedAt:  now,
	}
	return id, now, nil
}

func (t *memResTx) ReplaceSeats(_ context.Context, reservationID, showtimeID int64, seatIDs []int64) error {
	for k, owner := range t.taken {
		if owner == reservationID {
			delete(t.taken, k)
		}
	}
	for _, sid := range seatIDs {
		k := resSeatKey{showtimeID: showtimeID, seatID: sid}
		if _, busy := t.taken[k]; busy {
			return domain.ErrSeatTaken
		}
		t.taken[k] = reservationID
	}
	r := t.reservations[reservationID]
	seats, _ := t.FindSeatsInAuditorium(context.Background(), r.Showtime.AuditoriumID, seatIDs)
	r.Seats = seats
	r.TotalCents = len(seats) * r.Showtime.PriceCents
	t.reservations[reservationID] = r
	return nil
}

func (t *memResTx) CancelReservation(_ context.Context, reservationID int64) error {
	for k, owner := range t.taken {
		if owner == reservationID {
			delete(t.taken, k)
		}
	}
	r := t.reservations[reservationID]
	r.Status = domain.ReservationCancelled
	r.Seats = nil
	r.TotalCents = 0
	t.reservations[reservationID] = r
	return nil
}

type memAdmin struct {
	mu             sync.Mutex
	nextMovieID    int64
	nextShowtimeID int64
	auditoriums    map[int64]domain.Auditorium
	movies         map[int64]domain.Movie
	showtimes      map[int64]domain.Showtime
	activeShowtime map[int64]bool
}

func newMemAdmin() *memAdmin {
	return &memAdmin{
		auditoriums:    map[int64]domain.Auditorium{},
		movies:         map[int64]domain.Movie{},
		showtimes:      map[int64]domain.Showtime{},
		activeShowtime: map[int64]bool{},
	}
}

func (m *memAdmin) ListAuditoriums(context.Context) ([]domain.Auditorium, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Auditorium, 0, len(m.auditoriums))
	for _, a := range m.auditoriums {
		out = append(out, a)
	}
	slices.SortFunc(out, func(a, b domain.Auditorium) int { return strings.Compare(a.Name, b.Name) })
	return out, nil
}

func (m *memAdmin) CreateMovie(_ context.Context, mv domain.Movie) (domain.Movie, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextMovieID++
	mv.ID = m.nextMovieID
	m.movies[mv.ID] = mv
	return mv, nil
}

func (m *memAdmin) UpdateMovie(_ context.Context, mv domain.Movie) (domain.Movie, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.movies[mv.ID]; !ok {
		return domain.Movie{}, domain.ErrNotFound
	}
	m.movies[mv.ID] = mv
	return mv, nil
}

func (m *memAdmin) DeleteMovie(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *memAdmin) CreateShowtime(_ context.Context, st domain.Showtime) (domain.Showtime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *memAdmin) UpdateShowtime(_ context.Context, st domain.Showtime) (domain.Showtime, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
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

func (m *memAdmin) DeleteShowtime(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.showtimes[id]; !ok {
		return domain.ErrNotFound
	}
	if m.activeShowtime[id] {
		return domain.ErrHasActiveReservations
	}
	delete(m.showtimes, id)
	return nil
}
