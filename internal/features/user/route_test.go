package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-template-v2/internal/features/user"
)

type mockUserService struct{}

func (m *mockUserService) Create(ctx context.Context, email, name string, password string) (user.User, error) {
	return user.User{
		ID:        uuid.New(),
		Email:     email,
		Name:      name,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockUserService) Get(ctx context.Context, id uuid.UUID) (user.User, error) {
	return user.User{
		ID:        id,
		Email:     "test@example.com",
		Name:      "Test",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}, nil
}

func (m *mockUserService) List(ctx context.Context, page, perPage int) ([]user.User, int64, error) {
	return []user.User{}, 0, nil
}

func (m *mockUserService) Update(ctx context.Context, id uuid.UUID, email, name *string) (user.User, error) {
	return user.User{ID: id}, nil
}

func (m *mockUserService) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func dummyAuthMiddleware(c *gin.Context) {
	cookie, err := c.Cookie("sid")
	if err != nil || cookie != "valid-cookie" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}

func TestOptionB_UserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handler := user.NewHandler(&mockUserService{})
	v1 := r.Group("/api/v1")
	user.RegisterRoutes(v1, handler, dummyAuthMiddleware)

	// 1. POST /api/v1/users is PUBLIC (registration succeeds without cookie)
	body, _ := json.Marshal(map[string]string{
		"email":    "newuser@example.com",
		"name":     "New User",
		"password": "Password123!",
	})
	req := httptest.NewRequest("POST", "/api/v1/users", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for public registration, got %d (body: %s)", w.Code, w.Body.String())
	}

	// 2. GET /api/v1/users is PROTECTED (fails 401 without cookie)
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 Unauthorized for GET /users without cookie, got %d", w.Code)
	}

	// 3. GET /api/v1/users succeeds with valid cookie
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "valid-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for GET /users with cookie, got %d", w.Code)
	}
}
