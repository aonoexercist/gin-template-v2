package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"gin-template-v2/internal/features/auth"
	"gin-template-v2/internal/platform/dbgen"
)

type mockQuerier struct {
	dbgen.Querier
	users      map[uuid.UUID]dbgen.User
	identities map[string]uuid.UUID     // key: provider:provider_user_id -> user_id
	sessions   map[string]dbgen.Session // key: tokenHash -> session
}

func newMockQuerier() *mockQuerier {
	return &mockQuerier{
		users:      make(map[uuid.UUID]dbgen.User),
		identities: make(map[string]uuid.UUID),
		sessions:   make(map[string]dbgen.Session),
	}
}

func (m *mockQuerier) GetUserByIdentity(ctx context.Context, arg dbgen.GetUserByIdentityParams) (dbgen.User, error) {
	key := arg.Provider + ":" + arg.ProviderUserID
	userID, ok := m.identities[key]
	if !ok {
		return dbgen.User{}, pgx.ErrNoRows
	}
	u, ok := m.users[userID]
	if !ok {
		return dbgen.User{}, pgx.ErrNoRows
	}
	return u, nil
}

func (m *mockQuerier) UpdateUserAvatarAndName(ctx context.Context, arg dbgen.UpdateUserAvatarAndNameParams) (dbgen.User, error) {
	u, ok := m.users[arg.ID]
	if !ok {
		return dbgen.User{}, pgx.ErrNoRows
	}
	if arg.AvatarUrl != nil {
		u.AvatarUrl = arg.AvatarUrl
	}
	if arg.Name != nil {
		u.Name = *arg.Name
	}
	u.UpdatedAt = time.Now()
	m.users[arg.ID] = u
	return u, nil
}

func (m *mockQuerier) GetUserByEmail(ctx context.Context, email string) (dbgen.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return dbgen.User{}, pgx.ErrNoRows
}

func (m *mockQuerier) GetUserWithPasswordByEmail(ctx context.Context, email string) (dbgen.User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return dbgen.User{}, pgx.ErrNoRows
}

func (m *mockQuerier) CreateOAuthUser(ctx context.Context, arg dbgen.CreateOAuthUserParams) (dbgen.User, error) {
	u := dbgen.User{
		ID:        uuid.New(),
		Email:     arg.Email,
		Name:      arg.Name,
		AvatarUrl: arg.AvatarUrl,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.users[u.ID] = u
	return u, nil
}

func (m *mockQuerier) CreateIdentity(ctx context.Context, arg dbgen.CreateIdentityParams) (dbgen.UserIdentity, error) {
	key := arg.Provider + ":" + arg.ProviderUserID
	m.identities[key] = arg.UserID
	return dbgen.UserIdentity{
		ID:             uuid.New(),
		UserID:         arg.UserID,
		Provider:       arg.Provider,
		ProviderUserID: arg.ProviderUserID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}, nil
}

func (m *mockQuerier) CreateSession(ctx context.Context, arg dbgen.CreateSessionParams) (dbgen.Session, error) {
	s := dbgen.Session{
		ID:           uuid.New(),
		UserID:       arg.UserID,
		TokenHash:    arg.TokenHash,
		UserAgent:    arg.UserAgent,
		IpAddress:    arg.IpAddress,
		ExpiresAt:    arg.ExpiresAt,
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
	m.sessions[arg.TokenHash] = s
	return s, nil
}

func (m *mockQuerier) GetSessionAndUserByTokenHash(ctx context.Context, tokenHash string) (dbgen.GetSessionAndUserByTokenHashRow, error) {
	s, ok := m.sessions[tokenHash]
	if !ok {
		return dbgen.GetSessionAndUserByTokenHashRow{}, pgx.ErrNoRows
	}
	u, ok := m.users[s.UserID]
	if !ok {
		return dbgen.GetSessionAndUserByTokenHashRow{}, pgx.ErrNoRows
	}
	return dbgen.GetSessionAndUserByTokenHashRow{
		SessionID:        s.ID,
		UserID:           s.UserID,
		TokenHash:        s.TokenHash,
		UserAgent:        s.UserAgent,
		IpAddress:        s.IpAddress,
		ExpiresAt:        s.ExpiresAt,
		SessionCreatedAt: s.CreatedAt,
		LastActiveAt:     s.LastActiveAt,
		UserPk:           u.ID,
		Email:            u.Email,
		Name:             u.Name,
		AvatarUrl:        u.AvatarUrl,
		IsAdmin:          u.IsAdmin,
		UserCreatedAt:    u.CreatedAt,
		UserUpdatedAt:    u.UpdatedAt,
	}, nil
}

func (m *mockQuerier) TouchSession(ctx context.Context, id uuid.UUID) error {
	for k, s := range m.sessions {
		if s.ID == id {
			s.LastActiveAt = time.Now()
			m.sessions[k] = s
			return nil
		}
	}
	return nil
}

func (m *mockQuerier) DeleteSessionByTokenHash(ctx context.Context, tokenHash string) (int64, error) {
	if _, ok := m.sessions[tokenHash]; ok {
		delete(m.sessions, tokenHash)
		return 1, nil
	}
	return 0, nil
}

func (m *mockQuerier) DeleteSessionByID(ctx context.Context, arg dbgen.DeleteSessionByIDParams) (int64, error) {
	for k, s := range m.sessions {
		if s.ID == arg.ID && s.UserID == arg.UserID {
			delete(m.sessions, k)
			return 1, nil
		}
	}
	return 0, nil
}

func (m *mockQuerier) DeleteOtherSessionsForUser(ctx context.Context, arg dbgen.DeleteOtherSessionsForUserParams) (int64, error) {
	var count int64
	for k, s := range m.sessions {
		if s.UserID == arg.UserID && k != arg.TokenHash {
			delete(m.sessions, k)
			count++
		}
	}
	return count, nil
}

func (m *mockQuerier) DeleteAllSessionsForUser(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	for k, s := range m.sessions {
		if s.UserID == userID {
			delete(m.sessions, k)
			count++
		}
	}
	return count, nil
}

func (m *mockQuerier) ListActiveSessionsForUser(ctx context.Context, userID uuid.UUID) ([]dbgen.Session, error) {
	var out []dbgen.Session
	now := time.Now()
	for _, s := range m.sessions {
		if s.UserID == userID && s.ExpiresAt.After(now) {
			out = append(out, s)
		}
	}
	return out, nil
}

func TestMultiDeviceSessions(t *testing.T) {
	q := newMockQuerier()
	reg := auth.NewProviderRegistry()
	reg.Register("google", &mockVerifier{
		profile: &auth.OAuthProfile{
			Provider:       "google",
			ProviderUserID: "google-uid-1",
			Email:          "alex@example.com",
			Name:           "Alex",
		},
	})

	svc := auth.NewService(q, nil, reg, 24*time.Hour)
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

	// 4. Listing sessions from PC 1 should show 2 sessions
	sessionsFromPC1, err := svc.ListSessions(ctx, user1.ID, token1)
	if err != nil {
		t.Fatalf("list sessions failed: %v", err)
	}
	if len(sessionsFromPC1) != 2 {
		t.Fatalf("expected 2 active sessions, got %d", len(sessionsFromPC1))
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
	q := newMockQuerier()
	reg := auth.NewProviderRegistry()
	svc := auth.NewService(q, nil, reg, 24*time.Hour)
	ctx := context.Background()

	// Setup a user with a hashed password
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("secretPassword123"), bcrypt.DefaultCost)
	hashedPasswordStr := string(hashedPassword)
	userID := uuid.New()
	user := dbgen.User{
		ID:           userID,
		Email:        "alex@example.com",
		Name:         "Alex",
		PasswordHash: &hashedPasswordStr,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	q.users[userID] = user

	// Setup an OAuth-only user (no password set)
	oauthUserID := uuid.New()
	oauthUser := dbgen.User{
		ID:        oauthUserID,
		Email:     "oauth@example.com",
		Name:      "OAuth User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	q.users[oauthUserID] = oauthUser

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
