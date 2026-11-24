// Package domain holds the core entities and business errors. It has no
// dependencies on HTTP, SQL or any framework.
package domain

import (
	"errors"
	"time"
)

type Role string

const (
	RoleUser  Role = "user"
	RoleAdmin Role = "admin"
)

func (r Role) Valid() bool { return r == RoleUser || r == RoleAdmin }

type User struct {
	ID           int64
	Email        string
	Name         string
	Role         Role
	PasswordHash string
	CreatedAt    time.Time
}

var (
	ErrNotFound           = errors.New("not found")
	ErrEmailTaken         = errors.New("email already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

// ValidationError carries a message that is safe to show to the client.
type ValidationError struct{ Message string }

func (e *ValidationError) Error() string { return e.Message }

func Invalid(msg string) error { return &ValidationError{Message: msg} }
