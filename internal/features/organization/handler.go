package organization

import (
	"errors"
	"net/http"

	"gin-template-v2/internal/platform/httpx"
	"gin-template-v2/internal/platform/logger"

	"github.com/gin-gonic/gin"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) CreateOrganization(c *gin.Context) {
	var req CreateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}

	org, err := h.svc.CreateOrganization(c.Request.Context(), req)
	if err != nil {
		h.fail(c, err)
		return
	}

	c.Header("Location", c.Request.URL.Path+"/"+org.ID.String())
	httpx.JSON(c, http.StatusCreated, newOrganizationResponse(org))
}

// TODO: implement; route.go already mounts these.
func (h *Handler) GetOrganization(c *gin.Context)    {}
func (h *Handler) ListOrganizations(c *gin.Context)  {}
func (h *Handler) UpdateOrganization(c *gin.Context) {}
func (h *Handler) DeleteOrganization(c *gin.Context) {}

// fail maps domain errors to HTTP responses; unknown errors are logged and
// returned as an opaque 500 so internals never leak to clients.
func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "Organization not found")
	case errors.Is(err, ErrOrganizationTaken):
		httpx.Error(c, http.StatusConflict, "organization_taken", "Organization already in use")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_error", "Invalid input")
	default:
		logger.FromContext(c.Request.Context()).Error("organizations: unexpected error", "error", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
