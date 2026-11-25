package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

const pgForeignKeyViolation = "23503"

type ReservationRepo struct{ pool *pgxpool.Pool }

func NewReservationRepo(pool *pgxpool.Pool) *ReservationRepo { return &ReservationRepo{pool: pool} }

const reservationSelect = `
	SELECT r.id, r.user_id, r.status, r.created_at,
	       st.id, st.auditorium_id, m.title, st.starts_at, st.price_cents
	FROM reservations r
	JOIN showtimes st ON st.id = r.showtime_id
	JOIN movies m ON m.id = st.movie_id`

func scanReservationHeader(row pgx.Row) (domain.Reservation, error) {
	var res domain.Reservation
	err := row.Scan(
		&res.ID, &res.UserID, &res.Status, &res.CreatedAt,
		&res.Showtime.ID, &res.Showtime.AuditoriumID, &res.Showtime.MovieTitle,
		&res.Showtime.StartsAt, &res.Showtime.PriceCents,
	)
	return res, err
}

func (r *ReservationRepo) ListByUser(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	rows, err := r.pool.Query(ctx,
		reservationSelect+` WHERE r.user_id = $1 ORDER BY r.created_at DESC, r.id DESC`,
		userID)
	if err != nil {
		return nil, fmt.Errorf("list reservations: %w", err)
	}
	defer rows.Close()

	var out []domain.Reservation
	ids := make([]int64, 0)
	byID := make(map[int64]int)
	for rows.Next() {
		res, err := scanReservationHeader(rows)
		if err != nil {
			return nil, fmt.Errorf("scan reservation: %w", err)
		}
		byID[res.ID] = len(out)
		ids = append(ids, res.ID)
		out = append(out, res)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return out, nil
	}

	// Fetch all seats for the listed reservations in one batch query (no N+1).
	seatRows, err := r.pool.Query(ctx, `
		SELECT rs.reservation_id, se.id, se.row_label, se.number
		FROM reservation_seats rs
		JOIN seats se ON se.id = rs.seat_id
		WHERE rs.reservation_id = ANY($1)
		ORDER BY se.row_label, se.number`,
		ids)
	if err != nil {
		return nil, fmt.Errorf("list reservation seats: %w", err)
	}
	defer seatRows.Close()

	for seatRows.Next() {
		var resID int64
		var seat domain.ReservedSeat
		if err := seatRows.Scan(&resID, &seat.ID, &seat.Row, &seat.Number); err != nil {
			return nil, fmt.Errorf("scan reservation seat: %w", err)
		}
		idx := byID[resID]
		out[idx].Seats = append(out[idx].Seats, seat)
	}
	if err := seatRows.Err(); err != nil {
		return nil, err
	}

	for i := range out {
		out[i].TotalCents = len(out[i].Seats) * out[i].Showtime.PriceCents
	}
	return out, nil
}

func (r *ReservationRepo) FindByID(ctx context.Context, id int64) (domain.Reservation, error) {
	res, err := scanReservationHeader(r.pool.QueryRow(ctx, reservationSelect+` WHERE r.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("find reservation: %w", err)
	}

	seatRows, err := r.pool.Query(ctx, `
		SELECT se.id, se.row_label, se.number
		FROM reservation_seats rs
		JOIN seats se ON se.id = rs.seat_id
		WHERE rs.reservation_id = $1
		ORDER BY se.row_label, se.number`,
		res.ID)
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("find reservation seats: %w", err)
	}
	defer seatRows.Close()

	for seatRows.Next() {
		var seat domain.ReservedSeat
		if err := seatRows.Scan(&seat.ID, &seat.Row, &seat.Number); err != nil {
			return domain.Reservation{}, fmt.Errorf("scan reservation seat: %w", err)
		}
		res.Seats = append(res.Seats, seat)
	}
	if err := seatRows.Err(); err != nil {
		return domain.Reservation{}, err
	}

	res.TotalCents = len(res.Seats) * res.Showtime.PriceCents
	return res, nil
}

func (r *ReservationRepo) WithTx(ctx context.Context, fn func(service.ReservationTx) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin reservation tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op after Commit

	if err := fn(&pgReservationTx{tx: tx}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit reservation tx: %w", err)
	}
	return nil
}

type pgReservationTx struct{ tx pgx.Tx }

func (t *pgReservationTx) FindShowtime(ctx context.Context, id int64) (domain.ReservationShowtime, error) {
	var st domain.ReservationShowtime
	err := t.tx.QueryRow(ctx, `
		SELECT st.id, st.auditorium_id, m.title, st.starts_at, st.price_cents
		FROM showtimes st
		JOIN movies m ON m.id = st.movie_id
		WHERE st.id = $1`,
		id,
	).Scan(&st.ID, &st.AuditoriumID, &st.MovieTitle, &st.StartsAt, &st.PriceCents)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ReservationShowtime{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.ReservationShowtime{}, fmt.Errorf("find showtime: %w", err)
	}
	return st, nil
}

func (t *pgReservationTx) FindSeatsInAuditorium(ctx context.Context, auditoriumID int64, seatIDs []int64) ([]domain.ReservedSeat, error) {
	rows, err := t.tx.Query(ctx, `
		SELECT id, row_label, number
		FROM seats
		WHERE auditorium_id = $1 AND id = ANY($2)
		ORDER BY row_label, number`,
		auditoriumID, seatIDs)
	if err != nil {
		return nil, fmt.Errorf("find seats: %w", err)
	}
	defer rows.Close()

	var out []domain.ReservedSeat
	for rows.Next() {
		var seat domain.ReservedSeat
		if err := rows.Scan(&seat.ID, &seat.Row, &seat.Number); err != nil {
			return nil, fmt.Errorf("scan seat: %w", err)
		}
		out = append(out, seat)
	}
	return out, rows.Err()
}

func (t *pgReservationTx) FindReservationForUpdate(ctx context.Context, id int64) (domain.Reservation, error) {
	res, err := scanReservationHeader(t.tx.QueryRow(ctx, reservationSelect+` WHERE r.id = $1 FOR UPDATE OF r`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Reservation{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Reservation{}, fmt.Errorf("lock reservation: %w", err)
	}
	return res, nil
}

func (t *pgReservationTx) InsertReservation(ctx context.Context, userID, showtimeID int64, seatIDs []int64) (int64, time.Time, error) {
	var id int64
	var createdAt time.Time
	err := t.tx.QueryRow(ctx,
		`INSERT INTO reservations (user_id, showtime_id) VALUES ($1, $2) RETURNING id, created_at`,
		userID, showtimeID,
	).Scan(&id, &createdAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return 0, time.Time{}, domain.ErrNotFound
		}
		return 0, time.Time{}, fmt.Errorf("insert reservation: %w", err)
	}
	if err := t.insertSeats(ctx, id, showtimeID, seatIDs); err != nil {
		return 0, time.Time{}, err
	}
	return id, createdAt, nil
}

func (t *pgReservationTx) ReplaceSeats(ctx context.Context, reservationID, showtimeID int64, seatIDs []int64) error {
	if _, err := t.tx.Exec(ctx, `DELETE FROM reservation_seats WHERE reservation_id = $1`, reservationID); err != nil {
		return fmt.Errorf("clear reservation seats: %w", err)
	}
	if err := t.insertSeats(ctx, reservationID, showtimeID, seatIDs); err != nil {
		return err
	}
	if _, err := t.tx.Exec(ctx, `UPDATE reservations SET updated_at = now() WHERE id = $1`, reservationID); err != nil {
		return fmt.Errorf("touch reservation: %w", err)
	}
	return nil
}

func (t *pgReservationTx) CancelReservation(ctx context.Context, reservationID int64) error {
	if _, err := t.tx.Exec(ctx, `DELETE FROM reservation_seats WHERE reservation_id = $1`, reservationID); err != nil {
		return fmt.Errorf("delete reservation seats: %w", err)
	}
	if _, err := t.tx.Exec(ctx,
		`UPDATE reservations SET status = 'cancelled', updated_at = now() WHERE id = $1`,
		reservationID); err != nil {
		return fmt.Errorf("cancel reservation: %w", err)
	}
	return nil
}

func (t *pgReservationTx) insertSeats(ctx context.Context, reservationID, showtimeID int64, seatIDs []int64) error {
	_, err := t.tx.Exec(ctx, `
		INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id)
		SELECT $1, $2, seat_id FROM unnest($3::bigint[]) AS seat_id`,
		reservationID, showtimeID, seatIDs)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == pgUniqueViolation {
		return domain.ErrSeatTaken
	}
	if err != nil {
		return fmt.Errorf("insert reservation seats: %w", err)
	}
	return nil
}
