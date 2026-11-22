package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"gin-template-v2/internal/platform/dbgen"
)

// Service encapsulates authentication and session management logic.
type Service interface {
	LoginWithProvider(ctx context.Context, provider string, token string, clientInfo ClientInfo) (rawToken string, session Session, user User, err error)
	LoginWithPassword(ctx context.Context, email, password string, clientInfo ClientInfo) (rawToken string, session Session, user User, err error)
	ValidateSession(ctx context.Context, rawToken string) (user User, session Session, err error)
	Logout(ctx context.Context, rawToken string) error
	RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error
	RevokeAllSessions(ctx context.Context, userID uuid.UUID) error
	RevokeOtherSessions(ctx context.Context, rawToken string, userID uuid.UUID) error
	ListSessions(ctx context.Context, userID uuid.UUID, rawToken string) ([]Session, error)
}

// service talks to sqlc's generated Querier directly. Querier is already an
// interface, so it can be mocked in tests; a separate Repository layer would
// only forward calls. Use pool.Begin(ctx) / q.WithTx(tx) when you need a transaction.
type service struct {
	q               dbgen.Querier
	pool            *pgxpool.Pool
	providers       *ProviderRegistry
	sessionDuration time.Duration
}

// NewService constructs an auth Service.
func NewService(q dbgen.Querier, pool *pgxpool.Pool, providers *ProviderRegistry, sessionDuration time.Duration) Service {
	if sessionDuration <= 0 {
		sessionDuration = 7 * 24 * time.Hour
	}
	return &service{
		q:               q,
		pool:            pool,
		providers:       providers,
		sessionDuration: sessionDuration,
	}
}

func (s *service) findOrCreateUserByIdentity(ctx context.Context, profile *OAuthProfile) (User, error) {
	// 1. Try to find user by linked identity
	row, err := s.q.GetUserByIdentity(ctx, dbgen.GetUserByIdentityParams{
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
	})
	if err == nil {
		// Update avatar or name if profile has newer info
		if profile.AvatarURL != nil || profile.Name != "" {
			updated, updateErr := s.q.UpdateUserAvatarAndName(ctx, dbgen.UpdateUserAvatarAndNameParams{
				ID:        row.ID,
				AvatarUrl: profile.AvatarURL,
				Name:      &profile.Name,
			})
			if updateErr == nil {
				return updated, nil
			}
		}
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	// 2. Identity not found: link to existing user or create user + identity
	qtx := s.q
	var tx pgx.Tx
	if s.pool != nil {
		var txErr error
		tx, txErr = s.pool.Begin(ctx)
		if txErr != nil {
			return User{}, txErr
		}
		defer func() { _ = tx.Rollback(ctx) }()
		qtx = dbgen.New(tx)
	}

	// Check if user with this email already exists
	existingUser, err := qtx.GetUserByEmail(ctx, profile.Email)
	if err == nil {
		_, err = qtx.CreateIdentity(ctx, dbgen.CreateIdentityParams{
			UserID:         existingUser.ID,
			Provider:       profile.Provider,
			ProviderUserID: profile.ProviderUserID,
		})
		if err != nil {
			return User{}, err
		}
		if tx != nil {
			if err := tx.Commit(ctx); err != nil {
				return User{}, err
			}
		}
		return existingUser, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return User{}, err
	}

	// User does not exist, create new user
	newUser, err := qtx.CreateOAuthUser(ctx, dbgen.CreateOAuthUserParams{
		Email:     profile.Email,
		Name:      profile.Name,
		AvatarUrl: profile.AvatarURL,
	})
	if err != nil {
		return User{}, err
	}

	// Create identity
	_, err = qtx.CreateIdentity(ctx, dbgen.CreateIdentityParams{
		UserID:         newUser.ID,
		Provider:       profile.Provider,
		ProviderUserID: profile.ProviderUserID,
	})
	if err != nil {
		return User{}, err
	}

	if tx != nil {
		if err := tx.Commit(ctx); err != nil {
			return User{}, err
		}
	}

	return newUser, nil
}

func (s *service) LoginWithProvider(ctx context.Context, provider string, token string, clientInfo ClientInfo) (string, Session, User, error) {
	verifier, err := s.providers.Get(provider)
	if err != nil {
		return "", Session{}, User{}, err
	}

	profile, err := verifier.Verify(ctx, token)
	if err != nil {
		return "", Session{}, User{}, err
	}

	user, err := s.findOrCreateUserByIdentity(ctx, profile)
	if err != nil {
		return "", Session{}, User{}, err
	}

	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return "", Session{}, User{}, err
	}

	var userAgent *string
	if ua := strings.TrimSpace(clientInfo.UserAgent); ua != "" {
		userAgent = &ua
	}

	var ipAddr *string
	if ip := strings.TrimSpace(clientInfo.IPAddress); ip != "" {
		ipAddr = &ip
	}

	expiresAt := time.Now().Add(s.sessionDuration)
	sess, err := s.q.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IpAddress: ipAddr,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", Session{}, User{}, err
	}

	return rawToken, sess, user, nil
}

func (s *service) LoginWithPassword(ctx context.Context, email, password string, clientInfo ClientInfo) (string, Session, User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	user, err := s.q.GetUserWithPasswordByEmail(ctx, email)
	if err != nil {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	if user.PasswordHash == nil || *user.PasswordHash == "" {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.PasswordHash), []byte(password)); err != nil {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	rawToken, tokenHash, err := generateSessionToken()
	if err != nil {
		return "", Session{}, User{}, err
	}

	var userAgent *string
	if ua := strings.TrimSpace(clientInfo.UserAgent); ua != "" {
		userAgent = &ua
	}

	var ipAddr *string
	if ip := strings.TrimSpace(clientInfo.IPAddress); ip != "" {
		ipAddr = &ip
	}

	expiresAt := time.Now().Add(s.sessionDuration)
	sess, err := s.q.CreateSession(ctx, dbgen.CreateSessionParams{
		UserID:    user.ID,
		TokenHash: tokenHash,
		UserAgent: userAgent,
		IpAddress: ipAddr,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return "", Session{}, User{}, err
	}

	return rawToken, sess, user, nil
}

func (s *service) ValidateSession(ctx context.Context, rawToken string) (User, Session, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return User{}, Session{}, ErrUnauthorized
	}

	tokenHash := HashToken(rawToken)
	row, err := s.q.GetSessionAndUserByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return User{}, Session{}, ErrSessionNotFound
		}
		return User{}, Session{}, err
	}

	sess := Session{
		ID:           row.SessionID,
		UserID:       row.UserID,
		TokenHash:    row.TokenHash,
		UserAgent:    row.UserAgent,
		IpAddress:    row.IpAddress,
		ExpiresAt:    row.ExpiresAt,
		CreatedAt:    row.SessionCreatedAt,
		LastActiveAt: row.LastActiveAt,
	}

	user := User{
		ID:        row.UserPk,
		Email:     row.Email,
		Name:      row.Name,
		AvatarUrl: row.AvatarUrl,
		IsAdmin:   row.IsAdmin,
		CreatedAt: row.UserCreatedAt,
		UpdatedAt: row.UserUpdatedAt,
	}

	if sess.ExpiresAt.Before(time.Now()) {
		_, _ = s.q.DeleteSessionByTokenHash(ctx, tokenHash)
		return User{}, Session{}, ErrSessionExpired
	}

	// Throttle updating last_active_at (e.g. once every 5 minutes)
	if time.Since(sess.LastActiveAt) > 5*time.Minute {
		_ = s.q.TouchSession(ctx, sess.ID)
	}

	return user, sess, nil
}

func (s *service) Logout(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil
	}
	_, err := s.q.DeleteSessionByTokenHash(ctx, HashToken(rawToken))
	return err
}

func (s *service) RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	n, err := s.q.DeleteSessionByID(ctx, dbgen.DeleteSessionByIDParams{
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

func (s *service) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	_, err := s.q.DeleteAllSessionsForUser(ctx, userID)
	return err
}

func (s *service) RevokeOtherSessions(ctx context.Context, rawToken string, userID uuid.UUID) error {
	_, err := s.q.DeleteOtherSessionsForUser(ctx, dbgen.DeleteOtherSessionsForUserParams{
		UserID:    userID,
		TokenHash: HashToken(rawToken),
	})
	return err
}

func (s *service) ListSessions(ctx context.Context, userID uuid.UUID, rawToken string) ([]Session, error) {
	return s.q.ListActiveSessionsForUser(ctx, userID)
}

func generateSessionToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(b)
	return rawToken, HashToken(rawToken), nil
}

func HashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
