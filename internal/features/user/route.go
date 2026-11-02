package user

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware, adminMiddleware gin.HandlerFunc) {
	g := rg.Group("/users")

	// 1. Public: anyone can register
	g.POST("", h.Create)

	// 2. Protected: any authenticated user
	protected := g.Group("", authMiddleware)
	protected.GET("/:id", h.Get)

	// 3. Admin-only: only admins can list, update, or delete users
	adminOnly := protected.Group("", adminMiddleware)
	adminOnly.GET("", h.List)
	adminOnly.PATCH("/:id", h.Update)
	adminOnly.DELETE("/:id", h.Delete)
}
