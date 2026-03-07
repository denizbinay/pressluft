-- +goose Up
ALTER TABLE providers ADD COLUMN api_token_encrypted TEXT;
ALTER TABLE providers ADD COLUMN api_token_key_id TEXT;
ALTER TABLE providers ADD COLUMN api_token_version INTEGER NOT NULL DEFAULT 0;

-- +goose Down
