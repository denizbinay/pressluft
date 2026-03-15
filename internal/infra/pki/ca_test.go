package pki

import (
	"crypto/ecdsa"
	"crypto/ed25519"
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

	"pressluft/internal/shared/security"
)

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

// testCA creates an in-memory database with the required schema, generates an
// age key, and returns a fully initialised CA plus the underlying DB so that
// callers can inspect state.
func testCA(t *testing.T) (*CA, *sql.DB) {
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

	return ca, db
}

func generateCSR(t *testing.T, cn string) *x509.CertificateRequest {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: cn},
	}, key)
	if err != nil {
		t.Fatalf("CreateCertificateRequest() error = %v", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		t.Fatalf("ParseCertificateRequest() error = %v", err)
	}
	return csr
}

func generateCSRPEM(t *testing.T, cn string) []byte {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: cn},
	}, key)
	if err != nil {
		t.Fatalf("CreateCertificateRequest() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})
}

// ---------------------------------------------------------------------------
// LoadOrCreateCA
// ---------------------------------------------------------------------------

func TestLoadOrCreateCA_CreatesNewCA(t *testing.T) {
	ca, db := testCA(t)

	if ca.cert == nil {
		t.Fatal("expected CA certificate, got nil")
	}
	if ca.key == nil {
		t.Fatal("expected CA private key, got nil")
	}
	if !ca.cert.IsCA {
		t.Fatal("CA certificate IsCA = false")
	}
	if ca.cert.Subject.CommonName != "Pressluft CA" {
		t.Fatalf("CA CN = %q, want %q", ca.cert.Subject.CommonName, "Pressluft CA")
	}
	// MaxPathLen 0 with MaxPathLenZero=true means "no sub-CAs allowed".
	// Go's x509 library represents this as MaxPathLen=0 when MaxPathLenZero
	// is true, but as -1 when MaxPathLenZero is false (which is the case
	// when the template sets MaxPathLen=0 without setting MaxPathLenZero).
	if ca.cert.MaxPathLen > 0 {
		t.Fatalf("MaxPathLen = %d, want <= 0", ca.cert.MaxPathLen)
	}
	if !ca.cert.BasicConstraintsValid {
		t.Fatal("BasicConstraintsValid = false")
	}

	// Certificate should be persisted in the database.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ca_certificates").Scan(&count); err != nil {
		t.Fatalf("count CA certs: %v", err)
	}
	if count != 1 {
		t.Fatalf("ca_certificates count = %d, want 1", count)
	}
}

func TestLoadOrCreateCA_LoadsExistingCA(t *testing.T) {
	// First call creates; second call with same paths should load the same CA.
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

	tmpDir := t.TempDir()
	agePath := filepath.Join(tmpDir, "age.key")
	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey() error = %v", err)
	}
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	ca1, err := LoadOrCreateCA(db, agePath, caKeyPath)
	if err != nil {
		t.Fatalf("first LoadOrCreateCA() error = %v", err)
	}

	ca2, err := LoadOrCreateCA(db, agePath, caKeyPath)
	if err != nil {
		t.Fatalf("second LoadOrCreateCA() error = %v", err)
	}

	if ca1.cert.SerialNumber.Cmp(ca2.cert.SerialNumber) != 0 {
		t.Fatal("serial numbers differ between first and second load")
	}

	// Still only one row.
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ca_certificates").Scan(&count); err != nil {
		t.Fatalf("count CA certs: %v", err)
	}
	if count != 1 {
		t.Fatalf("ca_certificates count = %d, want 1", count)
	}
}

// ---------------------------------------------------------------------------
// CA.Certificate / CA.CertPool
// ---------------------------------------------------------------------------

func TestCA_Certificate(t *testing.T) {
	ca, _ := testCA(t)

	cert := ca.Certificate()
	if cert == nil {
		t.Fatal("Certificate() returned nil")
	}
	if cert != ca.cert {
		t.Fatal("Certificate() did not return the internal cert")
	}
}

func TestCA_CertPool(t *testing.T) {
	ca, _ := testCA(t)

	pool := ca.CertPool()
	if pool == nil {
		t.Fatal("CertPool() returned nil")
	}
	// The pool should trust the CA cert.
	opts := x509.VerifyOptions{Roots: pool}
	// Self-signed CA should verify against its own pool (with CA usage).
	chains, err := ca.cert.Verify(opts)
	if err != nil {
		t.Fatalf("CA cert does not verify against its own pool: %v", err)
	}
	if len(chains) == 0 {
		t.Fatal("expected at least one chain")
	}
}

// ---------------------------------------------------------------------------
// CA.SignCSR
// ---------------------------------------------------------------------------

func TestSignCSR_ValidCSR(t *testing.T) {
	ca, _ := testCA(t)
	csr := generateCSR(t, "server:test-node")

	cert, err := ca.SignCSR(csr, 0)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}

	// Subject should carry over from the CSR.
	if cert.Subject.CommonName != "server:test-node" {
		t.Fatalf("CN = %q, want %q", cert.Subject.CommonName, "server:test-node")
	}

	// Certificate should be signed by the CA.
	if err := cert.CheckSignatureFrom(ca.cert); err != nil {
		t.Fatalf("CheckSignatureFrom() error = %v", err)
	}

	// Default validity should be ~90 days.
	expected := 90 * 24 * time.Hour
	actual := cert.NotAfter.Sub(cert.NotBefore)
	if actual < expected-time.Minute || actual > expected+time.Minute {
		t.Fatalf("validity = %v, want ~%v", actual, expected)
	}

	// KeyUsage should be DigitalSignature.
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		t.Fatal("missing KeyUsageDigitalSignature")
	}

	// ExtKeyUsage should contain ClientAuth.
	found := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing ExtKeyUsageClientAuth")
	}
}

func TestSignCSR_CustomValidity(t *testing.T) {
	ca, _ := testCA(t)
	csr := generateCSR(t, "server:custom-validity")

	cert, err := ca.SignCSR(csr, 30)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}

	expected := 30 * 24 * time.Hour
	actual := cert.NotAfter.Sub(cert.NotBefore)
	if actual < expected-time.Minute || actual > expected+time.Minute {
		t.Fatalf("validity = %v, want ~%v", actual, expected)
	}
}

func TestSignCSR_ChainVerification(t *testing.T) {
	ca, _ := testCA(t)
	csr := generateCSR(t, "server:chain-test")

	cert, err := ca.SignCSR(csr, 0)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}

	opts := x509.VerifyOptions{
		Roots:     ca.CertPool(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	chains, err := cert.Verify(opts)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if len(chains) == 0 {
		t.Fatal("expected at least one verification chain")
	}
	// The chain should be leaf → CA.
	if len(chains[0]) != 2 {
		t.Fatalf("chain length = %d, want 2", len(chains[0]))
	}
}

func TestSignCSR_BadSignature(t *testing.T) {
	ca, _ := testCA(t)

	// Create a CSR and tamper with its signature to make it invalid.
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "tampered"},
	}, key)
	if err != nil {
		t.Fatalf("CreateCertificateRequest() error = %v", err)
	}

	// Flip a byte in the signature area (near the end of DER).
	csrDER[len(csrDER)-1] ^= 0xff

	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		// Some implementations may reject at parse time — that's fine.
		t.Skipf("ParseCertificateRequest rejected tampered CSR: %v", err)
	}

	_, err = ca.SignCSR(csr, 0)
	if err == nil {
		t.Fatal("SignCSR() should reject CSR with bad signature")
	}
}

// ---------------------------------------------------------------------------
// GenerateServerCert
// ---------------------------------------------------------------------------

func TestGenerateServerCert(t *testing.T) {
	ca, _ := testCA(t)

	tlsCert, err := GenerateServerCert(ca, "node1.example.com")
	if err != nil {
		t.Fatalf("GenerateServerCert() error = %v", err)
	}

	if tlsCert.Leaf == nil {
		t.Fatal("Leaf is nil")
	}
	if tlsCert.Leaf.Subject.CommonName != "node1.example.com" {
		t.Fatalf("CN = %q, want %q", tlsCert.Leaf.Subject.CommonName, "node1.example.com")
	}

	// DNSNames should include the hostname and localhost.
	dnsNames := make(map[string]bool)
	for _, name := range tlsCert.Leaf.DNSNames {
		dnsNames[name] = true
	}
	if !dnsNames["node1.example.com"] {
		t.Fatal("missing hostname in DNSNames")
	}
	if !dnsNames["localhost"] {
		t.Fatal("missing localhost in DNSNames")
	}

	// Should have ServerAuth EKU.
	found := false
	for _, eku := range tlsCert.Leaf.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("missing ExtKeyUsageServerAuth")
	}

	// The leaf should be verifiable against the CA pool.
	opts := x509.VerifyOptions{
		Roots:     ca.CertPool(),
		DNSName:   "node1.example.com",
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	if _, err := tlsCert.Leaf.Verify(opts); err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
}

// ---------------------------------------------------------------------------
// ParseCertificateFromPEM / ParseCSRFromPEM
// ---------------------------------------------------------------------------

func TestParseCertificateFromPEM(t *testing.T) {
	ca, _ := testCA(t)
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: ca.cert.Raw})

	parsed, err := ParseCertificateFromPEM(certPEM)
	if err != nil {
		t.Fatalf("ParseCertificateFromPEM() error = %v", err)
	}
	if parsed.SerialNumber.Cmp(ca.cert.SerialNumber) != 0 {
		t.Fatal("parsed certificate serial differs from original")
	}
}

func TestParseCertificateFromPEM_NoPEM(t *testing.T) {
	_, err := ParseCertificateFromPEM([]byte("not pem data"))
	if err == nil {
		t.Fatal("expected error for non-PEM input")
	}
}

func TestParseCertificateFromPEM_Nil(t *testing.T) {
	_, err := ParseCertificateFromPEM(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestParseCSRFromPEM(t *testing.T) {
	csrPEM := generateCSRPEM(t, "server:pem-test")
	csr, err := ParseCSRFromPEM(csrPEM)
	if err != nil {
		t.Fatalf("ParseCSRFromPEM() error = %v", err)
	}
	if csr.Subject.CommonName != "server:pem-test" {
		t.Fatalf("CN = %q, want %q", csr.Subject.CommonName, "server:pem-test")
	}
}

func TestParseCSRFromPEM_NoPEM(t *testing.T) {
	_, err := ParseCSRFromPEM([]byte("garbage"))
	if err == nil {
		t.Fatal("expected error for non-PEM input")
	}
}

func TestParseCSRFromPEM_MalformedDER(t *testing.T) {
	malformed := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: []byte("not a real CSR")})
	_, err := ParseCSRFromPEM(malformed)
	if err == nil {
		t.Fatal("expected error for malformed DER in PEM")
	}
}

// ---------------------------------------------------------------------------
// calculateFingerprint
// ---------------------------------------------------------------------------

func TestCalculateFingerprint(t *testing.T) {
	ca, _ := testCA(t)
	fp := calculateFingerprint(ca.cert)

	if len(fp) == 0 {
		t.Fatal("fingerprint is empty")
	}
	if fp[:7] != "sha256:" {
		t.Fatalf("fingerprint prefix = %q, want %q", fp[:7], "sha256:")
	}
	// SHA-256 hex = 64 chars + "sha256:" prefix = 71 chars.
	if len(fp) != 71 {
		t.Fatalf("fingerprint length = %d, want 71", len(fp))
	}

	// Same cert produces the same fingerprint.
	fp2 := calculateFingerprint(ca.cert)
	if fp != fp2 {
		t.Fatal("fingerprint is not deterministic")
	}
}

// ---------------------------------------------------------------------------
// generateSerialNumber
// ---------------------------------------------------------------------------

func TestGenerateSerialNumber(t *testing.T) {
	sn1, err := generateSerialNumber()
	if err != nil {
		t.Fatalf("generateSerialNumber() error = %v", err)
	}
	sn2, err := generateSerialNumber()
	if err != nil {
		t.Fatalf("generateSerialNumber() error = %v", err)
	}
	if sn1.Cmp(sn2) == 0 {
		t.Fatal("two generated serial numbers are identical")
	}
	if sn1.Cmp(big.NewInt(0)) <= 0 {
		t.Fatal("serial number should be positive")
	}
}

// ---------------------------------------------------------------------------
// ValidateStoredCA
// ---------------------------------------------------------------------------

func TestValidateStoredCA_NoCert(t *testing.T) {
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE ca_certificates (id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')))`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	exists, err := ValidateStoredCA(db, "/nonexistent/age.key", "/nonexistent/ca.key")
	if err != nil {
		t.Fatalf("ValidateStoredCA() error = %v", err)
	}
	if exists {
		t.Fatal("expected exists = false for empty database")
	}
}

func TestValidateStoredCA_ExistingCA(t *testing.T) {
	// We need real paths that persist key files. Create a fresh CA setup.
	db2, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db2.Close() })

	for _, stmt := range []string{
		`CREATE TABLE ca_certificates (id TEXT PRIMARY KEY, fingerprint TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')))`,
	} {
		if _, err := db2.Exec(stmt); err != nil {
			t.Fatalf("exec %q: %v", stmt, err)
		}
	}

	tmpDir := t.TempDir()
	agePath := filepath.Join(tmpDir, "age.key")
	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey() error = %v", err)
	}
	caKeyPath := filepath.Join(tmpDir, "ca.key")

	// Create a CA (writes to db2 and key files).
	if _, err := LoadOrCreateCA(db2, agePath, caKeyPath); err != nil {
		t.Fatalf("LoadOrCreateCA() error = %v", err)
	}

	exists, err := ValidateStoredCA(db2, agePath, caKeyPath)
	if err != nil {
		t.Fatalf("ValidateStoredCA() error = %v", err)
	}
	if !exists {
		t.Fatal("expected exists = true for populated database")
	}
}

// ---------------------------------------------------------------------------
// SignCSR with ed25519 key (different key type)
// ---------------------------------------------------------------------------

func TestSignCSR_Ed25519Key(t *testing.T) {
	ca, _ := testCA(t)

	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		Subject: pkix.Name{CommonName: "server:ed25519-node"},
	}, key)
	if err != nil {
		t.Fatalf("CreateCertificateRequest() error = %v", err)
	}
	csr, err := x509.ParseCertificateRequest(csrDER)
	if err != nil {
		t.Fatalf("ParseCertificateRequest() error = %v", err)
	}

	cert, err := ca.SignCSR(csr, 7)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}

	if cert.Subject.CommonName != "server:ed25519-node" {
		t.Fatalf("CN = %q, want %q", cert.Subject.CommonName, "server:ed25519-node")
	}

	// Validity should be ~7 days.
	expected := 7 * 24 * time.Hour
	actual := cert.NotAfter.Sub(cert.NotBefore)
	if actual < expected-time.Minute || actual > expected+time.Minute {
		t.Fatalf("validity = %v, want ~%v", actual, expected)
	}
}
