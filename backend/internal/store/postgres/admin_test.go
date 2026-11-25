package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

// Integration test against a real PostgreSQL instance (see testDatabaseURL).
func TestAdminRepo_MoviesShowtimesAndReservationProtection(t *testing.T) {
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

	adminRepo := NewAdminRepo(pool)
	adminSvc := service.NewAdminService(adminRepo)
	catalogRepo := NewCatalogRepo(pool)
	resSvc := service.NewReservationService(NewReservationRepo(pool))

	// Create a disposable user for active/cancelled reservation checks.
	var userID int64
	email := "admin-test-" + time.Now().Format("150405.000000") + "@mail.com"
	if err := pool.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, name, role) VALUES ($1, 'h', 'Tester', 'user') RETURNING id`,
		email).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	auditoriums, err := adminSvc.ListAuditoriums(ctx)
	if err != nil || len(auditoriums) < 2 {
		t.Fatalf("ListAuditoriums = %+v, %v; want >= 2", auditoriums, err)
	}

	// 1. Create and update a movie.
	mov, err := adminSvc.CreateMovie(ctx, service.MovieInput{
		Title:       "Admin Integration Movie",
		Synopsis:    "Test synopsis.",
		DurationMin: 120,
		Rating:      "B",
	})
	if err != nil {
		t.Fatalf("CreateMovie: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM reservations WHERE showtime_id IN (SELECT id FROM showtimes WHERE movie_id = $1)`, mov.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM showtimes WHERE movie_id = $1`, mov.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM movies WHERE id = $1`, mov.ID)
	})

	mov, err = adminSvc.UpdateMovie(ctx, mov.ID, service.MovieInput{
		Title:       "Admin Integration Movie v2",
		Synopsis:    "Updated.",
		DurationMin: 125,
		Rating:      "B15",
		PosterURL:   "https://example.com/p.jpg",
	})
	if err != nil || mov.Title != "Admin Integration Movie v2" || mov.DurationMin != 125 {
		t.Fatalf("UpdateMovie: %+v, %v", mov, err)
	}

	// 2. Create showtime with invalid FK -> ErrNotFound.
	future := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	if _, err := adminSvc.CreateShowtime(ctx, service.ShowtimeInput{
		MovieID: 999999, AuditoriumID: auditoriums[0].ID, StartsAt: future, PriceCents: 7000,
	}); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("unknown movieId err = %v, want ErrNotFound", err)
	}

	// 3. Create valid showtime and book a seat.
	st, err := adminSvc.CreateShowtime(ctx, service.ShowtimeInput{
		MovieID: mov.ID, AuditoriumID: auditoriums[0].ID, StartsAt: future, PriceCents: 7500,
	})
	if err != nil || st.Auditorium != auditoriums[0].Name {
		t.Fatalf("CreateShowtime: %+v, %v", st, err)
	}
	seats, err := catalogRepo.SeatsWithAvailability(ctx, st.ID)
	if err != nil || len(seats) == 0 {
		t.Fatalf("SeatsWithAvailability: %d, %v", len(seats), err)
	}
	res, err := resSvc.Create(ctx, userID, service.CreateReservationInput{
		ShowtimeID: st.ID,
		SeatIDs:    []int64{seats[0].ID},
	})
	if err != nil {
		t.Fatalf("Create reservation: %v", err)
	}

	// 4. While active reservation exists:
	//    - cannot delete showtime (409)
	//    - cannot delete movie (409)
	//    - cannot change auditorium (409)
	//    - CAN update price/time in the same auditorium
	if err := adminSvc.DeleteShowtime(ctx, st.ID); !errors.Is(err, domain.ErrHasActiveReservations) {
		t.Errorf("DeleteShowtime with active res = %v, want ErrHasActiveReservations", err)
	}
	if err := adminSvc.DeleteMovie(ctx, mov.ID); !errors.Is(err, domain.ErrHasActiveReservations) {
		t.Errorf("DeleteMovie with active res = %v, want ErrHasActiveReservations", err)
	}
	if _, err := adminSvc.UpdateShowtime(ctx, st.ID, service.ShowtimeInput{
		MovieID: mov.ID, AuditoriumID: auditoriums[1].ID, StartsAt: future, PriceCents: 7500,
	}); !errors.Is(err, domain.ErrHasActiveReservations) {
		t.Errorf("UpdateShowtime auditorium with active res = %v, want ErrHasActiveReservations", err)
	}
	st, err = adminSvc.UpdateShowtime(ctx, st.ID, service.ShowtimeInput{
		MovieID: mov.ID, AuditoriumID: auditoriums[0].ID, StartsAt: future.Add(time.Hour), PriceCents: 9000,
	})
	if err != nil || st.PriceCents != 9000 {
		t.Fatalf("UpdateShowtime price in same auditorium = %+v, %v", st, err)
	}

	// 5. After cancelling the reservation, deleting showtime succeeds cleanly.
	if err := resSvc.Cancel(ctx, userID, res.ID); err != nil {
		t.Fatalf("Cancel reservation: %v", err)
	}
	if err := adminSvc.DeleteShowtime(ctx, st.ID); err != nil {
		t.Fatalf("DeleteShowtime after cancel: %v", err)
	}

	// 6. Create another showtime with a cancelled reservation and delete the movie cleanly.
	st2, err := adminSvc.CreateShowtime(ctx, service.ShowtimeInput{
		MovieID: mov.ID, AuditoriumID: auditoriums[1].ID, StartsAt: future.Add(2 * time.Hour), PriceCents: 8000,
	})
	if err != nil {
		t.Fatal(err)
	}
	seats2, _ := catalogRepo.SeatsWithAvailability(ctx, st2.ID)
	res2, err := resSvc.Create(ctx, userID, service.CreateReservationInput{
		ShowtimeID: st2.ID, SeatIDs: []int64{seats2[0].ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := resSvc.Cancel(ctx, userID, res2.ID); err != nil {
		t.Fatal(err)
	}
	if err := adminSvc.DeleteMovie(ctx, mov.ID); err != nil {
		t.Fatalf("DeleteMovie with only cancelled reservations: %v", err)
	}
	if _, err := catalogRepo.FindMovie(ctx, mov.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("deleted movie still exists: err = %v", err)
	}
}
