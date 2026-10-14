package users

import (
	"time"

	"github.com/google/uuid"
)

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email,max=254"`
	Name     string `json:"name" binding:"required,min=1,max=100"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// UpdateUserRequest is a partial update: omitted fields are left unchanged.
type UpdateUserRequest struct {
	Email    *string `json:"email" binding:"omitempty,email,max=254"`
	Name     *string `json:"name" binding:"omitempty,min=1,max=100"`
	Password *string `json:"password" binding:"omitempty,min=8,max=100"`
}

type ListUsersQuery struct {
	Page    int `form:"page" binding:"omitempty,min=1,max=1000000"`
	PerPage int `form:"per_page" binding:"omitempty,min=1,max=100"`
}

type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newUserResponse(u User) UserResponse {
	return UserResponse{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}
