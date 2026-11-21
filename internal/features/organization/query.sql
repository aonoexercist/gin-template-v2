-- name: CreateOrganization :one
INSERT INTO organizations (name)
VALUES ($1)
RETURNING id, name, created_at, updated_at;

-- name: GetOrganization :one
SELECT id, name, created_at, updated_at
FROM organizations
WHERE id = $1;

-- name: ListOrganizations :many
SELECT id, name, created_at, updated_at
FROM organizations
ORDER BY created_at DESC, id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountOrganizations :one
SELECT count(*) FROM organizations;

-- name: UpdateOrganization :one
UPDATE organizations
SET name       = COALESCE(sqlc.narg(name), name),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING id, name, created_at, updated_at;

-- name: DeleteOrganization :execrows
DELETE FROM organizations WHERE id = $1;
