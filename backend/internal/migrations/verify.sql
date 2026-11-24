-- Manual checks for the schema (run after migrating, against a throwaway DB).
-- Usage: psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f verify.sql
-- Not embedded by goose (it only picks up files named NNNNN_*.sql).
-- Everything runs inside a transaction that is rolled back at the end.

\echo '--- seed counts (expect: users 2, auditoriums 2, seats 128, movies 4, showtimes 56)'
SELECT (SELECT count(*) FROM users)       AS users,
       (SELECT count(*) FROM auditoriums) AS auditoriums,
       (SELECT count(*) FROM seats)       AS seats,
       (SELECT count(*) FROM movies)      AS movies,
       (SELECT count(*) FROM showtimes)   AS showtimes;

\echo '--- seeded password matches, email lookup is case-insensitive (expect: t)'
SELECT password_hash = crypt('User123!', password_hash) AS password_ok
FROM users WHERE email = 'USER@cine.local';

\echo '--- no overlapping showtimes in the same auditorium (expect: 0)'
SELECT count(*) AS overlaps
FROM showtimes a
JOIN movies ma ON ma.id = a.movie_id
JOIN showtimes b ON b.auditorium_id = a.auditorium_id AND b.id > a.id
JOIN movies mb ON mb.id = b.movie_id
WHERE tstzrange(a.starts_at, a.starts_at + make_interval(mins => ma.duration_min))
   && tstzrange(b.starts_at, b.starts_at + make_interval(mins => mb.duration_min));

BEGIN;

-- Fixture: the demo user books the first seat of the first showtime.
CREATE TEMP TABLE fx ON COMMIT DROP AS
SELECT st.id AS showtime_id,
       (SELECT min(s.id) FROM seats s WHERE s.auditorium_id = st.auditorium_id) AS seat_id,
       (SELECT min(id) FROM showtimes WHERE id <> st.id) AS other_showtime_id
FROM showtimes st ORDER BY st.id LIMIT 1;

WITH r AS (
    INSERT INTO reservations (user_id, showtime_id)
    SELECT u.id, fx.showtime_id FROM users u, fx WHERE u.email = 'user@cine.local'
    RETURNING id, showtime_id
)
INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id)
SELECT r.id, r.showtime_id, fx.seat_id FROM r, fx;

ALTER TABLE fx ADD COLUMN reservation_id BIGINT;
UPDATE fx SET reservation_id = (SELECT id FROM reservations WHERE showtime_id = fx.showtime_id);

\echo '--- double booking the same seat in the same showtime (expect: NOTICE OK)'
DO $$
DECLARE
    f fx%ROWTYPE;
    new_id BIGINT;
BEGIN
    SELECT * INTO f FROM fx;
    INSERT INTO reservations (user_id, showtime_id)
    SELECT id, f.showtime_id FROM users WHERE email = 'admin@cine.local'
    RETURNING id INTO new_id;
    INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id)
    VALUES (new_id, f.showtime_id, f.seat_id);
    RAISE EXCEPTION 'FAIL: double booking was accepted';
EXCEPTION WHEN unique_violation THEN
    RAISE NOTICE 'OK: double booking rejected';
END $$;

\echo '--- seat row pointing to a showtime different from its reservation (expect: NOTICE OK)'
DO $$
DECLARE
    f fx%ROWTYPE;
BEGIN
    SELECT * INTO f FROM fx;
    INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id)
    VALUES (f.reservation_id, f.other_showtime_id, f.seat_id + 1);
    RAISE EXCEPTION 'FAIL: mismatched showtime was accepted';
EXCEPTION WHEN foreign_key_violation THEN
    RAISE NOTICE 'OK: mismatched showtime rejected';
END $$;

\echo '--- cancelling frees the seat: delete seat rows, then rebook (expect: no error)'
DELETE FROM reservation_seats WHERE reservation_id = (SELECT reservation_id FROM fx);
UPDATE reservations SET status = 'cancelled', updated_at = now()
WHERE id = (SELECT reservation_id FROM fx);
WITH r AS (
    INSERT INTO reservations (user_id, showtime_id)
    SELECT u.id, fx.showtime_id FROM users u, fx WHERE u.email = 'admin@cine.local'
    RETURNING id, showtime_id
)
INSERT INTO reservation_seats (reservation_id, showtime_id, seat_id)
SELECT r.id, r.showtime_id, fx.seat_id FROM r, fx;

ROLLBACK;
