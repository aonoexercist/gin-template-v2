package users

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler) {
	g := rg.Group("/users")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
}

func (h *Handler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	u, err := h.svc.Create(c.Request.Context(), req.Email, req.Name)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Location", c.Request.URL.Path+"/"+u.ID.String())
	httpx.JSON(c, http.StatusCreated, newUserResponse(u))
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	u, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, newUserResponse(u))
}

func (h *Handler) List(c *gin.Context) {
	var q ListUsersQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpx.BindError(c, err)
		return
	}
	if q.Page == 0 {
		q.Page = 1
	}
	if q.PerPage == 0 {
		q.PerPage = 20
	}

	items, total, err := h.svc.List(c.Request.Context(), q.Page, q.PerPage)
	if err != nil {
		h.fail(c, err)
		return
	}
	out := make([]UserResponse, len(items))
	for i, u := range items {
		out[i] = newUserResponse(u)
	}
	httpx.Paginated(c, out, httpx.Meta{Page: q.Page, PerPage: q.PerPage, Total: total})
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	u, err := h.svc.Update(c.Request.Context(), id, req.Email, req.Name)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, newUserResponse(u))
}

func (h *Handler) Delete(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		h.fail(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "ID must be a valid UUID")
		return uuid.Nil, false
	}
	return id, true
}

// fail maps domain errors to HTTP responses; unknown errors are logged and
// returned as an opaque 500 so internals never leak to clients.
func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "User not found")
	case errors.Is(err, ErrEmailTaken):
		httpx.Error(c, http.StatusConflict, "email_taken", "Email already in use")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_error", "Invalid input")
	default:
		logger.FromContext(c.Request.Context()).Error("users: unexpected error", "error", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
