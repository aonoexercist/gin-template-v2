package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"gin-template-v2/internal/features/auth"
)

type mockRepo struct {
	users     map[uuid.UUID]auth.User
	passwords map[uuid.UUID]string
	sessions  map[string]auth.Session // key is tokenHash
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		users:     make(map[uuid.UUID]auth.User),
		passwords: make(map[uuid.UUID]string),
		sessions:  make(map[string]auth.Session),
	}
}

func (m *mockRepo) GetUserWithPasswordByEmail(ctx context.Context, email string) (auth.User, *string, error) {
	for id, u := range m.users {
		if u.Email == email {
			if pw, ok := m.passwords[id]; ok {
				return u, &pw, nil
			}
			return u, nil, nil
		}
	}
	return auth.User{}, nil, auth.ErrUserNotFound
}

func (m *mockRepo) FindOrCreateUserByIdentity(ctx context.Context, profile *auth.OAuthProfile) (auth.User, error) {
	for _, u := range m.users {
		if u.Email == profile.Email {
			return u, nil
		}
	}
	u := auth.User{
		ID:        uuid.New(),
		Email:     profile.Email,
		Name:      profile.Name,
		AvatarURL: profile.AvatarURL,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockRepo) CreateSession(ctx context.Context, userID uuid.UUID, tokenHash string, userAgent, ipAddress *string, expiresAt time.Time) (auth.Session, error) {
	s := auth.Session{
		ID:           uuid.New(),
		UserID:       userID,
		TokenHash:    tokenHash,
		UserAgent:    userAgent,
		IPAddress:    ipAddress,
		ExpiresAt:    expiresAt,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	m.sessions[tokenHash] = s
	return s, nil
}

func (m *mockRepo) GetSessionAndUserByTokenHash(ctx context.Context, tokenHash string) (auth.Session, auth.User, error) {
	s, ok := m.sessions[tokenHash]
	if !ok {
		return auth.Session{}, auth.User{}, auth.ErrSessionNotFound
	}
	u, ok := m.users[s.UserID]
	if !ok {
		return auth.Session{}, auth.User{}, auth.ErrUserNotFound
	}
	return s, u, nil
}

func (m *mockRepo) TouchSession(ctx context.Context, sessionID uuid.UUID) error {
	for k, s := range m.sessions {
		if s.ID == sessionID {
			s.LastActiveAt = time.Now()
			m.sessions[k] = s
			return nil
		}
	}
	return nil
}

func (m *mockRepo) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) error {
	delete(m.sessions, tokenHash)
	return nil
}

func (m *mockRepo) DeleteSessionByID(ctx context.Context, sessionID, userID uuid.UUID) error {
	for k, s := range m.sessions {
		if s.ID == sessionID && s.UserID == userID {
			delete(m.sessions, k)
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (m *mockRepo) DeleteOtherSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) error {
	for k, s := range m.sessions {
		if s.UserID == userID && k != currentTokenHash {
			delete(m.sessions, k)
		}
	}
	return nil
}

func (m *mockRepo) DeleteAllSessions(ctx context.Context, userID uuid.UUID) error {
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
		}
	}
	return nil
}

func (m *mockRepo) ListActiveSessions(ctx context.Context, userID uuid.UUID, currentTokenHash string) ([]auth.Session, error) {
	var out []auth.Session
	for k, s := range m.sessions {
		if s.UserID == userID {
			s.IsCurrent = (k == currentTokenHash)
			out = append(out, s)
		}
	}
	return out, nil
}

func TestMultiDeviceSessions(t *testing.T) {
	repo := newMockRepo()
	reg := auth.NewProviderRegistry()
	reg.Register("google", &mockVerifier{
		profile: &auth.OAuthProfile{
			Provider:       "google",
			ProviderUserID: "google-uid-1",
			Email:          "alex@example.com",
			Name:           "Alex",
		},
	})

	svc := auth.NewService(repo, reg, 24*time.Hour)
	ctx := context.Background()

	// 1. Login on PC 1 (Chrome on Windows)
	token1, sess1, user1, err := svc.LoginWithProvider(ctx, "google", "token-pc1", auth.ClientInfo{
		UserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) Chrome/120.0.0.0",
		IPAddress: "192.168.1.10",
	})
	if err != nil {
		t.Fatalf("login PC1 failed: %v", err)
	}
	if token1 == "" || user1.Email != "alex@example.com" {
		t.Fatalf("unexpected user or token: %+v", user1)
	}

	// 2. Login on PC 2 (Firefox on macOS)
	token2, sess2, user2, err := svc.LoginWithProvider(ctx, "google", "token-pc2", auth.ClientInfo{
		UserAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15) Firefox/121.0",
		IPAddress: "192.168.1.20",
	})
	if err != nil {
		t.Fatalf("login PC2 failed: %v", err)
	}
	if user2.ID != user1.ID {
		t.Fatalf("expected same user ID across devices, got %v and %v", user1.ID, user2.ID)
	}
	if token1 == token2 {
		t.Fatalf("tokens must be unique per device session")
	}

	// 3. Both sessions should validate independently
	validatedUser1, validatedSess1, err := svc.ValidateSession(ctx, token1)
	if err != nil || validatedUser1.ID != user1.ID || validatedSess1.ID != sess1.ID {
		t.Fatalf("validate PC1 session failed: %v", err)
	}

	validatedUser2, validatedSess2, err := svc.ValidateSession(ctx, token2)
	if err != nil || validatedUser2.ID != user2.ID || validatedSess2.ID != sess2.ID {
		t.Fatalf("validate PC2 session failed: %v", err)
	}

	// 4. Listing sessions from PC 1 should show 2 sessions, with sess1 marked as current
	sessionsFromPC1, err := svc.ListSessions(ctx, user1.ID, token1)
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessionsFromPC1) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(sessionsFromPC1))
	}
	for _, s := range sessionsFromPC1 {
		if s.ID == sess1.ID && !s.IsCurrent {
			t.Errorf("sess1 should be marked as current from PC1")
		}
		if s.ID == sess2.ID && s.IsCurrent {
			t.Errorf("sess2 should not be marked as current from PC1")
		}
	}

	// 5. Revoking PC 2 session from PC 1
	err = svc.RevokeSession(ctx, sess2.ID, user1.ID)
	if err != nil {
		t.Fatalf("revoke session 2 failed: %v", err)
	}

	// PC 2 session is now invalid
	_, _, err = svc.ValidateSession(ctx, token2)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound for PC2 after revocation, got: %v", err)
	}

	// PC 1 session is still active
	_, _, err = svc.ValidateSession(ctx, token1)
	if err != nil {
		t.Fatalf("PC1 session should still be valid, got: %v", err)
	}

	// 6. Logout from PC 1
	err = svc.Logout(ctx, token1)
	if err != nil {
		t.Fatalf("logout PC1 failed: %v", err)
	}
	_, _, err = svc.ValidateSession(ctx, token1)
	if !errors.Is(err, auth.ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound for PC1 after logout, got: %v", err)
	}
}

func TestLoginWithPassword(t *testing.T) {
	repo := newMockRepo()
	reg := auth.NewProviderRegistry()
	svc := auth.NewService(repo, reg, 24*time.Hour)
	ctx := context.Background()

	// Setup a user with a hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("secretPassword123"), bcrypt.DefaultCost)
	userID := uuid.New()
	user := auth.User{
		ID:        userID,
		Email:     "alex@example.com",
		Name:      "Alex",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.users[userID] = user
	repo.passwords[userID] = string(hashedPassword)

	// Setup an OAuth-only user (no password set)
	oauthUserID := uuid.New()
	oauthUser := auth.User{
		ID:        oauthUserID,
		Email:     "oauth@example.com",
		Name:      "OAuth User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	repo.users[oauthUserID] = oauthUser

	// 1. Success case: correct email and password
	token, session, loggedInUser, err := svc.LoginWithPassword(ctx, "alex@example.com", "secretPassword123", auth.ClientInfo{
		UserAgent: "Mozilla/5.0",
		IPAddress: "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("expected login to succeed, got %v", err)
	}
	if token == "" || session.UserID != userID || loggedInUser.Email != "alex@example.com" {
		t.Fatalf("unexpected user or session details: %+v, %+v", loggedInUser, session)
	}

	// 2. Validate the issued session
	valUser, valSess, err := svc.ValidateSession(ctx, token)
	if err != nil || valUser.ID != userID || valSess.ID != session.ID {
		t.Fatalf("expected session validation to succeed, got %v", err)
	}

	// 3. Failure case: wrong password
	_, _, _, err = svc.LoginWithPassword(ctx, "alex@example.com", "wrongPassword", auth.ClientInfo{})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for wrong password, got %v", err)
	}

	// 4. Failure case: wrong email
	_, _, _, err = svc.LoginWithPassword(ctx, "nonexistent@example.com", "secretPassword123", auth.ClientInfo{})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for nonexistent email, got %v", err)
	}

	// 5. Failure case: user has no password set (OAuth-only account)
	_, _, _, err = svc.LoginWithPassword(ctx, "oauth@example.com", "somePassword", auth.ClientInfo{})
	if !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials for OAuth-only user, got %v", err)
	}
}
