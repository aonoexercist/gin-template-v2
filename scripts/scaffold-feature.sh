#!/usr/bin/env bash
# Scaffold a new feature directory with migration, queries, service, handler, and routes.
# Usage:
#   ./scripts/scaffold-feature.sh posts
#   make feature name=posts
set -euo pipefail

NAME="${1:-}"
if [ -z "$NAME" ]; then
    echo "Error: feature name is required"
    echo "Usage: $0 <feature_name>"
    echo "Example: make feature name=posts"
    exit 1
fi

# Lowercase and clean name
FEATURE=$(echo "$NAME" | tr '[:upper:]' '[:lower:]' | tr -cd '[:alnum:]_')
FEATURE_DIR="internal/features/${FEATURE}"
MIGRATIONS_DIR="db/migrations"

# Detect module name from go.mod
MODULE=$(go list -m 2>/dev/null || echo "gin-template-v2")

cd "$(dirname "$0")/.."

if [ -d "$FEATURE_DIR" ]; then
    echo "Error: Directory ${FEATURE_DIR} already exists."
    exit 1
fi

# Capitalize for Go names (e.g. posts -> Post / Posts)
# Simple singularization if ends with s
SINGULAR="$FEATURE"
if [[ "$FEATURE" == *s ]]; then
    SINGULAR="${FEATURE%s}"
fi
TYPE_NAME="$(echo "${SINGULAR:0:1}" | tr '[:lower:]' '[:upper:]')${SINGULAR:1}"

TIMESTAMP=$(date -u +%Y%m%d%H%M%S)
MIGRATION_FILE="${MIGRATIONS_DIR}/${TIMESTAMP}_create_${FEATURE}_table.sql"

echo "Creating feature: ${FEATURE} (${TYPE_NAME})"

# 1. Create migration
mkdir -p "$MIGRATIONS_DIR"
cat <<EOF > "$MIGRATION_FILE"
-- +goose Up
CREATE TABLE ${FEATURE} (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title      TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS ${FEATURE};
EOF
echo "  ✓ Created migration: ${MIGRATION_FILE}"

# 2. Create feature directory and files
mkdir -p "$FEATURE_DIR"

# query.sql
cat <<EOF > "${FEATURE_DIR}/query.sql"
-- name: Create${TYPE_NAME} :one
INSERT INTO ${FEATURE} (title)
VALUES (\$1)
RETURNING id, title, created_at, updated_at;

-- name: Get${TYPE_NAME} :one
SELECT id, title, created_at, updated_at
FROM ${FEATURE}
WHERE id = \$1;

-- name: List${TYPE_NAME}s :many
SELECT id, title, created_at, updated_at
FROM ${FEATURE}
ORDER BY created_at DESC, id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: Count${TYPE_NAME}s :one
SELECT count(*) FROM ${FEATURE};

-- name: Update${TYPE_NAME} :one
UPDATE ${FEATURE}
SET title      = COALESCE(sqlc.narg(title), title),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING id, title, created_at, updated_at;

-- name: Delete${TYPE_NAME} :execrows
DELETE FROM ${FEATURE} WHERE id = \$1;
EOF
echo "  ✓ Created ${FEATURE_DIR}/query.sql"

# <feature>.go
cat <<EOF > "${FEATURE_DIR}/${FEATURE}.go"
package ${FEATURE}

import (
	"errors"
	"time"

	"${MODULE}/internal/platform/dbgen"

	"github.com/google/uuid"
)

// ${TYPE_NAME} aliases the sqlc-generated row model.
type ${TYPE_NAME} = dbgen.${TYPE_NAME}

var (
	ErrNotFound     = errors.New("${FEATURE}: not found")
	ErrInvalidInput = errors.New("${FEATURE}: invalid input")
)

// ---- HTTP DTOs ----

type Create${TYPE_NAME}Request struct {
	Title string \`json:"title" binding:"required,min=1,max=255"\`
}

type Update${TYPE_NAME}Request struct {
	Title *string \`json:"title" binding:"omitempty,min=1,max=255"\`
}

type List${TYPE_NAME}sQuery struct {
	Page    int \`form:"page" binding:"omitempty,min=1,max=1000000"\`
	PerPage int \`form:"per_page" binding:"omitempty,min=1,max=100"\`
}

type ${TYPE_NAME}Response struct {
	ID        uuid.UUID \`json:"id"\`
	Title     string    \`json:"title"\`
	CreatedAt time.Time \`json:"created_at"\`
	UpdatedAt time.Time \`json:"updated_at"\`
}

func new${TYPE_NAME}Response(item ${TYPE_NAME}) ${TYPE_NAME}Response {
	return ${TYPE_NAME}Response{
		ID:        item.ID,
		Title:     item.Title,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}
EOF
echo "  ✓ Created ${FEATURE_DIR}/${FEATURE}.go"

# service.go
cat <<EOF > "${FEATURE_DIR}/service.go"
package ${FEATURE}

import (
	"context"
	"errors"
	"strings"

	"${MODULE}/internal/platform/dbgen"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service interface {
	Create(ctx context.Context, title string) (${TYPE_NAME}, error)
	Get(ctx context.Context, id uuid.UUID) (${TYPE_NAME}, error)
	List(ctx context.Context, page, perPage int) ([]${TYPE_NAME}, int64, error)
	Update(ctx context.Context, id uuid.UUID, title *string) (${TYPE_NAME}, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type service struct{ q dbgen.Querier }

func NewService(q dbgen.Querier) Service { return &service{q: q} }

func (s *service) Create(ctx context.Context, title string) (${TYPE_NAME}, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return ${TYPE_NAME}{}, ErrInvalidInput
	}
	return s.q.Create${TYPE_NAME}(ctx, title)
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (${TYPE_NAME}, error) {
	item, err := s.q.Get${TYPE_NAME}(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ${TYPE_NAME}{}, ErrNotFound
		}
		return ${TYPE_NAME}{}, err
	}
	return item, nil
}

func (s *service) List(ctx context.Context, page, perPage int) ([]${TYPE_NAME}, int64, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	items, err := s.q.List${TYPE_NAME}s(ctx, dbgen.List${TYPE_NAME}sParams{
		PageLimit:  int32(perPage),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := s.q.Count${TYPE_NAME}s(ctx)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *service) Update(ctx context.Context, id uuid.UUID, title *string) (${TYPE_NAME}, error) {
	params := dbgen.Update${TYPE_NAME}Params{ID: id}
	if title != nil {
		t := strings.TrimSpace(*title)
		if t == "" {
			return ${TYPE_NAME}{}, ErrInvalidInput
		}
		params.Title = pgtype.Text{String: t, Valid: true}
	}

	item, err := s.q.Update${TYPE_NAME}(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ${TYPE_NAME}{}, ErrNotFound
		}
		return ${TYPE_NAME}{}, err
	}
	return item, nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	rows, err := s.q.Delete${TYPE_NAME}(ctx, id)
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}
EOF
echo "  ✓ Created ${FEATURE_DIR}/service.go"

# handler.go
cat <<EOF > "${FEATURE_DIR}/handler.go"
package ${FEATURE}

import (
	"errors"
	"net/http"

	"${MODULE}/internal/platform/httpx"
	"${MODULE}/internal/platform/logger"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct{ svc Service }

func NewHandler(svc Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Create(c *gin.Context) {
	var req Create${TYPE_NAME}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	item, err := h.svc.Create(c.Request.Context(), req.Title)
	if err != nil {
		h.fail(c, err)
		return
	}
	c.Header("Location", c.Request.URL.Path+"/"+item.ID.String())
	httpx.JSON(c, http.StatusCreated, new${TYPE_NAME}Response(item))
}

func (h *Handler) Get(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	item, err := h.svc.Get(c.Request.Context(), id)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, new${TYPE_NAME}Response(item))
}

func (h *Handler) List(c *gin.Context) {
	var q List${TYPE_NAME}sQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		httpx.BindError(c, err)
		return
	}
	items, total, err := h.svc.List(c.Request.Context(), q.Page, q.PerPage)
	if err != nil {
		h.fail(c, err)
		return
	}
	resp := make([]${TYPE_NAME}Response, len(items))
	for i, item := range items {
		resp[i] = new${TYPE_NAME}Response(item)
	}
	httpx.JSON(c, http.StatusOK, gin.H{
		"items": resp,
		"total": total,
	})
}

func (h *Handler) Update(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req Update${TYPE_NAME}Request
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.BindError(c, err)
		return
	}
	item, err := h.svc.Update(c.Request.Context(), id, req.Title)
	if err != nil {
		h.fail(c, err)
		return
	}
	httpx.JSON(c, http.StatusOK, new${TYPE_NAME}Response(item))
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
		httpx.Error(c, http.StatusBadRequest, "invalid_id", "Malformed UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) fail(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpx.Error(c, http.StatusNotFound, "not_found", "${TYPE_NAME} not found")
	case errors.Is(err, ErrInvalidInput):
		httpx.Error(c, http.StatusUnprocessableEntity, "validation_error", "Invalid input")
	default:
		logger.FromContext(c.Request.Context()).Error("${FEATURE}: unexpected error", "error", err)
		httpx.Error(c, http.StatusInternalServerError, "internal_error", "Internal server error")
	}
}
EOF
echo "  ✓ Created ${FEATURE_DIR}/handler.go"

# route.go
cat <<EOF > "${FEATURE_DIR}/route.go"
package ${FEATURE}

import "github.com/gin-gonic/gin"

// RegisterRoutes mounts the feature on a (versioned) router group.
func RegisterRoutes(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	g := rg.Group("/${FEATURE}")

	// Public routes:
	g.GET("", h.List)
	g.GET("/:id", h.Get)

	// Protected routes (requires auth):
	protected := g.Group("", authMiddleware)
	protected.POST("", h.Create)
	protected.PATCH("/:id", h.Update)
	protected.DELETE("/:id", h.Delete)
}
EOF
echo "  ✓ Created ${FEATURE_DIR}/route.go"

echo ""
echo "Feature '${FEATURE}' scaffolded successfully!"
echo ""
echo "Next steps:"
echo "  1. Review migration: ${MIGRATION_FILE}"
echo "  2. Apply migration:  make migrate-up"
echo "  3. Generate queries: make sqlc"
echo "  4. Wire into internal/server/router.go:"
echo "     import \"${MODULE}/${FEATURE_DIR}\""
echo ""
echo "     // In NewRouter dependency wiring:"
echo "     ${FEATURE}Handler := ${FEATURE}.NewHandler(${FEATURE}.NewService(q))"
echo ""
echo "     // In NewRouter route registration (v1):"
echo "     ${FEATURE}.RegisterRoutes(v1, ${FEATURE}Handler, authMiddleware)"
