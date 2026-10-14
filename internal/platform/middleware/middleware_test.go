package middleware_test

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/middleware"
)

func init() { gin.SetMode(gin.TestMode) }

func newEngine(buf *bytes.Buffer, allowedOrigins ...string) *gin.Engine {
	log := slog.New(slog.NewJSONHandler(buf, nil))
	r := gin.New()
	r.Use(
		middleware.RequestID(),
		middleware.RequestLogger(log, "/healthz"),
		middleware.Recovery(),
		middleware.SecureHeaders(),
		middleware.CORS(allowedOrigins),
	)
	r.GET("/ok", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/panic", func(*gin.Context) { panic("boom") })
	return r
}

func do(r http.Handler, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequestID_GeneratedAndPropagated(t *testing.T) {
	r := newEngine(&bytes.Buffer{})

	if id := do(r, "GET", "/ok", nil).Header().Get("X-Request-ID"); id == "" {
		t.Error("expected generated request id")
	}
	w := do(r, "GET", "/ok", map[string]string{"X-Request-ID": "abc-123"})
	if got := w.Header().Get("X-Request-ID"); got != "abc-123" {
		t.Errorf("request id = %q, want abc-123", got)
	}
	w = do(r, "GET", "/ok", map[string]string{"X-Request-ID": "bad id\nwith newline"})
	if got := w.Header().Get("X-Request-ID"); got == "" || strings.ContainsAny(got, " \n") {
		t.Errorf("unsafe request id accepted: %q", got)
	}
}

func TestRequestLogger_LogsAndSkips(t *testing.T) {
	var buf bytes.Buffer
	r := newEngine(&buf)

	do(r, "GET", "/healthz", nil)
	if buf.Len() != 0 {
		t.Errorf("skipped path was logged: %s", buf.String())
	}
	do(r, "GET", "/ok", nil)
	if !strings.Contains(buf.String(), `"path":"/ok"`) || !strings.Contains(buf.String(), `"request_id"`) {
		t.Errorf("access log missing fields: %s", buf.String())
	}
}

func TestRecovery_Returns500JSON(t *testing.T) {
	var buf bytes.Buffer
	r := newEngine(&buf)

	w := do(r, "GET", "/panic", nil)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"internal_error"`) {
		t.Errorf("body = %s", w.Body.String())
	}
	if !strings.Contains(buf.String(), "panic recovered") {
		t.Errorf("panic not logged: %s", buf.String())
	}
}

func TestCORS(t *testing.T) {
	r := newEngine(&bytes.Buffer{}, "https://app.example.com")

	w := do(r, "GET", "/ok", map[string]string{"Origin": "https://app.example.com"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Errorf("allowed origin header = %q", got)
	}
	w = do(r, "GET", "/ok", map[string]string{"Origin": "https://evil.example.com"})
	if got := w.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("disallowed origin got header %q", got)
	}
	w = do(r, "OPTIONS", "/ok", map[string]string{
		"Origin":                        "https://app.example.com",
		"Access-Control-Request-Method": "POST",
	})
	if w.Code != http.StatusNoContent {
		t.Errorf("preflight status = %d", w.Code)
	}
}
