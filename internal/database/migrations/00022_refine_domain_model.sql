-- +goose Up
PRAGMA foreign_keys = OFF;

ALTER TABLE domains RENAME TO domains_old;

CREATE TABLE domains (
    id               TEXT PRIMARY KEY,
    hostname         TEXT    NOT NULL COLLATE NOCASE,
    kind             TEXT    NOT NULL,
    ownership        TEXT    NOT NULL,
    status           TEXT    NOT NULL DEFAULT 'active',
    site_id          TEXT,
    parent_domain_id TEXT,
    is_primary       INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

INSERT INTO domains (id, hostname, kind, ownership, status, site_id, parent_domain_id, is_primary, created_at, updated_at)
SELECT
    id,
    hostname,
    CASE
        WHEN kind = 'base' THEN 'wildcard'
        WHEN kind = 'hostname' THEN 'direct'
        ELSE kind
    END,
    CASE
        WHEN hostname IN ('pressluft.bombig.app', 'pressluft.dev') THEN 'platform'
        WHEN ownership = 'platform' AND kind = 'base' AND site_id IS NULL AND parent_domain_id IS NULL THEN 'customer'
        ELSE ownership
    END,
    status,
    site_id,
    parent_domain_id,
    is_primary,
    created_at,
    updated_at
FROM domains_old;

DROP TABLE domains_old;

CREATE UNIQUE INDEX idx_domains_hostname_unique ON domains(hostname);
CREATE INDEX idx_domains_site_id ON domains(site_id);
CREATE INDEX idx_domains_parent_domain_id ON domains(parent_domain_id);
CREATE INDEX idx_domains_status ON domains(status);
CREATE INDEX idx_domains_kind ON domains(kind);
CREATE UNIQUE INDEX idx_domains_primary_site_unique ON domains(site_id) WHERE site_id IS NOT NULL AND is_primary = 1;

PRAGMA foreign_keys = ON;

-- +goose Down
PRAGMA foreign_keys = OFF;

ALTER TABLE domains RENAME TO domains_new;

CREATE TABLE domains (
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

INSERT INTO domains (id, hostname, kind, ownership, source, status, site_id, parent_domain_id, is_primary, created_at, updated_at)
SELECT
    id,
    hostname,
    CASE
        WHEN kind = 'wildcard' THEN 'base'
        WHEN kind = 'direct' THEN 'hostname'
        ELSE kind
    END,
    ownership,
    CASE
        WHEN parent_domain_id IS NOT NULL THEN 'sandbox'
        WHEN ownership = 'platform' AND kind = 'wildcard' THEN 'sandbox'
        ELSE 'custom'
    END,
    status,
    site_id,
    parent_domain_id,
    is_primary,
    created_at,
    updated_at
FROM domains_new;

DROP TABLE domains_new;

CREATE UNIQUE INDEX idx_domains_hostname_unique ON domains(hostname);
CREATE INDEX idx_domains_site_id ON domains(site_id);
CREATE INDEX idx_domains_parent_domain_id ON domains(parent_domain_id);
CREATE INDEX idx_domains_status ON domains(status);
CREATE INDEX idx_domains_kind ON domains(kind);
CREATE UNIQUE INDEX idx_domains_primary_site_unique ON domains(site_id) WHERE site_id IS NOT NULL AND is_primary = 1;

PRAGMA foreign_keys = ON;
