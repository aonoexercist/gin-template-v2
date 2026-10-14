// Package users is a feature module: HTTP handler -> service -> repository -> sqlc.
// Copy this folder as the starting point for a new feature.
package users

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User is the domain model, decoupled from the sqlc row type and the JSON DTO.
type User struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

var (
	ErrNotFound     = errors.New("user not found")
	ErrEmailTaken   = errors.New("email already in use")
	ErrInvalidInput = errors.New("invalid input")
)
