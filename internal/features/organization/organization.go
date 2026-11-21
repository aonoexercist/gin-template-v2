package organization

import (
	"errors"
	"time"

	"gin-template-v2/internal/platform/dbgen"

	"github.com/google/uuid"
)

// Organization is the sqlc row type. No hand-written copy, no mapper to keep
// in sync: add a column in the migration + query.sql and it appears here.
type Organization = dbgen.Organization

var (
	ErrNotFound          = errors.New("organization not found")
	ErrOrganizationTaken = errors.New("organization already in use")
	ErrInvalidInput      = errors.New("invalid input")
)

// ---- HTTP DTOs ----

type CreateOrganizationRequest struct {
	Name string `json:"name"`
}

type OrganizationResponse struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func newOrganizationResponse(o Organization) OrganizationResponse {
	return OrganizationResponse{ID: o.ID, Name: o.Name, CreatedAt: o.CreatedAt, UpdatedAt: o.UpdatedAt}
}
