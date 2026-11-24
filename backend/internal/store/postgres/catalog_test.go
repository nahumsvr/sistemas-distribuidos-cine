package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
)

// Integration test against the seeded database (see testDatabaseURL).
func TestCatalogRepo_Seeded(t *testing.T) {
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
	repo := NewCatalogRepo(pool)

	movies, err := repo.ListMovies(ctx)
	if err != nil || len(movies) != 4 {
		t.Fatalf("ListMovies = %d movies, err %v; want 4", len(movies), err)
	}
	if _, err := repo.FindMovie(ctx, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("FindMovie(0) err = %v, want ErrNotFound", err)
	}

	// Only future showtimes, ascending, with the auditorium name joined in.
	now := time.Now()
	showtimes, err := repo.UpcomingShowtimes(ctx, movies[0].ID, now)
	if err != nil {
		t.Fatal(err)
	}
	for i, s := range showtimes {
		if !s.StartsAt.After(now) || s.Auditorium == "" {
			t.Errorf("showtime %d = %+v: want future with auditorium", i, s)
		}
		if i > 0 && s.StartsAt.Before(showtimes[i-1].StartsAt) {
			t.Errorf("showtimes not ordered by start time")
		}
	}
	far, err := repo.UpcomingShowtimes(ctx, movies[0].ID, now.AddDate(50, 0, 0))
	if err != nil || len(far) != 0 {
		t.Errorf("far future = %v, %v; want empty", far, err)
	}

	if _, err := repo.FindShowtime(ctx, 0); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("FindShowtime(0) err = %v, want ErrNotFound", err)
	}
}

func TestCatalogRepo_SeatAvailabilityFollowsReservations(t *testing.T) {
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
	repo := NewCatalogRepo(pool)

	// Work inside a transaction that is always rolled back, so the shared
	// test database is left untouched.
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var showtimeID, auditoriumSeats int64
	if err := tx.QueryRow(ctx, `
		SELECT s.id, (SELECT count(*) FROM seats WHERE auditorium_id = s.auditorium_id)
		FROM showtimes s ORDER BY s.id LIMIT 1`).Scan(&showtimeID, &auditoriumSeats); err != nil {
		t.Fatal(err)
	}

	seats, err := repo.SeatsWithAvailability(ctx, showtimeID)
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(seats)) != auditoriumSeats {
		t.Fatalf("seats = %d, want %d", len(seats), auditoriumSeats)
	}
	for _, s := range seats {
		if !s.Available {
			t.Fatalf("seed has no reservations; seat %+v should be free", s)
		}
	}
	if seats[0].Row != "A" || seats[0].Number != 1 {
		t.Errorf("first seat = %+v, want A1 (ordered by row, number)", seats[0])
	}

	var resID int64
	if err := tx.QueryRow(ctx,
		`INSERT INTO reservations (user_id, showtime_id) VALUES ((SELECT min(id) FROM users), $1) RETURNING id`,
		showtimeID).Scan(&resID); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id) VALUES ($1, $2, $3)`,
		resID, showtimeID, seats[0].ID); err != nil {
		t.Fatal(err)
	}

	// The repo reads through the pool, not the tx: it cannot see uncommitted
	// rows. Re-run the same availability query inside the tx to check the SQL.
	rows, err := tx.Query(ctx, availabilitySQL, showtimeID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	taken := 0
	for rows.Next() {
		var id int64
		var row string
		var number int
		var available bool
		if err := rows.Scan(&id, &row, &number, &available); err != nil {
			t.Fatal(err)
		}
		if !available {
			taken++
			if id != seats[0].ID {
				t.Errorf("seat %d reported taken, want only %d", id, seats[0].ID)
			}
		}
	}
	if taken != 1 {
		t.Errorf("taken = %d, want 1", taken)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM reservation_seats WHERE reservation_id = $1`, resID); err != nil {
		t.Fatal(err)
	}
	var stillTaken int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM (`+availabilitySQL+`) q WHERE NOT q.available`, showtimeID).Scan(&stillTaken); err != nil {
		t.Fatal(err)
	}
	if stillTaken != 0 {
		t.Errorf("after cancel, taken = %d, want 0", stillTaken)
	}
}
