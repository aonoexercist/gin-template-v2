-- name: CreateUser :one
INSERT INTO users (email, name, password_hash)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetUser :one
SELECT *
FROM users
WHERE id = $1;

-- name: ListUsers :many
SELECT *
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
RETURNING *;

-- name: DeleteUser :execrows
DELETE FROM users WHERE id = $1;
