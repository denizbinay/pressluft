-- +goose Up
ALTER TABLE servers ADD COLUMN ipv4 TEXT;
ALTER TABLE servers ADD COLUMN ipv6 TEXT;

CREATE INDEX IF NOT EXISTS idx_servers_ipv4 ON servers(ipv4);
CREATE INDEX IF NOT EXISTS idx_servers_ipv6 ON servers(ipv6);

-- +goose Down
DROP INDEX IF EXISTS idx_servers_ipv6;
DROP INDEX IF EXISTS idx_servers_ipv4;

ALTER TABLE servers DROP COLUMN ipv6;
ALTER TABLE servers DROP COLUMN ipv4;
