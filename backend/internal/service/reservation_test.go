package service

import (
	"context"
	"errors"
	"maps"
	"slices"
	"sync"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

type seatKey struct {
	showtimeID int64
	seatID     int64
}

type memReservationRepo struct {
	mu           sync.Mutex
	nextID       int64
	showtimes    map[int64]domain.ReservationShowtime
	auditoriums  map[int64][]domain.ReservedSeat
	reservations map[int64]domain.Reservation
	taken        map[seatKey]int64 // (showtime, seat) -> reservationID
}

func newMemReservationRepo() *memReservationRepo {
	return &memReservationRepo{
		showtimes: map[int64]domain.ReservationShowtime{
			10: {ID: 10, AuditoriumID: 1, MovieTitle: "Dune", StartsAt: testNow.Add(-time.Hour), PriceCents: 8000},
			11: {ID: 11, AuditoriumID: 1, MovieTitle: "Dune", StartsAt: testNow.Add(2 * time.Hour), PriceCents: 8000},
			12: {ID: 12, AuditoriumID: 1, MovieTitle: "Dune", StartsAt: testNow, PriceCents: 8000},
			20: {ID: 20, AuditoriumID: 2, MovieTitle: "Alien", StartsAt: testNow.Add(3 * time.Hour), PriceCents: 7500},
		},
		auditoriums: map[int64][]domain.ReservedSeat{
			1: {
				{ID: 1, Row: "A", Number: 1},
				{ID: 2, Row: "A", Number: 2},
				{ID: 3, Row: "A", Number: 3},
				{ID: 4, Row: "A", Number: 4},
				{ID: 5, Row: "A", Number: 5},
				{ID: 6, Row: "B", Number: 1},
				{ID: 7, Row: "B", Number: 2},
				{ID: 8, Row: "B", Number: 3},
				{ID: 9, Row: "B", Number: 4},
				{ID: 10, Row: "B", Number: 5},
				{ID: 11, Row: "C", Number: 1},
			},
			2: {
				{ID: 101, Row: "A", Number: 1},
				{ID: 102, Row: "A", Number: 2},
			},
		},
		reservations: map[int64]domain.Reservation{},
		taken:        map[seatKey]int64{},
	}
}

func cloneReservations(src map[int64]domain.Reservation) map[int64]domain.Reservation {
	dst := make(map[int64]domain.Reservation, len(src))
	for k, v := range src {
		v.Seats = slices.Clone(v.Seats)
		dst[k] = v
	}
	return dst
}

func (m *memReservationRepo) ListByUser(_ context.Context, userID int64) ([]domain.Reservation, error) {
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

func (m *memReservationRepo) FindByID(_ context.Context, id int64) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r, ok := m.reservations[id]
	if !ok {
		return domain.Reservation{}, domain.ErrNotFound
	}
	r.Seats = slices.Clone(r.Seats)
	return r, nil
}

// WithTx executes fn against a snapshot and commits only when fn returns nil,
// mirroring PostgreSQL rollback semantics in unit tests.
func (m *memReservationRepo) WithTx(_ context.Context, fn func(ReservationTx) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	tx := &memReservationTx{
		nextID:       m.nextID,
		showtimes:    m.showtimes,
		auditoriums:  m.auditoriums,
		reservations: cloneReservations(m.reservations),
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

type memReservationTx struct {
	nextID       int64
	showtimes    map[int64]domain.ReservationShowtime
	auditoriums  map[int64][]domain.ReservedSeat
	reservations map[int64]domain.Reservation
	taken        map[seatKey]int64
}

func (t *memReservationTx) FindShowtime(_ context.Context, id int64) (domain.ReservationShowtime, error) {
	st, ok := t.showtimes[id]
	if !ok {
		return domain.ReservationShowtime{}, domain.ErrNotFound
	}
	return st, nil
}

func (t *memReservationTx) FindSeatsInAuditorium(_ context.Context, auditoriumID int64, seatIDs []int64) ([]domain.ReservedSeat, error) {
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

func (t *memReservationTx) FindReservationForUpdate(_ context.Context, id int64) (domain.Reservation, error) {
	r, ok := t.reservations[id]
	if !ok {
		return domain.Reservation{}, domain.ErrNotFound
	}
	r.Seats = slices.Clone(r.Seats)
	return r, nil
}

func (t *memReservationTx) InsertReservation(_ context.Context, userID, showtimeID int64, seatIDs []int64) (int64, time.Time, error) {
	t.nextID++
	id := t.nextID
	for _, sid := range seatIDs {
		k := seatKey{showtimeID: showtimeID, seatID: sid}
		if _, busy := t.taken[k]; busy {
			return 0, time.Time{}, domain.ErrSeatTaken
		}
		t.taken[k] = id
	}
	st := t.showtimes[showtimeID]
	seats, _ := t.FindSeatsInAuditorium(context.Background(), st.AuditoriumID, seatIDs)
	now := testNow
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

func (t *memReservationTx) ReplaceSeats(_ context.Context, reservationID, showtimeID int64, seatIDs []int64) error {
	for k, owner := range t.taken {
		if owner == reservationID {
			delete(t.taken, k)
		}
	}
	for _, sid := range seatIDs {
		k := seatKey{showtimeID: showtimeID, seatID: sid}
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

func (t *memReservationTx) CancelReservation(_ context.Context, reservationID int64) error {
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

func newTestReservationService() (*ReservationService, *memReservationRepo) {
	repo := newMemReservationRepo()
	svc := NewReservationService(repo)
	svc.now = func() time.Time { return testNow }
	return svc, repo
}

func TestReservationService_Create(t *testing.T) {
	t.Run("creates active reservation with total and sorted seats", func(t *testing.T) {
		svc, _ := newTestReservationService()
		res, err := svc.Create(context.Background(), 1, CreateReservationInput{
			ShowtimeID: 11,
			SeatIDs:    []int64{3, 1},
		})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if res.ID == 0 || res.Status != domain.ReservationActive || res.TotalCents != 16000 {
			t.Fatalf("unexpected reservation: %+v", res)
		}
		if len(res.Seats) != 2 || res.Seats[0].ID != 1 || res.Seats[1].ID != 3 {
			t.Errorf("seats = %+v, want [1, 3]", res.Seats)
		}
	})

	t.Run("allows up to 10 seats", func(t *testing.T) {
		svc, _ := newTestReservationService()
		res, err := svc.Create(context.Background(), 1, CreateReservationInput{
			ShowtimeID: 11,
			SeatIDs:    []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10},
		})
		if err != nil {
			t.Fatalf("Create 10 seats: %v", err)
		}
		if len(res.Seats) != 10 || res.TotalCents != 80000 {
			t.Fatalf("got %d seats, total %d", len(res.Seats), res.TotalCents)
		}
	})

	validationCases := []struct {
		name string
		in   CreateReservationInput
	}{
		{"non-positive showtimeId", CreateReservationInput{ShowtimeID: 0, SeatIDs: []int64{1}}},
		{"empty seatIds", CreateReservationInput{ShowtimeID: 11, SeatIDs: nil}},
		{"more than 10 seats", CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11}}},
		{"duplicate seatIds", CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{2, 2}}},
		{"non-positive seatId", CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 0}}},
		{"seat from another auditorium", CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 101}}},
		{"unknown seatId", CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{999}}},
	}
	for _, tc := range validationCases {
		t.Run(tc.name, func(t *testing.T) {
			svc, _ := newTestReservationService()
			_, err := svc.Create(context.Background(), 1, tc.in)
			var ve *domain.ValidationError
			if !errors.As(err, &ve) {
				t.Fatalf("err = %v, want *domain.ValidationError", err)
			}
		})
	}

	t.Run("unknown showtime is ErrNotFound", func(t *testing.T) {
		svc, _ := newTestReservationService()
		if _, err := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 999, SeatIDs: []int64{1}}); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("past or already-starting showtime is ErrShowtimeStarted", func(t *testing.T) {
		svc, _ := newTestReservationService()
		for _, id := range []int64{10, 12} {
			if _, err := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: id, SeatIDs: []int64{1}}); !errors.Is(err, domain.ErrShowtimeStarted) {
				t.Errorf("showtime %d: err = %v, want ErrShowtimeStarted", id, err)
			}
		}
	})

	t.Run("taken seat is ErrSeatTaken and rolls back completely", func(t *testing.T) {
		svc, repo := newTestReservationService()
		if _, err := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{2}}); err != nil {
			t.Fatal(err)
		}
		if _, err := svc.Create(context.Background(), 2, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2}}); !errors.Is(err, domain.ErrSeatTaken) {
			t.Fatalf("err = %v, want ErrSeatTaken", err)
		}
		// Seat 1 must still be free because user 2's transaction rolled back.
		if _, busy := repo.taken[seatKey{showtimeID: 11, seatID: 1}]; busy {
			t.Fatal("seat 1 should not remain taken after rollback")
		}
		list, _ := svc.List(context.Background(), 2)
		if len(list) != 0 {
			t.Fatalf("user 2 should have 0 reservations after rollback, got %d", len(list))
		}
	})
}

func TestReservationService_GetAndList_Ownership(t *testing.T) {
	svc, _ := newTestReservationService()
	r1, err := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Create(context.Background(), 2, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{2}}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.Get(context.Background(), 1, r1.ID)
	if err != nil || got.ID != r1.ID {
		t.Fatalf("Get own = %+v, %v", got, err)
	}
	if _, err := svc.Get(context.Background(), 2, r1.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get other user's reservation = %v, want ErrNotFound", err)
	}
	if _, err := svc.Get(context.Background(), 1, 999); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Get missing = %v, want ErrNotFound", err)
	}

	list1, err := svc.List(context.Background(), 1)
	if err != nil || len(list1) != 1 || list1[0].ID != r1.ID {
		t.Fatalf("List user 1 = %+v, %v", list1, err)
	}
}

func TestReservationService_UpdateSeats(t *testing.T) {
	t.Run("replaces seats atomically including overlapping seats", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, err := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2}})
		if err != nil {
			t.Fatal(err)
		}

		updated, err := svc.UpdateSeats(context.Background(), 1, r.ID, []int64{2, 3, 4})
		if err != nil {
			t.Fatalf("UpdateSeats: %v", err)
		}
		if len(updated.Seats) != 3 || updated.TotalCents != 24000 {
			t.Fatalf("updated = %+v", updated)
		}
		// Seat 1 was released so user 2 can reserve it now.
		if _, err := svc.Create(context.Background(), 2, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}}); err != nil {
			t.Fatalf("old seat 1 should be free: %v", err)
		}
	})

	t.Run("conflict rolls back and keeps previous seats intact", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r1, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2}})
		_, _ = svc.Create(context.Background(), 2, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{3}})

		if _, err := svc.UpdateSeats(context.Background(), 1, r1.ID, []int64{4, 3}); !errors.Is(err, domain.ErrSeatTaken) {
			t.Fatalf("err = %v, want ErrSeatTaken", err)
		}
		still, err := svc.Get(context.Background(), 1, r1.ID)
		if err != nil || len(still.Seats) != 2 || still.Seats[0].ID != 1 || still.Seats[1].ID != 2 {
			t.Fatalf("original seats must remain after rollback: %+v, %v", still, err)
		}
	})

	t.Run("another user's reservation returns ErrNotFound even after start", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
		svc.now = func() time.Time { return testNow.Add(3 * time.Hour) }

		if _, err := svc.UpdateSeats(context.Background(), 2, r.ID, []int64{2}); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound (must not leak showtime status)", err)
		}
	})

	t.Run("after showtime starts returns ErrShowtimeStarted", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
		svc.now = func() time.Time { return testNow.Add(3 * time.Hour) }

		if _, err := svc.UpdateSeats(context.Background(), 1, r.ID, []int64{2}); !errors.Is(err, domain.ErrShowtimeStarted) {
			t.Fatalf("err = %v, want ErrShowtimeStarted", err)
		}
	})

	t.Run("cancelled reservation cannot be modified", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
		if err := svc.Cancel(context.Background(), 1, r.ID); err != nil {
			t.Fatal(err)
		}

		_, err := svc.UpdateSeats(context.Background(), 1, r.ID, []int64{2})
		var ve *domain.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want *domain.ValidationError", err)
		}
	})

	t.Run("seat from another auditorium rejected", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})

		_, err := svc.UpdateSeats(context.Background(), 1, r.ID, []int64{101})
		var ve *domain.ValidationError
		if !errors.As(err, &ve) {
			t.Fatalf("err = %v, want *domain.ValidationError", err)
		}
	})
}

func TestReservationService_Cancel(t *testing.T) {
	t.Run("cancels active reservation, releases seats, and is idempotent", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2}})

		if err := svc.Cancel(context.Background(), 1, r.ID); err != nil {
			t.Fatalf("Cancel: %v", err)
		}
		got, err := svc.Get(context.Background(), 1, r.ID)
		if err != nil || got.Status != domain.ReservationCancelled || len(got.Seats) != 0 || got.TotalCents != 0 {
			t.Fatalf("after cancel = %+v, %v", got, err)
		}
		// Second cancel before startsAt is idempotent.
		if err := svc.Cancel(context.Background(), 1, r.ID); err != nil {
			t.Fatalf("second Cancel: %v", err)
		}
		// Released seats can now be booked by someone else.
		if _, err := svc.Create(context.Background(), 2, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1, 2}}); err != nil {
			t.Fatalf("seats should be free after cancel: %v", err)
		}
	})

	t.Run("another user's reservation returns ErrNotFound", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
		if err := svc.Cancel(context.Background(), 2, r.ID); !errors.Is(err, domain.ErrNotFound) {
			t.Fatalf("err = %v, want ErrNotFound", err)
		}
	})

	t.Run("after showtime starts returns ErrShowtimeStarted", func(t *testing.T) {
		svc, _ := newTestReservationService()
		r, _ := svc.Create(context.Background(), 1, CreateReservationInput{ShowtimeID: 11, SeatIDs: []int64{1}})
		svc.now = func() time.Time { return testNow.Add(3 * time.Hour) }

		if err := svc.Cancel(context.Background(), 1, r.ID); !errors.Is(err, domain.ErrShowtimeStarted) {
			t.Fatalf("err = %v, want ErrShowtimeStarted", err)
		}
	})
}
