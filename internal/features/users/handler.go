package users

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"gin-template-v2/internal/platform/httpx"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Create(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	u, err := h.svc.Create(c.Request.Context(), req.Email, req.Name, req.Password)
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
