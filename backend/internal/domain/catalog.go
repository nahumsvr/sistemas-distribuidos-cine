package domain

import "time"

type Movie struct {
	ID          int64
	Title       string
	Synopsis    string
	DurationMin int
	Rating      string
	PosterURL   string
}

// Showtime is one screening of a movie in an auditorium.
type Showtime struct {
	ID           int64
	MovieID      int64
	AuditoriumID int64
	Auditorium   string
	StartsAt     time.Time
	PriceCents   int
}

// Seat is a physical seat plus whether it is free for one specific showtime.
type Seat struct {
	ID        int64
	Row       string
	Number    int
	Available bool
}
