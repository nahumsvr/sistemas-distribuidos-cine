package domain

import "errors"

var ErrHasActiveReservations = errors.New("resource has active reservations")

type Auditorium struct {
	ID   int64
	Name string
	Rows int
	Cols int
}
