package auth

import "time"

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
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func newSessionResponse(s Session) SessionResponse {
	return SessionResponse{
		ID:           s.ID.String(),
		UserAgent:    s.UserAgent,
		IPAddress:    s.IPAddress,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
		IsCurrent:    s.IsCurrent,
	}
}
