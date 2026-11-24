-- +goose Up
CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE users (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    email         CITEXT      NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    name          TEXT        NOT NULL CHECK (length(name) BETWEEN 1 AND 100),
    role          TEXT        NOT NULL DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE movies (
    id           BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title        TEXT NOT NULL CHECK (length(title) BETWEEN 1 AND 200),
    synopsis     TEXT NOT NULL DEFAULT '',
    duration_min INT  NOT NULL CHECK (duration_min > 0),
    rating       TEXT NOT NULL DEFAULT '',
    poster_url   TEXT NOT NULL DEFAULT ''
);

CREATE TABLE auditoriums (
    id   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    rows INT  NOT NULL CHECK (rows BETWEEN 1 AND 26),
    cols INT  NOT NULL CHECK (cols BETWEEN 1 AND 50)
);

CREATE TABLE seats (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    auditorium_id BIGINT NOT NULL REFERENCES auditoriums (id) ON DELETE CASCADE,
    row_label     TEXT   NOT NULL,
    number        INT    NOT NULL CHECK (number > 0),
    UNIQUE (auditorium_id, row_label, number)
);

CREATE TABLE showtimes (
    id            BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    movie_id      BIGINT      NOT NULL REFERENCES movies (id) ON DELETE RESTRICT,
    auditorium_id BIGINT      NOT NULL REFERENCES auditoriums (id) ON DELETE RESTRICT,
    starts_at     TIMESTAMPTZ NOT NULL,
    price_cents   INT         NOT NULL CHECK (price_cents >= 0)
);
CREATE INDEX idx_showtimes_movie_starts ON showtimes (movie_id, starts_at);
CREATE INDEX idx_showtimes_auditorium ON showtimes (auditorium_id);

CREATE TABLE reservations (
    id          BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    user_id     BIGINT      NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    showtime_id BIGINT      NOT NULL REFERENCES showtimes (id) ON DELETE RESTRICT,
    status      TEXT        NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'cancelled')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Target for the composite FK below: guarantees a reservation's seats
    -- always belong to the same showtime as the reservation itself.
    UNIQUE (id, showtime_id)
);
CREATE INDEX idx_reservations_user ON reservations (user_id);
CREATE INDEX idx_reservations_showtime ON reservations (showtime_id);

-- showtime_id is denormalized here on purpose: the UNIQUE constraint below is
-- what prevents double booking, atomically and race-free (see ADR-002).
CREATE TABLE reservation_seats (
    reservation_id BIGINT NOT NULL,
    showtime_id    BIGINT NOT NULL,
    seat_id        BIGINT NOT NULL REFERENCES seats (id) ON DELETE RESTRICT,
    PRIMARY KEY (reservation_id, seat_id),
    CONSTRAINT uq_reservation_seats_showtime_seat UNIQUE (showtime_id, seat_id),
    FOREIGN KEY (reservation_id, showtime_id)
        REFERENCES reservations (id, showtime_id) ON DELETE CASCADE
);
CREATE INDEX idx_reservation_seats_seat ON reservation_seats (seat_id);

-- +goose Down
DROP TABLE IF EXISTS reservation_seats;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS showtimes;
DROP TABLE IF EXISTS seats;
DROP TABLE IF EXISTS auditoriums;
DROP TABLE IF EXISTS movies;
DROP TABLE IF EXISTS users;
DROP EXTENSION IF EXISTS citext;
