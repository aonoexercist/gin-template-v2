package auth

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the auth endpoints on the versioned router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	g := rg.Group("/auth")

	// Public login endpoints
	g.POST("/google", h.LoginGoogle)
	g.POST("/login", h.Login)
	g.POST("/:provider", h.LoginProvider)

	// Protected endpoints (cookie session required)
	protected := g.Group("")
	protected.Use(authMiddleware)
	{
		protected.GET("/me", h.Me)
		protected.POST("/logout", h.Logout)
		protected.POST("/logout-all", h.LogoutAll)
		protected.GET("/sessions", h.ListSessions)
		protected.DELETE("/sessions/:id", h.RevokeSession)
	}
}
