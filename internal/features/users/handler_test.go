package users

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/httpx"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	httpx.RegisterJSONTagNames()
	r := gin.New()
	RegisterRoutes(r.Group("/api/v1"), NewHandler(NewService(newFakeRepo())))
	return r
}

func call(r http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_CRUDFlow(t *testing.T) {
	r := newTestRouter()

	w := call(r, "POST", "/api/v1/users", `{"email":"jane@example.com","name":"Jane"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("create status = %d body=%s", w.Code, w.Body)
	}
	var created struct {
		Data UserResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if w.Header().Get("Location") != "/api/v1/users/"+created.Data.ID.String() {
		t.Errorf("Location = %q", w.Header().Get("Location"))
	}

	path := "/api/v1/users/" + created.Data.ID.String()
	if w := call(r, "GET", path, ""); w.Code != http.StatusOK {
		t.Errorf("get status = %d", w.Code)
	}
	if w := call(r, "PATCH", path, `{"name":"Janet"}`); w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Janet") {
		t.Errorf("patch status = %d body=%s", w.Code, w.Body)
	}

	w = call(r, "GET", "/api/v1/users?page=1&per_page=10", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":1`) {
		t.Errorf("list status = %d body=%s", w.Code, w.Body)
	}

	if w := call(r, "DELETE", path, ""); w.Code != http.StatusNoContent {
		t.Errorf("delete status = %d", w.Code)
	}
	if w := call(r, "GET", path, ""); w.Code != http.StatusNotFound {
		t.Errorf("get after delete status = %d", w.Code)
	}
}

func TestHandler_Errors(t *testing.T) {
	r := newTestRouter()
	call(r, "POST", "/api/v1/users", `{"email":"dup@example.com","name":"Dup"}`)

	tests := []struct {
		name, method, path, body string
		want                     int
	}{
		{"invalid email", "POST", "/api/v1/users", `{"email":"nope","name":"X"}`, http.StatusUnprocessableEntity},
		{"missing name", "POST", "/api/v1/users", `{"email":"a@b.co"}`, http.StatusUnprocessableEntity},
		{"malformed json", "POST", "/api/v1/users", `{`, http.StatusBadRequest},
		{"duplicate email", "POST", "/api/v1/users", `{"email":"dup@example.com","name":"Dup"}`, http.StatusConflict},
		{"bad uuid", "GET", "/api/v1/users/not-a-uuid", "", http.StatusBadRequest},
		{"unknown id", "GET", "/api/v1/users/6f1c1c9e-5a43-4a7c-9d0b-0a7f3f1d2b11", "", http.StatusNotFound},
		{"empty patch", "PATCH", "/api/v1/users/6f1c1c9e-5a43-4a7c-9d0b-0a7f3f1d2b11", `{}`, http.StatusUnprocessableEntity},
		{"bad per_page", "GET", "/api/v1/users?per_page=1000", "", http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if w := call(r, tt.method, tt.path, tt.body); w.Code != tt.want {
				t.Errorf("status = %d, want %d (body=%s)", w.Code, tt.want, w.Body)
			}
		})
	}
}

func TestHandler_ValidationErrorUsesJSONFieldNames(t *testing.T) {
	r := newTestRouter()
	w := call(r, "POST", "/api/v1/users", `{"email":"nope","name":"X"}`)
	if !strings.Contains(w.Body.String(), `"email"`) || strings.Contains(w.Body.String(), `"Email"`) {
		t.Errorf("expected json field names in details: %s", w.Body)
	}
}
