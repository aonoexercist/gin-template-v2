package auth

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"gin-template-v2/internal/features/auth/authdb"
)

// Repository defines the persistence operations for authentication and sessions.
type Repository interface {
	FindOrCreateUserByIdentity(ctx context.Context, profile *OAuthProfile) (User, error)
	GetUserWithPasswordByEmail(ctx context.Context, email string) (User, *string, error)
	CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, userAgent, ipAddress *string, expiresAt time.Time) (Session, error)
	GetSessionAndUserByTokenHash(ctx context.Context, tokenHash string) (Session, User, error)
	TouchSession(ctx context.Context, sessionID uuid.UUID) error
	DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error
	DeleteSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error
	DeleteOtherSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) error
	DeleteAllSessions(ctx context.Context, userID uuid.UUID) error
	ListActiveSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) ([]Session, error)
}

type pgRepository struct {
	pool *pgxpool.Pool
	q    *authdb.Queries
}

// NewRepository returns a Postgres-backed Repository.
func NewRepository(pool *pgxpool.Pool) Repository {
	return &pgRepository{
		pool: pool,
		q:    authdb.New(pool),
	}
}

func (r *pgRepository) FindOrCreateUserByIdentity(ctx context.Context, profile *OAuthProfile) (User, error) {
	// 1. Try to find user by linked identity
	row, err := r.q.GetUserByIdentity(ctx, authdb.GetUserByIdentityParams{
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
	})
	if err == nil {
		// Update avatar or name if profile has newer info
		if profile.AvatarURL != nil || profile.Name != "" {
			updated, updateErr := r.q.UpdateUserAvatarAndName(ctx, authdb.UpdateUserAvatarAndNameParams{
				ID:        row.ID,
				AvatarUrl: profile.AvatarURL,
				Name:      &profile.Name,
			})
			if updateErr == nil {
				return toUserFromRow(updated), nil
			}
		}
		return toUserFromRow(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	// 2. Identity not found: start transaction to link to existing user or create user + identity atomically
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return User{}, err
	}
	defer tx.Rollback(ctx)

	qtx := r.q.WithTx(tx)

	// Check if user with this email already exists
	existingUser, err := qtx.GetUserByEmail(ctx, profile.Email)
	if err == nil {
		// Link identity to existing user
		_, err = qtx.CreateIdentity(ctx, authdb.CreateIdentityParams{
			UserID:         existingUser.ID,
			Provider:       profile.Provider,
			ProviderUserID: profile.ProviderUserID,
		})
		if err != nil {
			return User{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return User{}, err
		}
		return toUserFromRow(existingUser), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	// User does not exist, create new user
	newUser, err := qtx.CreateOAuthUser(ctx, authdb.CreateOAuthUserParams{
		Email:     profile.Email,
		Name:      profile.Name,
		AvatarUrl: profile.AvatarURL,
	})
	if err != nil {
		return User{}, err
	}

	// Create identity
	_, err = qtx.CreateIdentity(ctx, authdb.CreateIdentityParams{
		UserID:         newUser.ID,
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
	})
	if err != nil {
		return User{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return User{}, err
	}

	return toUserFromRow(newUser), nil
}

func (r *pgRepository) GetUserWithPasswordByEmail(ctx context.Context, email string) (User, *string, error) {
	row, err := r.q.GetUserWithPasswordByEmail(ctx, email)
	if err != nil {
		return User{}, nil, mapDBError(err)
	}
	u := User{
		ID:        row.ID,
		Email:     row.Email,
		Name:      row.Name,
		AvatarURL: row.AvatarUrl,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
	return u, row.PasswordHash, nil
}

func (r *pgRepository) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, userAgent, ipAddress *string, expiresAt time.Time) (Session, error) {
	row, err := r.q.CreateSession(ctx, authdb.CreateSessionParams{
		UserID:    userID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IpAddress: ipAddress,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return Session{}, err
	}
	return sessionRowToSession(row), nil
}

func (r *pgRepository) GetSessionAndUserByTokenHash(ctx context.Context, tokenHash string) (Session, User, error) {
	row, err := r.q.GetSessionAndUserByTokenHash(ctx, tokenHash)
	if err != nil {
		return Session{}, User{}, mapDBError(err)
	}

	sess := Session{
		ID:           row.SessionID,
		UserID:       row.UserID,
		TokenHash:    row.TokenHash,
		UserAgent:    row.UserAgent,
		IPAddress:    row.IpAddress,
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.SessionCreatedAt,
		LastActiveAt: row.LastActiveAt,
	}

	user := User{
		ID:        row.UserPk,
		Email:     row.Email,
		Name:      row.Name,
		AvatarURL: row.AvatarUrl,
		CreatedAt: row.UserCreatedAt,
		UpdatedAt: row.UserUpdatedAt,
	}

	return sess, user, nil
}

func (r *pgRepository) TouchSession(ctx context.Context, sessionID uuid.UUID) error {
	return r.q.TouchSession(ctx, sessionID)
}

func (r *pgRepository) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.q.DeleteSessionByTokenHash(ctx, tokenHash)
	return err
}

func (r *pgRepository) DeleteSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error {
	n, err := r.q.DeleteSessionByID(ctx, authdb.DeleteSessionByIDParams{
		ID:     sessionID,
		UserID: userID,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (r *pgRepository) DeleteOtherSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) error {
	_, err := r.q.DeleteOtherSessionsForUser(ctx, authdb.DeleteOtherSessionsForUserParams{
		UserID:    userID,
		TokenHash: currentTokenHash,
	})
	return err
}

func (r *pgRepository) DeleteAllSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := r.q.DeleteAllSessionsForUser(ctx, userID)
	return err
}

func (r *pgRepository) ListActiveSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) ([]Session, error) {
	rows, err := r.q.ListActiveSessionsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]Session, len(rows))
	for i, row := range rows {
		out[i] = listSessionRowToSession(row, currentTokenHash)
	}
	return out, nil
}
