package service

import (
	"context"
	"time"

	"proyecto-cine/backend/internal/domain"
)

type CatalogRepository interface {
	ListMovies(ctx context.Context) ([]domain.Movie, error)
	FindMovie(ctx context.Context, id int64) (domain.Movie, error)
	// UpcomingShowtimes returns the movie's showtimes that start after the
	// given instant, ordered by start time.
	UpcomingShowtimes(ctx context.Context, movieID int64, after time.Time) ([]domain.Showtime, error)
	FindShowtime(ctx context.Context, id int64) (domain.Showtime, error)
	// SeatsWithAvailability returns every seat of the showtime's auditorium
	// in row/number order, flagged with whether it is still free.
	SeatsWithAvailability(ctx context.Context, showtimeID int64) ([]domain.Seat, error)
}

// CatalogService is the read-only, public side of the system.
type CatalogService struct {
	repo CatalogRepository
	now  func() time.Time
}

func NewCatalogService(repo CatalogRepository) *CatalogService {
	return &CatalogService{repo: repo, now: time.Now}
}

func (s *CatalogService) ListMovies(ctx context.Context) ([]domain.Movie, error) {
	return s.repo.ListMovies(ctx)
}

func (s *CatalogService) GetMovie(ctx context.Context, id int64) (domain.Movie, error) {
	return s.repo.FindMovie(ctx, id)
}

// MovieShowtimes returns only future showtimes. An unknown movie is
// ErrNotFound; a known movie without showtimes is an empty list.
func (s *CatalogService) MovieShowtimes(ctx context.Context, movieID int64) ([]domain.Showtime, error) {
	if _, err := s.repo.FindMovie(ctx, movieID); err != nil {
		return nil, err
	}
	return s.repo.UpcomingShowtimes(ctx, movieID, s.now())
}

func (s *CatalogService) ShowtimeSeats(ctx context.Context, showtimeID int64) (domain.Showtime, []domain.Seat, error) {
	st, err := s.repo.FindShowtime(ctx, showtimeID)
	if err != nil {
		return domain.Showtime{}, nil, err
	}
	seats, err := s.repo.SeatsWithAvailability(ctx, showtimeID)
	if err != nil {
		return domain.Showtime{}, nil, err
	}
	return st, seats, nil
}
