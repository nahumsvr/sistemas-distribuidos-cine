-- Development seed data. Demo credentials are documented in README.md;
-- change or remove these users before any real deployment.

-- +goose Up
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- pgcrypto's bf algorithm produces standard $2a$ bcrypt hashes, compatible
-- with golang.org/x/crypto/bcrypt. Cost 12 matches the API.
INSERT INTO users (email, password_hash, name, role) VALUES
    ('admin@cine.local', crypt('Admin123!', gen_salt('bf', 12)), 'Admin', 'admin'),
    ('user@cine.local',  crypt('User123!',  gen_salt('bf', 12)), 'Demo User', 'user');

INSERT INTO auditoriums (name, rows, cols) VALUES
    ('Sala 1', 8, 10),
    ('Sala 2', 6, 8);

-- One seat per (row, column): rows are labeled A, B, C...
INSERT INTO seats (auditorium_id, row_label, number)
SELECT a.id, chr(64 + r), c
FROM auditoriums a
CROSS JOIN LATERAL generate_series(1, a.rows) AS r
CROSS JOIN LATERAL generate_series(1, a.cols) AS c;

INSERT INTO movies (title, synopsis, duration_min, rating, poster_url) VALUES
    ('Dune: Part Two', 'Paul Atreides unites with the Fremen to seek revenge.', 166, 'B', ''),
    ('Inside Out 2', 'Riley''s mind makes room for new emotions.', 96, 'A', ''),
    ('Interstellar', 'A team travels through a wormhole in search of a new home.', 169, 'B', ''),
    ('Coco', 'Miguel journeys to the Land of the Dead.', 105, 'A', '');

-- Two showtimes per movie per day for the next 7 days, without overlaps:
-- movies 1-2 (by insertion order) play in Sala 1, movies 3-4 in Sala 2;
-- odd movies at 14:00 and 20:00, even movies at 17:00 and 23:00.
-- Uses row_number() instead of raw ids so it also works after a down/up.
INSERT INTO showtimes (movie_id, auditorium_id, starts_at, price_cents)
SELECT m.id,
       (SELECT id FROM auditoriums WHERE name = CASE WHEN m.n <= 2 THEN 'Sala 1' ELSE 'Sala 2' END),
       date_trunc('day', now()) + make_interval(days => d)
           + make_interval(hours => CASE WHEN m.n % 2 = 1 THEN 14 ELSE 17 END + slot * 6),
       CASE WHEN slot = 0 THEN 7000 ELSE 8500 END
FROM (SELECT id, row_number() OVER (ORDER BY id) AS n FROM movies) AS m
CROSS JOIN generate_series(1, 7) AS d
CROSS JOIN generate_series(0, 1) AS slot;

-- +goose Down
DELETE FROM reservation_seats;
DELETE FROM reservations;
DELETE FROM showtimes;
DELETE FROM movies;
DELETE FROM seats;
DELETE FROM auditoriums;
DELETE FROM users WHERE email IN ('admin@cine.local', 'user@cine.local');
DROP EXTENSION IF EXISTS pgcrypto;
