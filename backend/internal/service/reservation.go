package service

import (
	"context"
	"slices"
	"time"

	"proyecto-cine/backend/internal/domain"
)

const maxSeatsPerReservation = 10

// ReservationRepository persists reservations and runs multi-step writes inside
// an atomic database transaction.
type ReservationRepository interface {
	ListByUser(ctx context.Context, userID int64) ([]domain.Reservation, error)
	FindByID(ctx context.Context, id int64) (domain.Reservation, error)
	WithTx(ctx context.Context, fn func(tx ReservationTx) error) error
}

// ReservationTx is the transactional surface used by ReservationService so
// business rules stay in the service layer while PostgreSQL enforces atomicity
// and the anti-double-booking UNIQUE constraint.
type ReservationTx interface {
	FindShowtime(ctx context.Context, id int64) (domain.ReservationShowtime, error)
	FindSeatsInAuditorium(ctx context.Context, auditoriumID int64, seatIDs []int64) ([]domain.ReservedSeat, error)
	FindReservationForUpdate(ctx context.Context, id int64) (domain.Reservation, error)
	InsertReservation(ctx context.Context, userID, showtimeID int64, seatIDs []int64) (id int64, createdAt time.Time, err error)
	ReplaceSeats(ctx context.Context, reservationID, showtimeID int64, seatIDs []int64) error
	CancelReservation(ctx context.Context, reservationID int64) error
}

type CreateReservationInput struct {
	ShowtimeID int64
	SeatIDs    []int64
}

type ReservationService struct {
	repo ReservationRepository
	now  func() time.Time
}

func NewReservationService(repo ReservationRepository) *ReservationService {
	return &ReservationService{repo: repo, now: time.Now}
}

func (s *ReservationService) List(ctx context.Context, userID int64) ([]domain.Reservation, error) {
	return s.repo.ListByUser(ctx, userID)
}

// Get returns a reservation only if it belongs to userID; another user's
// reservation is reported as ErrNotFound so existence is never leaked.
func (s *ReservationService) Get(ctx context.Context, userID, reservationID int64) (domain.Reservation, error) {
	res, err := s.repo.FindByID(ctx, reservationID)
	if err != nil {
		return domain.Reservation{}, err
	}
	if res.UserID != userID {
		return domain.Reservation{}, domain.ErrNotFound
	}
	return res, nil
}

func (s *ReservationService) Create(ctx context.Context, userID int64, in CreateReservationInput) (domain.Reservation, error) {
	if in.ShowtimeID <= 0 {
		return domain.Reservation{}, domain.Invalid("showtimeId must be a positive integer")
	}
	seatIDs, err := validateAndSortSeatIDs(in.SeatIDs)
	if err != nil {
		return domain.Reservation{}, err
	}

	var created domain.Reservation
	err = s.repo.WithTx(ctx, func(tx ReservationTx) error {
		st, err := tx.FindShowtime(ctx, in.ShowtimeID)
		if err != nil {
			return err
		}
		if !st.StartsAt.After(s.now()) {
			return domain.ErrShowtimeStarted
		}

		seats, err := tx.FindSeatsInAuditorium(ctx, st.AuditoriumID, seatIDs)
		if err != nil {
			return err
		}
		if len(seats) != len(seatIDs) {
			return domain.Invalid("one or more seats do not belong to this showtime's auditorium")
		}

		id, createdAt, err := tx.InsertReservation(ctx, userID, st.ID, seatIDs)
		if err != nil {
			return err
		}

		created = domain.Reservation{
			ID:         id,
			UserID:     userID,
			Status:     domain.ReservationActive,
			Showtime:   st,
			Seats:      seats,
			TotalCents: len(seats) * st.PriceCents,
			CreatedAt:  createdAt,
		}
		return nil
	})
	if err != nil {
		return domain.Reservation{}, err
	}
	return created, nil
}

func (s *ReservationService) UpdateSeats(ctx context.Context, userID, reservationID int64, seatIDs []int64) (domain.Reservation, error) {
	sortedIDs, err := validateAndSortSeatIDs(seatIDs)
	if err != nil {
		return domain.Reservation{}, err
	}

	var updated domain.Reservation
	err = s.repo.WithTx(ctx, func(tx ReservationTx) error {
		res, err := tx.FindReservationForUpdate(ctx, reservationID)
		if err != nil {
			return err
		}
		if res.UserID != userID {
			return domain.ErrNotFound
		}
		if !res.Showtime.StartsAt.After(s.now()) {
			return domain.ErrShowtimeStarted
		}
		if res.Status != domain.ReservationActive {
			return domain.Invalid("cancelled reservation cannot be modified")
		}

		seats, err := tx.FindSeatsInAuditorium(ctx, res.Showtime.AuditoriumID, sortedIDs)
		if err != nil {
			return err
		}
		if len(seats) != len(sortedIDs) {
			return domain.Invalid("one or more seats do not belong to this showtime's auditorium")
		}

		if err := tx.ReplaceSeats(ctx, res.ID, res.Showtime.ID, sortedIDs); err != nil {
			return err
		}

		res.Seats = seats
		res.TotalCents = len(seats) * res.Showtime.PriceCents
		updated = res
		return nil
	})
	if err != nil {
		return domain.Reservation{}, err
	}
	return updated, nil
}

func (s *ReservationService) Cancel(ctx context.Context, userID, reservationID int64) error {
	return s.repo.WithTx(ctx, func(tx ReservationTx) error {
		res, err := tx.FindReservationForUpdate(ctx, reservationID)
		if err != nil {
			return err
		}
		if res.UserID != userID {
			return domain.ErrNotFound
		}
		if !res.Showtime.StartsAt.After(s.now()) {
			return domain.ErrShowtimeStarted
		}
		if res.Status == domain.ReservationCancelled {
			return nil
		}
		return tx.CancelReservation(ctx, res.ID)
	})
}

func validateAndSortSeatIDs(seatIDs []int64) ([]int64, error) {
	if len(seatIDs) == 0 || len(seatIDs) > maxSeatsPerReservation {
		return nil, domain.Invalid("seatIds must contain between 1 and 10 seats")
	}
	sorted := make([]int64, len(seatIDs))
	copy(sorted, seatIDs)
	slices.Sort(sorted)
	for i, id := range sorted {
		if id <= 0 {
			return nil, domain.Invalid("seatIds must contain only positive integers")
		}
		if i > 0 && id == sorted[i-1] {
			return nil, domain.Invalid("seatIds must not contain duplicates")
		}
	}
	return sorted, nil
}
