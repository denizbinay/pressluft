package pki

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"strings"
	"time"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveNodeCertificate(serverID int64, cert *x509.Certificate) error {
	return s.SaveNodeCertificateTx(context.Background(), nil, serverID, cert)
}

func (s *Store) SaveNodeCertificateTx(ctx context.Context, tx *sql.Tx, serverID int64, cert *x509.Certificate) error {
	fingerprint := calculateFingerprint(cert)
	serialNumber := cert.SerialNumber.String()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	exec := execOrDB(tx, s.db)

	_, err := exec.ExecContext(ctx, `
		INSERT INTO node_certificates (server_id, fingerprint, serial_number, certificate, issued_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, serverID, fingerprint, serialNumber, certPEM, time.Now().UTC().Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))

	return err
}

func (s *Store) GetValidCertForServer(serverID int64) (*NodeCertificate, error) {
	return s.GetValidCertForServerTx(context.Background(), nil, serverID)
}

func (s *Store) GetValidCertForServerTx(ctx context.Context, tx *sql.Tx, serverID int64) (*NodeCertificate, error) {
	var (
		nc           NodeCertificate
		issuedAtRaw  string
		expiresAtRaw string
		revokedAtRaw sql.NullString
	)
	exec := execOrDB(tx, s.db)
	err := exec.QueryRowContext(ctx, `
		SELECT id, server_id, fingerprint, serial_number, issued_at, expires_at, revoked_at
		FROM node_certificates
		WHERE server_id = ?
		  AND revoked_at IS NULL
		  AND datetime(expires_at) > datetime('now')
		ORDER BY issued_at DESC
		LIMIT 1
	`, serverID).Scan(&nc.ID, &nc.ServerID, &nc.Fingerprint, &nc.SerialNumber, &issuedAtRaw, &expiresAtRaw, &revokedAtRaw)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if nc.IssuedAt, err = time.Parse(time.RFC3339, issuedAtRaw); err != nil {
		return nil, fmt.Errorf("parse issued_at: %w", err)
	}
	if nc.ExpiresAt, err = time.Parse(time.RFC3339, expiresAtRaw); err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	if revokedAtRaw.Valid {
		revokedAt, err := time.Parse(time.RFC3339, revokedAtRaw.String)
		if err != nil {
			return nil, fmt.Errorf("parse revoked_at: %w", err)
		}
		nc.RevokedAt = &revokedAt
	}

	return &nc, nil
}

func (s *Store) IsRevoked(serialNumber string) bool {
	var revokedAt *time.Time
	err := s.db.QueryRow("SELECT revoked_at FROM node_certificates WHERE serial_number = ?", serialNumber).Scan(&revokedAt)
	if err == sql.ErrNoRows {
		return false
	}
	return revokedAt != nil
}

func (s *Store) RevokeCertificate(serialNumber string) error {
	return s.RevokeCertificateTx(context.Background(), nil, serialNumber)
}

func (s *Store) RevokeCertificateTx(ctx context.Context, tx *sql.Tx, serialNumber string) error {
	exec := execOrDB(tx, s.db)
	_, err := exec.ExecContext(ctx, `
		UPDATE node_certificates
		SET revoked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE serial_number = ?
	`, serialNumber)
	return err
}

func (s *Store) GetCACertificate() (*x509.Certificate, error) {
	var certPEM []byte
	err := s.db.QueryRow("SELECT certificate FROM ca_certificates ORDER BY id DESC LIMIT 1").Scan(&certPEM)
	if err != nil {
		return nil, err
	}

	cert, err := ParseCertificateFromPEM(certPEM)
	if err != nil {
		return nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	return cert, nil
}

func (s *Store) GetCertificatePEMForServer(serverID int64) ([]byte, error) {
	var certPEM []byte
	err := s.db.QueryRow(`
		SELECT certificate
		FROM node_certificates
		WHERE server_id = ?
		  AND revoked_at IS NULL
		  AND datetime(expires_at) > datetime('now')
		ORDER BY issued_at DESC
		LIMIT 1
	`, serverID).Scan(&certPEM)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if block, _ := pem.Decode(certPEM); block == nil {
		certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certPEM})
	}

	return certPEM, nil
}

func (s *Store) ServerIDFromCN(cn string) (int64, error) {
	if !strings.HasPrefix(cn, "server-") {
		return 0, fmt.Errorf("invalid CN format")
	}
	return parseServerID(cn[7:])
}

func parseServerID(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid server ID")
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}

type sqlExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func execOrDB(tx *sql.Tx, db *sql.DB) sqlExecutor {
	if tx != nil {
		return tx
	}
	return db
}
