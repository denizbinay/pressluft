-- +goose Up
CREATE TABLE IF NOT EXISTS domains (
    id               TEXT PRIMARY KEY,
    hostname         TEXT    NOT NULL COLLATE NOCASE,
    kind             TEXT    NOT NULL,
    ownership        TEXT    NOT NULL,
    source           TEXT    NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'active',
    site_id          TEXT,
    parent_domain_id TEXT,
    is_primary       INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_hostname_unique ON domains(hostname);
CREATE INDEX IF NOT EXISTS idx_domains_site_id ON domains(site_id);
CREATE INDEX IF NOT EXISTS idx_domains_parent_domain_id ON domains(parent_domain_id);
CREATE INDEX IF NOT EXISTS idx_domains_status ON domains(status);
CREATE INDEX IF NOT EXISTS idx_domains_kind ON domains(kind);
CREATE UNIQUE INDEX IF NOT EXISTS idx_domains_primary_site_unique ON domains(site_id) WHERE site_id IS NOT NULL AND is_primary = 1;

-- +goose Down
DROP INDEX IF EXISTS idx_domains_primary_site_unique;
DROP INDEX IF EXISTS idx_domains_kind;
DROP INDEX IF EXISTS idx_domains_status;
DROP INDEX IF EXISTS idx_domains_parent_domain_id;
DROP INDEX IF EXISTS idx_domains_site_id;
DROP INDEX IF EXISTS idx_domains_hostname_unique;
DROP TABLE IF EXISTS domains;
