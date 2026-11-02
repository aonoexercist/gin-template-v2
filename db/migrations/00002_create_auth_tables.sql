-- +goose Up
ALTER TABLE users ALTER COLUMN password_hash DROP NOT NULL;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_url text;

CREATE TABLE user_identities (
    id               uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id          uuid            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider         text            NOT NULL,
    provider_user_id text            NOT NULL,
    created_at       timestamptz     NOT NULL DEFAULT now(),
    updated_at       timestamptz     NOT NULL DEFAULT now(),
    CONSTRAINT       user_identities_provider_unique UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_user_identities_user_id ON user_identities(user_id);

CREATE TABLE sessions (
    id              uuid            PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         uuid            NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash      text            NOT NULL UNIQUE,
    user_agent      text,
    ip_address      text,
    expires_at      timestamptz     NOT NULL,
    created_at      timestamptz     NOT NULL DEFAULT now(),
    last_active_at  timestamptz     NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- +goose Down
DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS user_identities;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE users ALTER COLUMN password_hash SET NOT NULL;
