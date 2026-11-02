package auth

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrUnsupportedProvider = errors.New("unsupported authentication provider")
	ErrEmailNotVerified    = errors.New("provider email is not verified")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrSessionNotFound     = errors.New("session not found")
	ErrSessionExpired      = errors.New("session expired")
	ErrUnauthorized        = errors.New("unauthorized")
	ErrUserNotFound        = errors.New("user not found")
	ErrInvalidInput        = errors.New("invalid input")
)

// User represents the domain user model for auth.
type User struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Session represents a client session tied to a user.
type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	TokenHash    string    `json:"-"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	IsCurrent    bool      `json:"is_current"`
}

// ClientInfo encapsulates metadata from the client request.
type ClientInfo struct {
	UserAgent string
	IPAddress string
}
