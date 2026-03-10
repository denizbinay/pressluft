package pki

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/idutil"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) SaveNodeCertificate(serverID string, cert *x509.Certificate) error {
	return s.SaveNodeCertificateTx(context.Background(), nil, serverID, cert)
}

func (s *Store) SaveNodeCertificateTx(ctx context.Context, tx *sql.Tx, serverID string, cert *x509.Certificate) error {
	serverID, err := s.lookupServerID(ctx, tx, serverID)
	if err != nil {
		return err
	}
	certID, err := idutil.New()
	if err != nil {
		return err
	}
	fingerprint := calculateFingerprint(cert)
	serialNumber := cert.SerialNumber.String()
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	exec := execOrDB(tx, s.db)

	_, err = exec.ExecContext(ctx, `
		INSERT INTO node_certificates (id, server_id, fingerprint, serial_number, certificate, issued_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, certID, serverID, fingerprint, serialNumber, certPEM, time.Now().UTC().Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))

	return err
}

func (s *Store) GetValidCertForServer(serverID string) (*NodeCertificate, error) {
	return s.GetValidCertForServerTx(context.Background(), nil, serverID)
}

func (s *Store) GetValidCertForServerTx(ctx context.Context, tx *sql.Tx, serverID string) (*NodeCertificate, error) {
	serverID, err := s.lookupServerID(ctx, tx, serverID)
	if err != nil {
		return nil, err
	}
	var (
		nc           NodeCertificate
		issuedAtRaw  string
		expiresAtRaw string
		revokedAtRaw sql.NullString
	)
	exec := execOrDB(tx, s.db)
	err = exec.QueryRowContext(ctx, `
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

func (s *Store) GetCertificatePEMForServer(serverID string) ([]byte, error) {
	serverID, err := s.lookupServerID(context.Background(), nil, serverID)
	if err != nil {
		return nil, err
	}
	var certPEM []byte
	err = s.db.QueryRow(`
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

func (s *Store) ServerIDFromCN(cn string) (string, error) {
	if !strings.HasPrefix(cn, "server:") {
		return "", fmt.Errorf("invalid CN format")
	}
	serverID, err := idutil.Normalize(strings.TrimSpace(cn[7:]))
	if err != nil {
		return "", err
	}
	return serverID, nil
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

func (s *Store) lookupServerID(ctx context.Context, tx *sql.Tx, serverID string) (string, error) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return "", fmt.Errorf("server_id is required")
	}
	exec := execOrDB(tx, s.db)
	var storedID string
	if err := exec.QueryRowContext(ctx, `SELECT id FROM servers WHERE id = ?`, serverID).Scan(&storedID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("server %s not found", serverID)
		}
		return "", fmt.Errorf("lookup server id: %w", err)
	}
	return storedID, nil
}
