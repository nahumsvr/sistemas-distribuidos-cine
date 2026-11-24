// Package service contains the business rules. It depends on small interfaces
// (ports), never on concrete databases or HTTP.
package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"proyecto-cine/backend/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u domain.User) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByID(ctx context.Context, id int64) (domain.User, error)
}

type TokenIssuer interface {
	Issue(u domain.User) (string, error)
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type AuthResult struct {
	Token string
	User  domain.User
}

type AuthService struct {
	users     UserRepository
	tokens    TokenIssuer
	cost      int
	dummyHash []byte
}

// NewAuthService uses bcrypt with the given cost (12 in production; tests use
// bcrypt.MinCost to stay fast).
func NewAuthService(users UserRepository, tokens TokenIssuer, cost int) *AuthService {
	dummy, err := bcrypt.GenerateFromPassword([]byte("timing-equalizer"), cost)
	if err != nil {
		panic(fmt.Sprintf("bcrypt setup: %v", err))
	}
	return &AuthService{users: users, tokens: tokens, cost: cost, dummyHash: dummy}
}

func (s *AuthService) Register(ctx context.Context, in RegisterInput) (AuthResult, error) {
	name := strings.TrimSpace(in.Name)
	email := normalizeEmail(in.Email)

	if err := validateRegistration(name, email, in.Password); err != nil {
		return AuthResult{}, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.cost)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	// Role is fixed here: nobody can register as admin through the API.
	user, err := s.users.Create(ctx, domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         domain.RoleUser,
	})
	if err != nil {
		return AuthResult{}, err
	}
	return s.withToken(user)
}

// Login returns ErrInvalidCredentials for both an unknown email and a wrong
// password, and spends the same bcrypt time in both cases, so attackers cannot
// tell which emails are registered.
func (s *AuthService) Login(ctx context.Context, email, password string) (AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, normalizeEmail(email))
	if errors.Is(err, domain.ErrNotFound) {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(password))
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return AuthResult{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return AuthResult{}, domain.ErrInvalidCredentials
	}
	return s.withToken(user)
}

func (s *AuthService) Me(ctx context.Context, userID int64) (domain.User, error) {
	return s.users.FindByID(ctx, userID)
}

func (s *AuthService) withToken(u domain.User) (AuthResult, error) {
	tok, err := s.tokens.Issue(u)
	if err != nil {
		return AuthResult{}, fmt.Errorf("issue token: %w", err)
	}
	return AuthResult{Token: tok, User: u}, nil
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }

func validateRegistration(name, email, password string) error {
	var problems []string
	if n := utf8.RuneCountInString(name); n < 1 || n > 100 {
		problems = append(problems, "name must have between 1 and 100 characters")
	}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email || len(email) > 254 {
		problems = append(problems, "email is not valid")
	}
	// bcrypt only uses the first 72 bytes; reject longer passwords explicitly.
	if len(password) < 8 || len(password) > 72 {
		problems = append(problems, "password must have between 8 and 72 bytes")
	}
	if len(problems) > 0 {
		return domain.Invalid(strings.Join(problems, "; "))
	}
	return nil
}
