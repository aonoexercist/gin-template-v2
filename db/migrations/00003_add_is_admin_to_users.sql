-- +goose Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_admin boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE users DROP COLUMN IF EXISTS is_admin;
