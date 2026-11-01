-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING id, email, name, password_hash, created_at, updated_at;

-- name: GetUser :one
SELECT id, email, name, created_at, updated_at
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT id, email, name, created_at, updated_at
FROM users
ORDER BY created_at DESC, id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountUsers :one
SELECT count(*) FROM users;

-- name: UpdateUser :one
UPDATE users
SET name       = COALESCE(sqlc.narg(name), name),
    email      = COALESCE(sqlc.narg(email), email),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING id, email, name, created_at, updated_at;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
