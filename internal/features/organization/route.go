package organization

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	g := rg.Group("/organizations")

	// Protected: authenticated user
	protected := g.Group("", authMiddleware)

	protected.POST("", h.CreateOrganization)
	protected.GET("/:id", h.GetOrganization)
	protected.GET("", h.ListOrganizations)
	protected.PATCH("/:id", h.UpdateOrganization)
	protected.DELETE("/:id", h.DeleteOrganization)
}
