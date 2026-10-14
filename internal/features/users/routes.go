package users

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/users")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}
