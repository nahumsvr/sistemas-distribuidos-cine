package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"proyecto-cine/backend/internal/domain"
)

type CatalogRepo struct{ pool *pgxpool.Pool }

func NewCatalogRepo(pool *pgxpool.Pool) *CatalogRepo { return &CatalogRepo{pool: pool} }

const movieColumns = "id, title, synopsis, duration_min, rating, poster_url"

func scanMovie(row pgx.Row) (domain.Movie, error) {
	var m domain.Movie
	err := row.Scan(&m.ID, &m.Title, &m.Synopsis, &m.DurationMin, &m.Rating, &m.PosterURL)
	return m, err
}

func (r *CatalogRepo) ListMovies(ctx context.Context) ([]domain.Movie, error) {
	rows, err := r.pool.Query(ctx, `SELECT `+movieColumns+` FROM movies ORDER BY title, id`)
	if err != nil {
		return nil, fmt.Errorf("list movies: %w", err)
	}
	defer rows.Close()

	var out []domain.Movie
	for rows.Next() {
		m, err := scanMovie(rows)
		if err != nil {
			return nil, fmt.Errorf("scan movie: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *CatalogRepo) FindMovie(ctx context.Context, id int64) (domain.Movie, error) {
	m, err := scanMovie(r.pool.QueryRow(ctx, `SELECT `+movieColumns+` FROM movies WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Movie{}, domain.ErrNotFound
	}
	return m, err
}

const showtimeSelect = `
	SELECT s.id, s.movie_id, a.name, s.starts_at, s.price_cents
	FROM showtimes s
	JOIN auditoriums a ON a.id = s.auditorium_id`

func scanShowtime(row pgx.Row) (domain.Showtime, error) {
	var s domain.Showtime
	err := row.Scan(&s.ID, &s.MovieID, &s.Auditorium, &s.StartsAt, &s.PriceCents)
	return s, err
}

func (r *CatalogRepo) UpcomingShowtimes(ctx context.Context, movieID int64, after time.Time) ([]domain.Showtime, error) {
	rows, err := r.pool.Query(ctx,
		showtimeSelect+` WHERE s.movie_id = $1 AND s.starts_at > $2 ORDER BY s.starts_at, s.id`,
		movieID, after)
	if err != nil {
		return nil, fmt.Errorf("list showtimes: %w", err)
	}
	defer rows.Close()

	var out []domain.Showtime
	for rows.Next() {
		s, err := scanShowtime(rows)
		if err != nil {
			return nil, fmt.Errorf("scan showtime: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *CatalogRepo) FindShowtime(ctx context.Context, id int64) (domain.Showtime, error) {
	s, err := scanShowtime(r.pool.QueryRow(ctx, showtimeSelect+` WHERE s.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Showtime{}, domain.ErrNotFound
	}
	return s, err
}

// availabilitySQL answers the whole seat map in ONE query: every seat of the
// showtime's auditorium, flagged as taken when a reservation_seats row exists
// for (showtime, seat). Cancelling deletes those rows, so cancelled seats show
// up as free. Served by uq_reservation_seats_showtime_seat.
const availabilitySQL = `
	SELECT se.id, se.row_label, se.number,
	       NOT EXISTS (
	           SELECT 1 FROM reservation_seats rs
	           WHERE rs.showtime_id = st.id AND rs.seat_id = se.id
	       ) AS available
	FROM showtimes st
	JOIN seats se ON se.auditorium_id = st.auditorium_id
	WHERE st.id = $1
	ORDER BY se.row_label, se.number`

func (r *CatalogRepo) SeatsWithAvailability(ctx context.Context, showtimeID int64) ([]domain.Seat, error) {
	rows, err := r.pool.Query(ctx, availabilitySQL, showtimeID)
	if err != nil {
		return nil, fmt.Errorf("list seats: %w", err)
	}
	defer rows.Close()

	var out []domain.Seat
	for rows.Next() {
		var s domain.Seat
		if err := rows.Scan(&s.ID, &s.Row, &s.Number, &s.Available); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
