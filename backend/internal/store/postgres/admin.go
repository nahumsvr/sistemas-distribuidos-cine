package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"proyecto-cine/backend/internal/domain"
)

type AdminRepo struct{ pool *pgxpool.Pool }

func NewAdminRepo(pool *pgxpool.Pool) *AdminRepo { return &AdminRepo{pool: pool} }

func (r *AdminRepo) ListAuditoriums(ctx context.Context) ([]domain.Auditorium, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name, rows, cols FROM auditoriums ORDER BY name, id`)
	if err != nil {
		return nil, fmt.Errorf("list auditoriums: %w", err)
	}
	defer rows.Close()

	var out []domain.Auditorium
	for rows.Next() {
		var a domain.Auditorium
		if err := rows.Scan(&a.ID, &a.Name, &a.Rows, &a.Cols); err != nil {
			return nil, fmt.Errorf("scan auditorium: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *AdminRepo) CreateMovie(ctx context.Context, m domain.Movie) (domain.Movie, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO movies (title, synopsis, duration_min, rating, poster_url)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING `+movieColumns,
		m.Title, m.Synopsis, m.DurationMin, m.Rating, m.PosterURL)
	created, err := scanMovie(row)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("create movie: %w", err)
	}
	return created, nil
}

func (r *AdminRepo) UpdateMovie(ctx context.Context, m domain.Movie) (domain.Movie, error) {
	row := r.pool.QueryRow(ctx, `
		UPDATE movies
		SET title = $2, synopsis = $3, duration_min = $4, rating = $5, poster_url = $6
		WHERE id = $1
		RETURNING `+movieColumns,
		m.ID, m.Title, m.Synopsis, m.DurationMin, m.Rating, m.PosterURL)
	updated, err := scanMovie(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Movie{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Movie{}, fmt.Errorf("update movie: %w", err)
	}
	return updated, nil
}

// DeleteMovie deletes a movie and its unused showtimes inside a transaction.
// Any showtime with active reservations aborts with ErrHasActiveReservations.
func (r *AdminRepo) DeleteMovie(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete movie tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var lockedID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM movies WHERE id = $1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("lock movie: %w", err)
	}

	var hasActive bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM reservations r
			JOIN showtimes s ON s.id = r.showtime_id
			WHERE s.movie_id = $1 AND r.status = 'active'
		)`, id).Scan(&hasActive); err != nil {
		return fmt.Errorf("check movie reservations: %w", err)
	}
	if hasActive {
		return domain.ErrHasActiveReservations
	}

	// Remove historical cancelled reservations before deleting showtimes so
	// reservations.showtime_id ON DELETE RESTRICT succeeds cleanly.
	if _, err := tx.Exec(ctx, `
		DELETE FROM reservations
		WHERE status = 'cancelled'
		  AND showtime_id IN (SELECT id FROM showtimes WHERE movie_id = $1)`, id); err != nil {
		return fmt.Errorf("delete cancelled movie reservations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM showtimes WHERE movie_id = $1`, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return domain.ErrHasActiveReservations
		}
		return fmt.Errorf("delete movie showtimes: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM movies WHERE id = $1`, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return domain.ErrHasActiveReservations
		}
		return fmt.Errorf("delete movie: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete movie tx: %w", err)
	}
	return nil
}

func scanAdminShowtime(row pgx.Row) (domain.Showtime, error) {
	var s domain.Showtime
	err := row.Scan(&s.ID, &s.MovieID, &s.AuditoriumID, &s.Auditorium, &s.StartsAt, &s.PriceCents)
	return s, err
}

func (r *AdminRepo) CreateShowtime(ctx context.Context, s domain.Showtime) (domain.Showtime, error) {
	row := r.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO showtimes (movie_id, auditorium_id, starts_at, price_cents)
			VALUES ($1, $2, $3, $4)
			RETURNING id, movie_id, auditorium_id, starts_at, price_cents
		)
		SELECT i.id, i.movie_id, i.auditorium_id, a.name, i.starts_at, i.price_cents
		FROM inserted i
		JOIN auditoriums a ON a.id = i.auditorium_id`,
		s.MovieID, s.AuditoriumID, s.StartsAt, s.PriceCents)
	created, err := scanAdminShowtime(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return domain.Showtime{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Showtime{}, fmt.Errorf("create showtime: %w", err)
	}
	return created, nil
}

// UpdateShowtime updates a showtime inside a transaction. Changing the
// auditorium is rejected with ErrHasActiveReservations if active reservations
// already reference seats in the current auditorium.
func (r *AdminRepo) UpdateShowtime(ctx context.Context, s domain.Showtime) (domain.Showtime, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return domain.Showtime{}, fmt.Errorf("begin update showtime tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var currentAuditoriumID int64
	if err := tx.QueryRow(ctx, `SELECT auditorium_id FROM showtimes WHERE id = $1 FOR UPDATE`, s.ID).Scan(&currentAuditoriumID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Showtime{}, domain.ErrNotFound
		}
		return domain.Showtime{}, fmt.Errorf("lock showtime: %w", err)
	}

	if currentAuditoriumID != s.AuditoriumID {
		var hasActive bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM reservations
				WHERE showtime_id = $1 AND status = 'active'
			)`, s.ID).Scan(&hasActive); err != nil {
			return domain.Showtime{}, fmt.Errorf("check showtime reservations: %w", err)
		}
		if hasActive {
			return domain.Showtime{}, domain.ErrHasActiveReservations
		}
	}

	row := tx.QueryRow(ctx, `
		WITH updated AS (
			UPDATE showtimes
			SET movie_id = $2, auditorium_id = $3, starts_at = $4, price_cents = $5
			WHERE id = $1
			RETURNING id, movie_id, auditorium_id, starts_at, price_cents
		)
		SELECT u.id, u.movie_id, u.auditorium_id, a.name, u.starts_at, u.price_cents
		FROM updated u
		JOIN auditoriums a ON a.id = u.auditorium_id`,
		s.ID, s.MovieID, s.AuditoriumID, s.StartsAt, s.PriceCents)
	updated, err := scanAdminShowtime(row)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
		return domain.Showtime{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Showtime{}, fmt.Errorf("update showtime: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return domain.Showtime{}, fmt.Errorf("commit update showtime tx: %w", err)
	}
	return updated, nil
}

// DeleteShowtime deletes a showtime inside a transaction, failing with
// ErrHasActiveReservations if any active reservation exists.
func (r *AdminRepo) DeleteShowtime(ctx context.Context, id int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete showtime tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	var lockedID int64
	if err := tx.QueryRow(ctx, `SELECT id FROM showtimes WHERE id = $1 FOR UPDATE`, id).Scan(&lockedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.ErrNotFound
		}
		return fmt.Errorf("lock showtime: %w", err)
	}

	var hasActive bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM reservations
			WHERE showtime_id = $1 AND status = 'active'
		)`, id).Scan(&hasActive); err != nil {
		return fmt.Errorf("check showtime reservations: %w", err)
	}
	if hasActive {
		return domain.ErrHasActiveReservations
	}

	if _, err := tx.Exec(ctx, `DELETE FROM reservations WHERE showtime_id = $1 AND status = 'cancelled'`, id); err != nil {
		return fmt.Errorf("delete cancelled showtime reservations: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM showtimes WHERE id = $1`, id); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return domain.ErrHasActiveReservations
		}
		return fmt.Errorf("delete showtime: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete showtime tx: %w", err)
	}
	return nil
}
