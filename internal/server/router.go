package server

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"gin-template-v2/internal/config"
	"gin-template-v2/internal/features/auth"
	"gin-template-v2/internal/features/health"
	users "gin-template-v2/internal/features/user"
	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/middleware"
)

// NewRouter wires middleware, features and API versions.
//
// Layout:
//
//	/healthz, /readyz   unversioned operational probes
//	/api/v1/...         current stable API
//
// To add a feature: build its repo/service/handler below and mount it on a version group.
// To add a breaking API version: create `v2 := r.Group("/api/v2")` and mount only the
// features that changed; v1 keeps serving existing clients untouched.
func NewRouter(cfg config.Config, log *slog.Logger, pool *pgxpool.Pool) *gin.Engine {
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}
	httpx.RegisterJSONTagNames()

	r := gin.New()
	r.HandleMethodNotAllowed = true
	if err := r.SetTrustedProxies(cfg.HTTP.TrustedProxies); err != nil {
		log.Error("invalid TRUSTED_PROXIES, trusting none", "error", err)
		_ = r.SetTrustedProxies(nil)
	}

	// Order matters: request id -> logger (needs id) -> recovery (needs logger).
	r.Use(
		middleware.RequestID(),
		middleware.RequestLogger(log, "/healthz", "/readyz"),
		middleware.Recovery(),
		middleware.SecureHeaders(),
		middleware.CORS(cfg.CORS.AllowedOrigins),
	)

	r.NoRoute(func(c *gin.Context) {
		httpx.Error(c, http.StatusNotFound, "not_found", "Route not found")
	})
	r.NoMethod(func(c *gin.Context) {
		httpx.Error(c, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
	})

	// Operational endpoints.
	health.RegisterRoutes(r, health.NewHandler(pool))

	// --- Auth & Identity Providers ---
	providers := auth.NewProviderRegistry()
	providers.Register("google", auth.NewGoogleVerifier(cfg.Auth.GoogleClientID))

	authRepo := auth.NewRepository(pool)
	authSvc := auth.NewService(authRepo, providers, cfg.Auth.SessionDuration)
	authHandler := auth.NewHandler(authSvc, cfg.Auth)
	authMiddleware := middleware.Authenticate(authSvc, cfg.Auth.CookieName)

	// --- Dependency wiring (one block per feature) ---
	usersHandler := users.NewHandler(users.NewService(users.NewRepository(pool)))

	// --- API v1 ---
	v1 := r.Group("/api/v1")
	auth.RegisterRoutes(v1, authHandler, authMiddleware)
	users.RegisterRoutes(v1, usersHandler, authMiddleware)

	return r
}
