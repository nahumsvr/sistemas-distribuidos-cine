package domain

import (
	"errors"
	"time"
)

type ReservationStatus string

const (
	ReservationActive    ReservationStatus = "active"
	ReservationCancelled ReservationStatus = "cancelled"
)

var (
	ErrSeatTaken       = errors.New("one or more seats are already reserved")
	ErrShowtimeStarted = errors.New("showtime has already started")
)

// ReservationShowtime is the showtime summary embedded in a Reservation.
type ReservationShowtime struct {
	ID           int64
	AuditoriumID int64
	MovieTitle   string
	StartsAt     time.Time
	PriceCents   int
}

// ReservedSeat is a physical seat assigned to a reservation.
type ReservedSeat struct {
	ID     int64
	Row    string
	Number int
}

type Reservation struct {
	ID         int64
	UserID     int64
	Status     ReservationStatus
	Showtime   ReservationShowtime
	Seats      []ReservedSeat
	TotalCents int
	CreatedAt  time.Time
}
