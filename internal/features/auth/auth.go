package auth

import (
	"errors"
	"time"

	"github.com/aonoexercist/gin-template-v2/internal/platform/dbgen"
)

// User is the sqlc row type. No hand-written copy, no mapper to keep
// in sync: add a column in the migration + query.sql and it appears here.
type User = dbgen.User

// Session is the sqlc row type.
type Session = dbgen.Session

// ClientInfo encapsulates metadata from the client request.
type ClientInfo struct {
	UserAgent string
	IPAddress string
}

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

// ---- HTTP DTOs ----

// OAuthLoginRequest carries the token passed from the frontend (e.g. Google ID token).
type OAuthLoginRequest struct {
	Token string `json:"token" binding:"required"`
}

// EmailLoginRequest carries credentials for traditional email and password login.
type EmailLoginRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Password string `json:"password" binding:"required"`
}

// UserResponse is the client-facing user representation.
type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	AvatarURL *string   `json:"avatar_url,omitempty"`
	IsAdmin   bool      `json:"is_admin"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// SessionResponse is the client-facing session representation.
type SessionResponse struct {
	ID           string    `json:"id"`
	UserAgent    *string   `json:"user_agent,omitempty"`
	IPAddress    *string   `json:"ip_address,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastActiveAt time.Time `json:"last_active_at"`
	IsCurrent    bool      `json:"is_current"`
}

// AuthResponse returns both user and session information upon login.
type AuthResponse struct {
	User    UserResponse    `json:"user"`
	Session SessionResponse `json:"session"`
}

func newUserResponse(u User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarUrl,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func newSessionResponse(s Session, isCurrent bool) SessionResponse {
	return SessionResponse{
		ID:           s.ID.String(),
		UserAgent:    s.UserAgent,
		IPAddress:    s.IpAddress,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
		IsCurrent:    isCurrent,
	}
}
