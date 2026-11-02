package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"gin-template-v2/internal/features/auth/authdb"
	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"
)

func parseUUID(c *gin.Context, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param(param))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "ID must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrInvalidToken):
		httpx.Error(c, http.StatusUnauthorized, "invalid_token", err.Error())
	case errors.Is(err, ErrInvalidCredentials):
		httpx.Error(c, http.StatusUnauthorized, "invalid_credentials", "Invalid email or password")
	case errors.Is(err, ErrUnsupportedProvider):
		httpx.Error(c, http.StatusBadRequest, "unsupported_provider", err.Error())
	case errors.Is(err, ErrEmailNotVerified):
		httpx.Error(c, http.StatusForbidden, "email_not_verified", "OAuth account email must be verified")
	case errors.Is(err, ErrSessionNotFound), errors.Is(err, ErrSessionExpired), errors.Is(err, ErrUnauthorized):
		httpx.Error(c, http.StatusUnauthorized, "unauthorized", "Unauthorized")
	case errors.Is(err, ErrUserNotFound):
		httpx.Error(c, http.StatusNotFound, "user_not_found", "User not found")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_error", "Invalid input")
	default:
		logger.FromContext(c.Request.Context()).Error("auth: unexpected error", "error", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}

type userRowType interface {
	authdb.GetUserByIdentityRow | authdb.GetUserByEmailRow | authdb.CreateOAuthUserRow | authdb.UpdateUserAvatarAndNameRow
}

func toUserFromRow[T userRowType](r T) User {
	u := authdb.GetUserByIdentityRow(r)
	return User{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarUrl,
		IsAdmin:   u.IsAdmin,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}

func sessionRowToSession(s authdb.Session) Session {
	return Session{
		ID:           s.ID,
		UserID:       s.UserID,
		TokenHash:    s.TokenHash,
		UserAgent:    s.UserAgent,
		IPAddress:    s.IpAddress,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
	}
}

func listSessionRowToSession(s authdb.Session, currentTokenHash string) Session {
	return Session{
		ID:           s.ID,
		UserID:       s.UserID,
		TokenHash:    s.TokenHash,
		UserAgent:    s.UserAgent,
		IPAddress:    s.IpAddress,
		ExpiresAt:    s.ExpiresAt,
		CreatedAt:    s.CreatedAt,
		LastActiveAt: s.LastActiveAt,
		IsCurrent:    s.TokenHash == currentTokenHash,
	}
}

func mapDBError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrSessionNotFound
	}
	return err
}
