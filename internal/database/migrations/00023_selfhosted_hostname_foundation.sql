-- +goose Up
PRAGMA foreign_keys = OFF;

ALTER TABLE domains RENAME TO domains_old;

CREATE TABLE domains (
    id                     TEXT PRIMARY KEY,
    hostname               TEXT    NOT NULL COLLATE NOCASE,
    kind                   TEXT    NOT NULL,
    source                 TEXT    NOT NULL,
    dns_state              TEXT    NOT NULL DEFAULT 'pending',
    routing_state          TEXT    NOT NULL DEFAULT 'not_configured',
    dns_status_message     TEXT,
    routing_status_message TEXT,
    last_checked_at        TEXT,
    site_id                TEXT,
    parent_domain_id       TEXT,
    is_primary             INTEGER NOT NULL DEFAULT 0,
    created_at             TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at             TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

INSERT INTO domains (
    id,
    hostname,
    kind,
    source,
    dns_state,
    routing_state,
    dns_status_message,
    routing_status_message,
    last_checked_at,
    site_id,
    parent_domain_id,
    is_primary,
    created_at,
    updated_at
)
SELECT
    id,
    hostname,
    CASE
        WHEN kind = 'wildcard' THEN 'base_domain'
        WHEN kind = 'direct' THEN 'hostname'
        ELSE kind
    END,
    CASE
        WHEN ownership = 'platform' THEN 'fallback_resolver'
        ELSE 'user'
    END,
    CASE
        WHEN status = 'active' THEN 'ready'
        WHEN status = 'attention' THEN 'issue'
        WHEN status = 'disabled' THEN 'disabled'
        ELSE 'pending'
    END,
    CASE
        WHEN site_id IS NULL THEN 'not_configured'
        WHEN status = 'attention' THEN 'issue'
        WHEN status = 'disabled' THEN 'issue'
        ELSE 'pending'
    END,
    NULL,
    NULL,
    NULL,
    site_id,
    CASE
        WHEN parent_domain_id IN (
            SELECT id
            FROM domains_old
            WHERE ownership = 'platform' AND kind = 'wildcard' AND site_id IS NULL AND parent_domain_id IS NULL
        ) THEN NULL
        ELSE parent_domain_id
    END,
    is_primary,
    created_at,
    updated_at
FROM domains_old
WHERE NOT (ownership = 'platform' AND kind = 'wildcard' AND site_id IS NULL AND parent_domain_id IS NULL);

DROP TABLE domains_old;

CREATE UNIQUE INDEX idx_domains_hostname_unique ON domains(hostname);
CREATE INDEX idx_domains_site_id ON domains(site_id);
CREATE INDEX idx_domains_parent_domain_id ON domains(parent_domain_id);
CREATE INDEX idx_domains_dns_state ON domains(dns_state);
CREATE INDEX idx_domains_routing_state ON domains(routing_state);
CREATE INDEX idx_domains_kind ON domains(kind);
CREATE INDEX idx_domains_source ON domains(source);
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
    status           TEXT    NOT NULL DEFAULT 'active',
    site_id          TEXT,
    parent_domain_id TEXT,
    is_primary       INTEGER NOT NULL DEFAULT 0,
    created_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at       TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (site_id) REFERENCES sites(id) ON DELETE SET NULL,
    FOREIGN KEY (parent_domain_id) REFERENCES domains(id) ON DELETE SET NULL
);

INSERT INTO domains (
    id,
    hostname,
    kind,
    ownership,
    status,
    site_id,
    parent_domain_id,
    is_primary,
    created_at,
    updated_at
)
SELECT
    id,
    hostname,
    CASE
        WHEN kind = 'base_domain' THEN 'wildcard'
        WHEN kind = 'hostname' THEN 'direct'
        ELSE kind
    END,
    CASE
        WHEN source = 'fallback_resolver' THEN 'platform'
        ELSE 'customer'
    END,
    CASE
        WHEN dns_state = 'ready' THEN 'active'
        WHEN dns_state = 'issue' THEN 'attention'
        WHEN dns_state = 'disabled' THEN 'disabled'
        ELSE 'pending'
    END,
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
