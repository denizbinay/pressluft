-- +goose Up
CREATE TABLE IF NOT EXISTS ca_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    fingerprint TEXT UNIQUE NOT NULL,
    certificate BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE TABLE IF NOT EXISTS node_certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL REFERENCES servers(id),
    fingerprint TEXT UNIQUE NOT NULL,
    serial_number TEXT UNIQUE NOT NULL,
    certificate BLOB NOT NULL,
    issued_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at TEXT NOT NULL,
    revoked_at TEXT
);

CREATE INDEX idx_node_certificates_server_id ON node_certificates(server_id);
CREATE INDEX idx_node_certificates_serial ON node_certificates(serial_number);

-- +goose Down
DROP INDEX IF EXISTS idx_node_certificates_serial;
DROP INDEX IF EXISTS idx_node_certificates_server_id;
DROP TABLE IF EXISTS node_certificates;
DROP TABLE IF EXISTS ca_certificates;
