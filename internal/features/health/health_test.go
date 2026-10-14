package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

type fakePinger struct{ err error }

func (f fakePinger) Ping(context.Context) error { return f.err }

func get(t *testing.T, p Pinger, path string) int {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	RegisterRoutes(r, NewHandler(p))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

func TestLive(t *testing.T) {
	if got := get(t, fakePinger{errors.New("down")}, "/healthz"); got != http.StatusOK {
		t.Errorf("/healthz = %d, want 200 even when db is down", got)
	}
}

func TestReady(t *testing.T) {
	if got := get(t, fakePinger{}, "/readyz"); got != http.StatusOK {
		t.Errorf("/readyz healthy = %d", got)
	}
	if got := get(t, fakePinger{errors.New("down")}, "/readyz"); got != http.StatusServiceUnavailable {
		t.Errorf("/readyz unhealthy = %d", got)
	}
}
