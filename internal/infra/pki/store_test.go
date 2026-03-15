package pki

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/pem"
	"math/big"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"pressluft/internal/shared/idutil"
	"pressluft/internal/shared/security"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

const testServerID = "019573a0-0000-7000-8000-000000000001"

func testStore(t *testing.T) (*Store, *CA, *sql.DB) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, stmt := range []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE servers (id TEXT PRIMARY KEY)`,
		`CREATE TABLE ca_certificates (id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')))`,
		`CREATE TABLE node_certificates (id TEXT PRIMARY KEY, server_id TEXT NOT NULL REFERENCES servers(id), fingerprint TEXT UNIQUE NOT NULL, serial_number TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, issued_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), expires_at TEXT NOT NULL, revoked_at TEXT)`,
	} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}

	// Insert a test server.
	if _, err := db.Exec(`INSERT INTO servers (id) VALUES (?)`, testServerID); err != nil {
		t.Fatalf("insert server: %v", err)
	}

	tmpDir := t.TempDir()
	agePath := filepath.Join(tmpDir, "age.key")
	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey() error = %v", err)
	}
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	ca, err := LoadOrCreateCA(db, agePath, caKeyPath)
	if err != nil {
		t.Fatalf("LoadOrCreateCA() error = %v", err)
	}

	return NewStore(db), ca, db
}

// signTestCert creates a signed certificate for a given server CN using the CA.
func signTestCert(t *testing.T, ca *CA, serverID string) *x509.Certificate {
	t.Helper()
	csr := generateCSR(t, "server:"+serverID)
	cert, err := ca.SignCSR(csr, 90)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}
	return cert
}

// signExpiredCert creates a certificate that is already expired by
// directly issuing with a past NotAfter. We can't use SignCSR for this
// because it always sets NotAfter in the future, so we craft one manually.
func signExpiredCert(t *testing.T, ca *CA, serverID string) *x509.Certificate {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	serial := make([]byte, 20)
	if _, err := rand.Read(serial); err != nil {
		t.Fatalf("generate serial: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: new(big.Int).SetBytes(serial),
		Subject:      pkix.Name{CommonName: "server:" + serverID},
		NotBefore:    time.Now().Add(-48 * time.Hour),
		NotAfter:     time.Now().Add(-24 * time.Hour), // expired yesterday
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca.cert, &key.PublicKey, ca.key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	cert, err := x509.ParseCertificate(certDER)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	return cert
}

// ---------------------------------------------------------------------------
// SaveNodeCertificate + GetValidCertForServer
// ---------------------------------------------------------------------------

func TestSaveAndGetValidCert(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}

	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc == nil {
		t.Fatal("expected node certificate, got nil")
	}
	if nc.ServerID != testServerID {
		t.Fatalf("ServerID = %q, want %q", nc.ServerID, testServerID)
	}
	if nc.SerialNumber != cert.SerialNumber.String() {
		t.Fatalf("SerialNumber = %q, want %q", nc.SerialNumber, cert.SerialNumber.String())
	}
	if nc.Fingerprint == "" {
		t.Fatal("Fingerprint is empty")
	}
	if nc.RevokedAt != nil {
		t.Fatal("expected RevokedAt to be nil")
	}
}

func TestSaveNodeCertificate_MissingServer(t *testing.T) {
	store, ca, _ := testStore(t)

	// Use a server ID that doesn't exist in the servers table.
	missingID := idutil.MustNew()
	cert := signTestCert(t, ca, missingID)

	err := store.SaveNodeCertificate(missingID, cert)
	if err == nil {
		t.Fatal("expected error when saving cert for non-existent server")
	}
}

func TestSaveNodeCertificate_EmptyServerID(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	err := store.SaveNodeCertificate("", cert)
	if err == nil {
		t.Fatal("expected error for empty server ID")
	}
}

// ---------------------------------------------------------------------------
// GetValidCertForServer - edge cases
// ---------------------------------------------------------------------------

func TestGetValidCertForServer_NoCert(t *testing.T) {
	store, _, _ := testStore(t)

	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc != nil {
		t.Fatal("expected nil for server with no certificates")
	}
}

func TestGetValidCertForServer_ExpiredCert(t *testing.T) {
	store, ca, db := testStore(t)

	cert := signExpiredCert(t, ca, testServerID)

	// Insert the expired cert manually.
	certID := idutil.MustNew()
	fingerprint := calculateFingerprint(cert)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	_, err := db.Exec(`
		INSERT INTO node_certificates (id, server_id, fingerprint, serial_number, certificate, issued_at, expires_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, certID, testServerID, fingerprint, cert.SerialNumber.String(), certPEM,
		cert.NotBefore.Format(time.RFC3339), cert.NotAfter.Format(time.RFC3339))
	if err != nil {
		t.Fatalf("insert expired cert: %v", err)
	}

	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc != nil {
		t.Fatal("expected nil for server with only expired certificate")
	}
}

func TestGetValidCertForServer_ReturnsLatest(t *testing.T) {
	store, ca, db := testStore(t)

	cert1 := signTestCert(t, ca, testServerID)
	if err := store.SaveNodeCertificate(testServerID, cert1); err != nil {
		t.Fatalf("SaveNodeCertificate(cert1) error = %v", err)
	}

	// Backdate cert1's issued_at so cert2 is strictly newer.
	// issued_at uses RFC3339 with second resolution so time.Sleep alone
	// is unreliable without sleeping >1s.
	_, err := db.Exec(`UPDATE node_certificates SET issued_at = ? WHERE serial_number = ?`,
		time.Now().Add(-time.Hour).UTC().Format(time.RFC3339), cert1.SerialNumber.String())
	if err != nil {
		t.Fatalf("backdate cert1: %v", err)
	}

	cert2 := signTestCert(t, ca, testServerID)
	if err := store.SaveNodeCertificate(testServerID, cert2); err != nil {
		t.Fatalf("SaveNodeCertificate(cert2) error = %v", err)
	}

	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc == nil {
		t.Fatal("expected node certificate, got nil")
	}
	// Should be the second (latest) cert.
	if nc.SerialNumber != cert2.SerialNumber.String() {
		t.Fatalf("SerialNumber = %q, want latest %q", nc.SerialNumber, cert2.SerialNumber.String())
	}
}

// ---------------------------------------------------------------------------
// RevokeCertificate / IsRevoked
// ---------------------------------------------------------------------------

func TestRevokeCertificate(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}

	serial := cert.SerialNumber.String()

	if store.IsRevoked(serial) {
		t.Fatal("cert should not be revoked yet")
	}

	if err := store.RevokeCertificate(serial); err != nil {
		t.Fatalf("RevokeCertificate() error = %v", err)
	}

	if !store.IsRevoked(serial) {
		t.Fatal("cert should be revoked after RevokeCertificate()")
	}

	// After revocation, GetValidCertForServer should not return it.
	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc != nil {
		t.Fatal("expected nil for server with only revoked certificate")
	}
}

func TestIsRevoked_UnknownSerial(t *testing.T) {
	store, _, _ := testStore(t)
	if store.IsRevoked("999999999") {
		t.Fatal("unknown serial should not be reported as revoked")
	}
}

func TestRevokeCertificateTx(t *testing.T) {
	store, ca, db := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}

	serial := cert.SerialNumber.String()

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := store.RevokeCertificateTx(context.Background(), tx, serial); err != nil {
		_ = tx.Rollback()
		t.Fatalf("RevokeCertificateTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	if !store.IsRevoked(serial) {
		t.Fatal("cert should be revoked after RevokeCertificateTx()")
	}
}

// ---------------------------------------------------------------------------
// GetCACertificate
// ---------------------------------------------------------------------------

func TestGetCACertificate(t *testing.T) {
	store, ca, _ := testStore(t)

	retrieved, err := store.GetCACertificate()
	if err != nil {
		t.Fatalf("GetCACertificate() error = %v", err)
	}
	if retrieved.SerialNumber.Cmp(ca.cert.SerialNumber) != 0 {
		t.Fatal("retrieved CA serial does not match")
	}
}

func TestGetCACertificate_NoCert(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`CREATE TABLE ca_certificates (id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')))`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	store := NewStore(db)
	_, err = store.GetCACertificate()
	if err == nil {
		t.Fatal("expected error when no CA certificate exists")
	}
}

// ---------------------------------------------------------------------------
// GetCertificatePEMForServer
// ---------------------------------------------------------------------------

func TestGetCertificatePEMForServer(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}

	pemData, err := store.GetCertificatePEMForServer(testServerID)
	if err != nil {
		t.Fatalf("GetCertificatePEMForServer() error = %v", err)
	}
	if pemData == nil {
		t.Fatal("expected PEM data, got nil")
	}

	// Should be valid PEM that parses back to the same certificate.
	block, _ := pem.Decode(pemData)
	if block == nil {
		t.Fatal("returned data is not valid PEM")
	}
	parsed, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	if parsed.SerialNumber.Cmp(cert.SerialNumber) != 0 {
		t.Fatal("parsed certificate serial differs from original")
	}
}

func TestGetCertificatePEMForServer_NoCert(t *testing.T) {
	store, _, _ := testStore(t)

	pemData, err := store.GetCertificatePEMForServer(testServerID)
	if err != nil {
		t.Fatalf("GetCertificatePEMForServer() error = %v", err)
	}
	if pemData != nil {
		t.Fatalf("expected nil PEM data, got %d bytes", len(pemData))
	}
}

func TestGetCertificatePEMForServer_SkipsRevoked(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}
	if err := store.RevokeCertificate(cert.SerialNumber.String()); err != nil {
		t.Fatalf("RevokeCertificate() error = %v", err)
	}

	pemData, err := store.GetCertificatePEMForServer(testServerID)
	if err != nil {
		t.Fatalf("GetCertificatePEMForServer() error = %v", err)
	}
	if pemData != nil {
		t.Fatal("expected nil for revoked certificate")
	}
}

// ---------------------------------------------------------------------------
// ServerIDFromCN
// ---------------------------------------------------------------------------

func TestServerIDFromCN(t *testing.T) {
	store, _, _ := testStore(t)

	id, err := store.ServerIDFromCN("server:" + testServerID)
	if err != nil {
		t.Fatalf("ServerIDFromCN() error = %v", err)
	}
	if id != testServerID {
		t.Fatalf("ServerIDFromCN() = %q, want %q", id, testServerID)
	}
}

func TestServerIDFromCN_InvalidPrefix(t *testing.T) {
	store, _, _ := testStore(t)

	_, err := store.ServerIDFromCN("node:some-id")
	if err == nil {
		t.Fatal("expected error for CN without 'server:' prefix")
	}
}

func TestServerIDFromCN_EmptyCN(t *testing.T) {
	store, _, _ := testStore(t)

	_, err := store.ServerIDFromCN("")
	if err == nil {
		t.Fatal("expected error for empty CN")
	}
}

func TestServerIDFromCN_InvalidUUID(t *testing.T) {
	store, _, _ := testStore(t)

	_, err := store.ServerIDFromCN("server:not-a-valid-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID in CN")
	}
}

// ---------------------------------------------------------------------------
// SaveNodeCertificateTx with explicit transaction
// ---------------------------------------------------------------------------

func TestSaveNodeCertificateTx(t *testing.T) {
	store, ca, db := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := store.SaveNodeCertificateTx(context.Background(), tx, testServerID, cert); err != nil {
		_ = tx.Rollback()
		t.Fatalf("SaveNodeCertificateTx() error = %v", err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatalf("Commit() error = %v", err)
	}

	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc == nil {
		t.Fatal("expected node certificate after tx commit")
	}
}

func TestSaveNodeCertificateTx_Rollback(t *testing.T) {
	store, ca, db := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("Begin() error = %v", err)
	}

	if err := store.SaveNodeCertificateTx(context.Background(), tx, testServerID, cert); err != nil {
		_ = tx.Rollback()
		t.Fatalf("SaveNodeCertificateTx() error = %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}

	// Certificate should not exist after rollback.
	nc, err := store.GetValidCertForServer(testServerID)
	if err != nil {
		t.Fatalf("GetValidCertForServer() error = %v", err)
	}
	if nc != nil {
		t.Fatal("expected nil after transaction rollback")
	}
}

// ---------------------------------------------------------------------------
// lookupServerID (indirectly tested through Store methods)
// ---------------------------------------------------------------------------

func TestLookupServerID_Whitespace(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	// Server ID with leading/trailing whitespace should still resolve.
	err := store.SaveNodeCertificate("  "+testServerID+"  ", cert)
	if err != nil {
		t.Fatalf("SaveNodeCertificate with whitespace ID error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// Duplicate fingerprint handling
// ---------------------------------------------------------------------------

func TestSaveNodeCertificate_DuplicateFingerprint(t *testing.T) {
	store, ca, _ := testStore(t)
	cert := signTestCert(t, ca, testServerID)

	if err := store.SaveNodeCertificate(testServerID, cert); err != nil {
		t.Fatalf("first SaveNodeCertificate() error = %v", err)
	}

	// Saving the exact same certificate again should fail due to UNIQUE
	// constraint on fingerprint.
	err := store.SaveNodeCertificate(testServerID, cert)
	if err == nil {
		t.Fatal("expected error when saving duplicate certificate")
	}
}
