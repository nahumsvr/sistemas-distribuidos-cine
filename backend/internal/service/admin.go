package service

import (
	"context"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"proyecto-cine/backend/internal/domain"
)

const (
	maxMovieTitleLen    = 200
	maxMovieSynopsisLen = 2000
	maxMovieRatingLen   = 20
	maxPosterURLLen     = 2048
)

type AdminRepository interface {
	ListAuditoriums(ctx context.Context) ([]domain.Auditorium, error)
	CreateMovie(ctx context.Context, m domain.Movie) (domain.Movie, error)
	UpdateMovie(ctx context.Context, m domain.Movie) (domain.Movie, error)
	DeleteMovie(ctx context.Context, id int64) error
	CreateShowtime(ctx context.Context, s domain.Showtime) (domain.Showtime, error)
	UpdateShowtime(ctx context.Context, s domain.Showtime) (domain.Showtime, error)
	DeleteShowtime(ctx context.Context, id int64) error
}

type MovieInput struct {
	Title       string
	Synopsis    string
	DurationMin int
	Rating      string
	PosterURL   string
}

type ShowtimeInput struct {
	MovieID      int64
	AuditoriumID int64
	StartsAt     time.Time
	PriceCents   int
}

type AdminService struct {
	repo AdminRepository
	now  func() time.Time
}

func NewAdminService(repo AdminRepository) *AdminService {
	return &AdminService{repo: repo, now: time.Now}
}

func (s *AdminService) ListAuditoriums(ctx context.Context) ([]domain.Auditorium, error) {
	return s.repo.ListAuditoriums(ctx)
}

func (s *AdminService) CreateMovie(ctx context.Context, in MovieInput) (domain.Movie, error) {
	m, err := validateMovieInput(in)
	if err != nil {
		return domain.Movie{}, err
	}
	return s.repo.CreateMovie(ctx, m)
}

func (s *AdminService) UpdateMovie(ctx context.Context, id int64, in MovieInput) (domain.Movie, error) {
	if id <= 0 {
		return domain.Movie{}, domain.Invalid("id must be a positive integer")
	}
	m, err := validateMovieInput(in)
	if err != nil {
		return domain.Movie{}, err
	}
	m.ID = id
	return s.repo.UpdateMovie(ctx, m)
}

func (s *AdminService) DeleteMovie(ctx context.Context, id int64) error {
	if id <= 0 {
		return domain.Invalid("id must be a positive integer")
	}
	return s.repo.DeleteMovie(ctx, id)
}

func (s *AdminService) CreateShowtime(ctx context.Context, in ShowtimeInput) (domain.Showtime, error) {
	st, err := s.validateShowtimeInput(in)
	if err != nil {
		return domain.Showtime{}, err
	}
	return s.repo.CreateShowtime(ctx, st)
}

func (s *AdminService) UpdateShowtime(ctx context.Context, id int64, in ShowtimeInput) (domain.Showtime, error) {
	if id <= 0 {
		return domain.Showtime{}, domain.Invalid("id must be a positive integer")
	}
	st, err := s.validateShowtimeInput(in)
	if err != nil {
		return domain.Showtime{}, err
	}
	st.ID = id
	return s.repo.UpdateShowtime(ctx, st)
}

func (s *AdminService) DeleteShowtime(ctx context.Context, id int64) error {
	if id <= 0 {
		return domain.Invalid("id must be a positive integer")
	}
	return s.repo.DeleteShowtime(ctx, id)
}

func validateMovieInput(in MovieInput) (domain.Movie, error) {
	title := strings.TrimSpace(in.Title)
	synopsis := strings.TrimSpace(in.Synopsis)
	rating := strings.TrimSpace(in.Rating)
	posterURL := strings.TrimSpace(in.PosterURL)

	var problems []string
	if n := utf8.RuneCountInString(title); n < 1 || n > maxMovieTitleLen {
		problems = append(problems, "title must have between 1 and 200 characters")
	}
	if utf8.RuneCountInString(synopsis) > maxMovieSynopsisLen {
		problems = append(problems, "synopsis must have at most 2000 characters")
	}
	if in.DurationMin <= 0 {
		problems = append(problems, "durationMin must be a positive integer")
	}
	if utf8.RuneCountInString(rating) > maxMovieRatingLen {
		problems = append(problems, "rating must have at most 20 characters")
	}
	if posterURL != "" {
		if len(posterURL) > maxPosterURLLen {
			problems = append(problems, "posterUrl is too long")
		} else if u, err := url.ParseRequestURI(posterURL); err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			problems = append(problems, "posterUrl must be a valid http or https URL")
		}
	}
	if len(problems) > 0 {
		return domain.Movie{}, domain.Invalid(strings.Join(problems, "; "))
	}
	return domain.Movie{
		Title:       title,
		Synopsis:    synopsis,
		DurationMin: in.DurationMin,
		Rating:      rating,
		PosterURL:   posterURL,
	}, nil
}

func (s *AdminService) validateShowtimeInput(in ShowtimeInput) (domain.Showtime, error) {
	var problems []string
	if in.MovieID <= 0 {
		problems = append(problems, "movieId must be a positive integer")
	}
	if in.AuditoriumID <= 0 {
		problems = append(problems, "auditoriumId must be a positive integer")
	}
	if in.StartsAt.IsZero() {
		problems = append(problems, "startsAt is required")
	} else if !in.StartsAt.After(s.now()) {
		problems = append(problems, "startsAt must be in the future")
	}
	if in.PriceCents < 0 {
		problems = append(problems, "priceCents must be zero or positive")
	}
	if len(problems) > 0 {
		return domain.Showtime{}, domain.Invalid(strings.Join(problems, "; "))
	}
	return domain.Showtime{
		MovieID:      in.MovieID,
		AuditoriumID: in.AuditoriumID,
		StartsAt:     in.StartsAt.UTC(),
		PriceCents:   in.PriceCents,
	}, nil
}
