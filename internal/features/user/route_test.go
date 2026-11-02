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
	if err != nil || cookie == "" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	c.Next()
}

func dummyAdminMiddleware(c *gin.Context) {
	cookie, _ := c.Cookie("sid")
	if cookie != "admin-cookie" {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		return
	}
	c.Next()
}

func TestOptionB_UserRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	handler := user.NewHandler(&mockUserService{})
	v1 := r.Group("/api/v1")
	user.RegisterRoutes(v1, handler, dummyAuthMiddleware, dummyAdminMiddleware)

	userID := uuid.New().String()

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
		t.Fatalf("expected 201 Created for public registration, got %d", w.Code)
	}

	// 2. GET /api/v1/users/:id requires authentication (fails 401 without cookie)
	req = httptest.NewRequest("GET", "/api/v1/users/"+userID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for GET /users/:id without cookie, got %d", w.Code)
	}

	// 3. GET /api/v1/users/:id succeeds for regular authenticated user
	req = httptest.NewRequest("GET", "/api/v1/users/"+userID, nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "regular-user-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for GET /users/:id with regular user, got %d", w.Code)
	}

	// 4. GET /api/v1/users (list) is ADMIN-ONLY (fails 403 for regular user)
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "regular-user-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for regular user on GET /users, got %d", w.Code)
	}

	// 5. GET /api/v1/users (list) succeeds for admin
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "admin-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for admin on GET /users, got %d", w.Code)
	}

	// 6. DELETE /api/v1/users/:id is ADMIN-ONLY (fails 403 for regular user)
	req = httptest.NewRequest("DELETE", "/api/v1/users/"+userID, nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "regular-user-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for regular user on DELETE /users/:id, got %d", w.Code)
	}

	// 7. DELETE /api/v1/users/:id succeeds for admin (204 No Content)
	req = httptest.NewRequest("DELETE", "/api/v1/users/"+userID, nil)
	req.AddCookie(&http.Cookie{Name: "sid", Value: "admin-cookie"})
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204 No Content for admin on DELETE /users/:id, got %d", w.Code)
	}
}
