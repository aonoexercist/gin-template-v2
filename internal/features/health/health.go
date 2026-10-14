// Package health exposes liveness and readiness probes (unversioned, for
// orchestrators and load balancers).
package health

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"
	"gin-template-v2/internal/version"
)

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Handler struct{ db Pinger }

func NewHandler(db Pinger) *Handler { return &Handler{db: db} }

// RegisterRoutes mounts /healthz and /readyz on the engine root.
func RegisterRoutes(r gin.IRoutes, h *Handler) {
	r.GET("/healthz", h.Live)
	r.GET("/readyz", h.Ready)
}

// Live reports that the process is up. It never touches dependencies.
func (h *Handler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "ok",
		"version": version.Version,
		"commit":  version.Commit,
	})
}

// Ready reports whether dependencies (the database) are reachable.
func (h *Handler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		logger.FromContext(c.Request.Context()).Error("readiness check failed", "error", err)
		httpx.Error(c, http.StatusServiceUnavailable, "unavailable", "Database not ready")
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ready"})
}
