package users

import (
	"errors"
	"net/http"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-template-v2/internal/features/users/usersdb"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "ID must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// fail maps domain errors to HTTP responses; unknown errors are logged and
// returned as an opaque 500 so internals never leak to clients.
func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "User not found")
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(c, http.StatusConflict, "email_taken", "Email already in use")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_error", "Invalid input")
	default:
		logger.FromContext(c.Request.Context()).Error("users: unexpected error", "error", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

func toUser(u usersdb.User) User {
	return User{ID: u.ID, Email: u.Email, Name: u.Name, CreatedAt: u.CreatedAt, UpdatedAt: u.UpdatedAt}
}

// mapError translates driver errors into domain errors.
func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return ErrEmailTaken
	}
	return err
}

func normalizeEmail(e string) string { return strings.ToLower(strings.TrimSpace(e)) }
