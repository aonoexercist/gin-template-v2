-- name: GetUserByIdentity :one
SELECT u.id, u.email, u.name, u.avatar_url, u.is_admin, u.created_at, u.updated_at
FROM users u
JOIN user_identities ui ON ui.user_id = u.id
WHERE ui.provider = $1 AND ui.provider_user_id = $2;

-- name: GetUserByEmail :one
SELECT id, email, name, avatar_url, is_admin, created_at, updated_at
FROM users
WHERE email = $1;

-- name: GetUserWithPasswordByEmail :one
SELECT id, email, name, password_hash, avatar_url, is_admin, created_at, updated_at
FROM users
WHERE email = $1;

-- name: CreateOAuthUser :one
INSERT INTO users (email, name, avatar_url)
VALUES ($1, $2, $3)
RETURNING id, email, name, avatar_url, is_admin, created_at, updated_at;

-- name: UpdateUserAvatarAndName :one
UPDATE users
SET avatar_url = COALESCE(sqlc.narg(avatar_url), avatar_url),
    name = COALESCE(sqlc.narg(name), name),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING id, email, name, avatar_url, is_admin, created_at, updated_at;

-- name: CreateIdentity :one
INSERT INTO user_identities (user_id, provider, provider_user_id)
VALUES ($1, $2, $3)
RETURNING id, user_id, provider, provider_user_id, created_at, updated_at;

-- name: CreateSession :one
INSERT INTO sessions (user_id, token_hash, user_agent, ip_address, expires_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, last_active_at;

-- name: GetSessionAndUserByTokenHash :one
SELECT
    s.id AS session_id,
    s.user_id,
    s.token_hash,
    s.user_agent,
    s.ip_address,
    s.expires_at,
    s.created_at AS session_created_at,
    s.last_active_at,
    u.id AS user_pk,
    u.email,
    u.name,
    u.avatar_url,
    u.is_admin,
    u.created_at AS user_created_at,
    u.updated_at AS user_updated_at
FROM sessions s
JOIN users u ON u.id = s.user_id
WHERE s.token_hash = $1 AND s.expires_at > now();

-- name: TouchSession :exec
UPDATE sessions
SET last_active_at = now()
WHERE id = $1;

-- name: DeleteSessionByTokenHash :execrows
DELETE FROM sessions
WHERE token_hash = $1;

-- name: DeleteSessionByID :execrows
DELETE FROM sessions
WHERE id = $1 AND user_id = $2;

-- name: DeleteOtherSessionsForUser :execrows
DELETE FROM sessions
WHERE user_id = $1 AND token_hash != $2;

-- name: DeleteAllSessionsForUser :execrows
DELETE FROM sessions
WHERE user_id = $1;

-- name: ListActiveSessionsForUser :many
SELECT id, user_id, token_hash, user_agent, ip_address, expires_at, created_at, last_active_at
FROM sessions
WHERE user_id = $1 AND expires_at > now()
ORDER BY last_active_at DESC;

-- name: CleanExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at <= now();
