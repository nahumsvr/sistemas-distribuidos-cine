package postgres

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

// Integration test against a real PostgreSQL instance (see testDatabaseURL).
func TestReservationRepo_EndToEndAndConcurrency(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	pool, err := Connect(ctx, testDatabaseURL(t))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	// Create two disposable users and pick a future showtime with its seats.
	suffix := time.Now().Format("150405.000000")
	var u1, u2 int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role) VALUES ($1, 'h', 'U1', 'user') RETURNING id`,
		"res-u1-"+suffix+"@mail.com").Scan(&u1); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role) VALUES ($1, 'h', 'U2', 'user') RETURNING id`,
		"res-u2-"+suffix+"@mail.com").Scan(&u2); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id IN ($1, $2)`, u1, u2)
	})

	var showtimeID int64
	var priceCents int
	if err := pool.QueryRow(ctx,
		`SELECT id, price_cents FROM showtimes WHERE starts_at > now() + interval '1 hour' ORDER BY id LIMIT 1`,
	).Scan(&showtimeID, &priceCents); err != nil {
		t.Fatal(err)
	}

	catalog := NewCatalogRepo(pool)
	allSeats, err := catalog.SeatsWithAvailability(ctx, showtimeID)
	if err != nil || len(allSeats) < 5 {
		t.Fatalf("need at least 5 seats, got %d (err %v)", len(allSeats), err)
	}

	svc := service.NewReservationService(NewReservationRepo(pool))

	// 1. Create reservation with 2 seats.
	res, err := svc.Create(ctx, u1, service.CreateReservationInput{
		ShowtimeID: showtimeID,
		SeatIDs:    []int64{allSeats[1].ID, allSeats[0].ID},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.Status != domain.ReservationActive || len(res.Seats) != 2 || res.TotalCents != 2*priceCents {
		t.Fatalf("unexpected created reservation: %+v", res)
	}

	// 2. List & Get return the reservation with seats ordered.
	listed, err := svc.List(ctx, u1)
	if err != nil || len(listed) != 1 || len(listed[0].Seats) != 2 {
		t.Fatalf("List: %+v, %v", listed, err)
	}
	fetched, err := svc.Get(ctx, u1, res.ID)
	if err != nil || fetched.ID != res.ID || len(fetched.Seats) != 2 {
		t.Fatalf("Get: %+v, %v", fetched, err)
	}

	// 3. User 2 reserves seat[2]; user 1 tries to update to [seat[1], seat[2]] -> 409 and rollback.
	res2, err := svc.Create(ctx, u2, service.CreateReservationInput{
		ShowtimeID: showtimeID,
		SeatIDs:    []int64{allSeats[2].ID},
	})
	if err != nil {
		t.Fatalf("Create user 2: %v", err)
	}
	if _, err := svc.UpdateSeats(ctx, u1, res.ID, []int64{allSeats[1].ID, allSeats[2].ID}); !errors.Is(err, domain.ErrSeatTaken) {
		t.Fatalf("conflicting UpdateSeats err = %v, want ErrSeatTaken", err)
	}
	// Rollback preserved user 1's original seats [0, 1].
	afterRollback, err := svc.Get(ctx, u1, res.ID)
	if err != nil || len(afterRollback.Seats) != 2 || afterRollback.Seats[0].ID != allSeats[0].ID || afterRollback.Seats[1].ID != allSeats[1].ID {
		t.Fatalf("seats after failed update = %+v, %v", afterRollback.Seats, err)
	}

	// 4. Valid overlapping update [0, 1] -> [1, 3].
	updated, err := svc.UpdateSeats(ctx, u1, res.ID, []int64{allSeats[1].ID, allSeats[3].ID})
	if err != nil || len(updated.Seats) != 2 || updated.Seats[0].ID != allSeats[1].ID || updated.Seats[1].ID != allSeats[3].ID {
		t.Fatalf("UpdateSeats: %+v, %v", updated, err)
	}

	// 5. Cancel frees seats in CatalogRepo.SeatsWithAvailability while preserving history.
	if err := svc.Cancel(ctx, u1, res.ID); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	cancelled, err := svc.Get(ctx, u1, res.ID)
	if err != nil || cancelled.Status != domain.ReservationCancelled || len(cancelled.Seats) != 0 || cancelled.TotalCents != 0 {
		t.Fatalf("cancelled reservation = %+v, %v", cancelled, err)
	}
	seatsNow, err := catalog.SeatsWithAvailability(ctx, showtimeID)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range seatsNow {
		if (s.ID == allSeats[1].ID || s.ID == allSeats[3].ID) && !s.Available {
			t.Errorf("cancelled seat %d still reported unavailable", s.ID)
		}
		if s.ID == allSeats[2].ID && s.Available {
			t.Errorf("user 2's active seat %d reported available", s.ID)
		}
	}
	_ = svc.Cancel(ctx, u2, res2.ID)

	// 6. Concurrent double-booking: 8 goroutines race for the exact same seat;
	// PostgreSQL UNIQUE (showtime_id, seat_id) must allow exactly 1 and reject 7.
	const workers = 8
	targetSeat := allSeats[4].ID
	errs := make(chan error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := svc.Create(ctx, u1, service.CreateReservationInput{
				ShowtimeID: showtimeID,
				SeatIDs:    []int64{targetSeat},
			})
			errs <- err
		}()
	}
	close(start)
	wg.Wait()
	close(errs)

	var succeeded, seatTaken int
	for err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, domain.ErrSeatTaken):
			seatTaken++
		default:
			t.Errorf("unexpected concurrent error: %v", err)
		}
	}
	if succeeded != 1 || seatTaken != workers-1 {
		t.Fatalf("concurrent race: succeeded=%d, seatTaken=%d; want 1 and %d", succeeded, seatTaken, workers-1)
	}
}
