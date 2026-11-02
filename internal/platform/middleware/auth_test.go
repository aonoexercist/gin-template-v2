package middleware_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-template-v2/internal/features/auth"
	"gin-template-v2/internal/platform/middleware"
)

type mockAuthService struct {
	validToken string
	user       auth.User
	session    auth.Session
}

func (m *mockAuthService) LoginWithProvider(ctx context.Context, provider string, token string, clientInfo auth.ClientInfo) (string, auth.Session, auth.User, error) {
	return "", auth.Session{}, auth.User{}, nil
}

func (m *mockAuthService) LoginWithPassword(ctx context.Context, email, password string, clientInfo auth.ClientInfo) (string, auth.Session, auth.User, error) {
	return "", auth.Session{}, auth.User{}, nil
}

func (m *mockAuthService) ValidateSession(ctx context.Context, rawToken string) (auth.User, auth.Session, error) {
	if rawToken != m.validToken {
		return auth.User{}, auth.Session{}, errors.New("invalid session")
	}
	return m.user, m.session, nil
}

func (m *mockAuthService) Logout(ctx context.Context, rawToken string) error {
	return nil
}

func (m *mockAuthService) RevokeSession(ctx context.Context, sessionID, userID uuid.UUID) error {
	return nil
}

func (m *mockAuthService) RevokeAllSessions(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func (m *mockAuthService) RevokeOtherSessions(ctx context.Context, rawToken string, userID uuid.UUID) error {
	return nil
}

func (m *mockAuthService) ListSessions(ctx context.Context, userID uuid.UUID, rawToken string) ([]auth.Session, error) {
	return nil, nil
}

func TestAuthMiddleware(t *testing.T) {
	mockUser := auth.User{
		ID:        uuid.New(),
		Email:     "user@example.com",
		Name:      "Test User",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	mockSess := auth.Session{
		ID:        uuid.New(),
		UserID:    mockUser.ID,
		ExpiresAt: time.Now().Add(time.Hour),
	}

	svc := &mockAuthService{
		validToken: "valid-session-secret",
		user:       mockUser,
		session:    mockSess,
	}

	r := gin.New()
	r.Use(middleware.Authenticate(svc, "sid"))
	r.GET("/protected", func(c *gin.Context) {
		u, ok := middleware.CurrentUser(c)
		if !ok || u.Email != "user@example.com" {
			c.Status(http.StatusInternalServerError)
			return
		}
		c.String(http.StatusOK, "ok")
	})

	// Case 1: Missing cookie -> 401
	req := httptest.NewRequest("GET", "/protected", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for missing cookie, got %d", w.Code)
	}

	// Case 2: Invalid cookie -> 401
	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "invalid-token"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for invalid cookie, got %d", w.Code)
	}

	// Case 3: Valid cookie -> 200
	req = httptest.NewRequest("GET", "/protected", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "valid-session-secret"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for valid cookie, got %d", w.Code)
	}
}
