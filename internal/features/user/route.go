package user

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	g := rg.Group("/users")

	// public
	g.POST("", h.Create)

	// protected
	protected := g.Group("", authMiddleware)
	protected.GET("", h.List)
	protected.GET("/:id", h.Get)
	protected.PATCH("/:id", h.Update)
	protected.DELETE("/:id", h.Delete)
}
