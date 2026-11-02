package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
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

type service struct {
	repo            Repository
	providers       *ProviderRegistry
	sessionDuration time.Duration
}

// NewService constructs an auth Service.
func NewService(repo Repository, providers *ProviderRegistry, sessionDuration time.Duration) Service {
	if sessionDuration <= 0 {
		sessionDuration = 7 * 24 * time.Hour
	}
	return &service{
		repo:            repo,
		providers:       providers,
		sessionDuration: sessionDuration,
	}
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

	user, err := s.repo.FindOrCreateUserByIdentity(ctx, profile)
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
	sess, err := s.repo.CreateSession(ctx, user.ID, tokenHash, userAgent, ipAddr, expiresAt)
	if err != nil {
		return "", Session{}, User{}, err
	}
	sess.IsCurrent = true

	return rawToken, sess, user, nil
}

func (s *service) LoginWithPassword(ctx context.Context, email, password string, clientInfo ClientInfo) (string, Session, User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	user, passwordHash, err := s.repo.GetUserWithPasswordByEmail(ctx, email)
	if err != nil {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	if passwordHash == nil || *passwordHash == "" {
		return "", Session{}, User{}, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*passwordHash), []byte(password)); err != nil {
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
	sess, err := s.repo.CreateSession(ctx, user.ID, tokenHash, userAgent, ipAddr, expiresAt)
	if err != nil {
		return "", Session{}, User{}, err
	}
	sess.IsCurrent = true

	return rawToken, sess, user, nil
}

func (s *service) ValidateSession(ctx context.Context, rawToken string) (User, Session, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return User{}, Session{}, ErrUnauthorized
	}

	tokenHash := hashToken(rawToken)
	sess, user, err := s.repo.GetSessionAndUserByTokenHash(ctx, tokenHash)
	if err != nil {
		return User{}, Session{}, err
	}

	if sess.ExpiresAt.Before(time.Now()) {
		_ = s.repo.DeleteSessionByTokenHash(ctx, tokenHash)
		return User{}, Session{}, ErrSessionExpired
	}

	// Throttle updating last_active_at (e.g. once every 5 minutes)
	if time.Since(sess.LastActiveAt) > 5*time.Minute {
		_ = s.repo.TouchSession(ctx, sess.ID)
	}

	sess.IsCurrent = true
	return user, sess, nil
}

func (s *service) Logout(ctx context.Context, rawToken string) error {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return nil
	}
	return s.repo.DeleteSessionByTokenHash(ctx, hashToken(rawToken))
}

func (s *service) RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	return s.repo.DeleteSessionByID(ctx, sessionID, userID)
}

func (s *service) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return s.repo.DeleteAllSessions(ctx, userID)
}

func (s *service) RevokeOtherSessions(ctx context.Context, rawToken string, userID uuid.UUID) error {
	return s.repo.DeleteOtherSessions(ctx, userID, hashToken(rawToken))
}

func (s *service) ListSessions(ctx context.Context, userID uuid.UUID, rawToken string) ([]Session, error) {
	return s.repo.ListActiveSessions(ctx, userID, hashToken(rawToken))
}

func generateSessionToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	rawToken := base64.RawURLEncoding.EncodeToString(b)
	return rawToken, hashToken(rawToken), nil
}

func hashToken(rawToken string) string {
	sum := sha256.Sum256([]byte(rawToken))
	return hex.EncodeToString(sum[:])
}
