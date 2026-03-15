-- +goose Up
ALTER TABLE sites ADD COLUMN wordpress_admin_email TEXT;

-- +goose Down
ALTER TABLE sites DROP COLUMN wordpress_admin_email;
