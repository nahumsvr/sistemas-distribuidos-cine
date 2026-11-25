package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"proyecto-cine/backend/internal/domain"
	"proyecto-cine/backend/internal/service"
)

type ReservationService interface {
	List(ctx context.Context, userID int64) ([]domain.Reservation, error)
	Get(ctx context.Context, userID, reservationID int64) (domain.Reservation, error)
	Create(ctx context.Context, userID int64, in service.CreateReservationInput) (domain.Reservation, error)
	UpdateSeats(ctx context.Context, userID, reservationID int64, seatIDs []int64) (domain.Reservation, error)
	Cancel(ctx context.Context, userID, reservationID int64) error
}

type reservationShowtimeJSON struct {
	ID         int64     `json:"id"`
	MovieTitle string    `json:"movieTitle"`
	StartsAt   time.Time `json:"startsAt"`
}

type reservedSeatJSON struct {
	ID     int64  `json:"id"`
	Row    string `json:"row"`
	Number int    `json:"number"`
}

type reservationJSON struct {
	ID         int64                    `json:"id"`
	Status     domain.ReservationStatus `json:"status"`
	Showtime   reservationShowtimeJSON  `json:"showtime"`
	Seats      []reservedSeatJSON       `json:"seats"`
	TotalCents int                      `json:"totalCents"`
	CreatedAt  time.Time                `json:"createdAt"`
}

func toReservationJSON(r domain.Reservation) reservationJSON {
	seats := make([]reservedSeatJSON, 0, len(r.Seats)) // non-nil: "[]", never "null"
	for _, s := range r.Seats {
		seats = append(seats, reservedSeatJSON{ID: s.ID, Row: s.Row, Number: s.Number})
	}
	return reservationJSON{
		ID:     r.ID,
		Status: r.Status,
		Showtime: reservationShowtimeJSON{
			ID:         r.Showtime.ID,
			MovieTitle: r.Showtime.MovieTitle,
			StartsAt:   r.Showtime.StartsAt.UTC(),
		},
		Seats:      seats,
		TotalCents: r.TotalCents,
		CreatedAt:  r.CreatedAt.UTC(),
	}
}

func listReservationsHandler(svc ReservationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFrom(r.Context())
		list, err := svc.List(r.Context(), claims.UserID)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		out := make([]reservationJSON, 0, len(list)) // non-nil: "[]", never "null"
		for _, res := range list {
			out = append(out, toReservationJSON(res))
		}
		writeJSON(w, http.StatusOK, out)
	}
}

func getReservationHandler(svc ReservationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		claims, _ := ClaimsFrom(r.Context())
		res, err := svc.Get(r.Context(), claims.UserID, id)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationJSON(res))
	}
}

func createReservationHandler(svc ReservationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			ShowtimeID int64   `json:"showtimeId"`
			SeatIDs    []int64 `json:"seatIds"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		claims, _ := ClaimsFrom(r.Context())
		res, err := svc.Create(r.Context(), claims.UserID, service.CreateReservationInput{
			ShowtimeID: body.ShowtimeID,
			SeatIDs:    body.SeatIDs,
		})
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusCreated, toReservationJSON(res))
	}
}

func updateReservationHandler(svc ReservationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		var body struct {
			SeatIDs []int64 `json:"seatIds"`
		}
		if !decodeJSON(w, r, &body) {
			return
		}
		claims, _ := ClaimsFrom(r.Context())
		res, err := svc.UpdateSeats(r.Context(), claims.UserID, id, body.SeatIDs)
		if err != nil {
			writeServiceError(w, logger, err)
			return
		}
		writeJSON(w, http.StatusOK, toReservationJSON(res))
	}
}

func cancelReservationHandler(svc ReservationService, logger *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := pathID(w, r)
		if !ok {
			return
		}
		claims, _ := ClaimsFrom(r.Context())
		if err := svc.Cancel(r.Context(), claims.UserID, id); err != nil {
			writeServiceError(w, logger, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
