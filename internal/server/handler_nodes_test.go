package server

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"log/slog"
	"math/big"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"pressluft/internal/pki"
	"pressluft/internal/registration"
	"pressluft/internal/security"

	_ "modernc.org/sqlite"
)

func TestNodeRegisterConsumesTokenOnce(t *testing.T) {
	h, stores := newNodeHandlerTestHarness(t)
	token, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	body := registerRequestBody(t, 1, token)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(body))
	h.handleNodeRegister(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("first status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := stores.registration.Validate(token, 1); !errors.Is(err, registration.ErrConsumedToken) {
		t.Fatalf("Validate() after consume error = %v, want %v", err, registration.ErrConsumedToken)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(body))
	h.handleNodeRegister(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("replay status = %d, body = %s", rec.Code, rec.Body.String())
	}
}

func TestNodeRegisterRejectsExpiredTokenAndCNMismatchWithoutConsuming(t *testing.T) {
	h, stores := newNodeHandlerTestHarness(t)

	expired, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() expired error = %v", err)
	}
	if _, err := stores.db.Exec(`UPDATE registration_tokens SET expires_at = '2000-01-01T00:00:00Z' WHERE token_hash = ?`, registration.HashToken(expired)); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(registerRequestBody(t, 1, expired)))
	h.handleNodeRegister(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cnMismatch, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() cn mismatch error = %v", err)
	}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(registerRequestBody(t, 2, cnMismatch)))
	h.handleNodeRegister(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("cn mismatch status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := stores.registration.Validate(cnMismatch, 1); err != nil {
		t.Fatalf("Validate() after CN mismatch error = %v", err)
	}
}

func TestNodeRegisterBlocksExistingValidCert(t *testing.T) {
	h, stores := newNodeHandlerTestHarness(t)
	csr := newCSR(t, 1)
	cert, err := stores.ca.SignCSR(csr, 90)
	if err != nil {
		t.Fatalf("SignCSR() error = %v", err)
	}
	if err := stores.pki.SaveNodeCertificate(1, cert); err != nil {
		t.Fatalf("SaveNodeCertificate() error = %v", err)
	}
	token, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(registerRequestBody(t, 1, token)))
	h.handleNodeRegister(rec, req)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := stores.registration.Validate(token, 1); err != nil {
		t.Fatalf("Validate() after conflict error = %v", err)
	}
}

func TestNodeRegisterKeepsTokenWhenCAOrPersistenceFails(t *testing.T) {
	base, stores := newNodeHandlerTestHarness(t)

	caToken, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() ca token error = %v", err)
	}
	caFail := &NodeHandler{db: stores.db, pkiStore: stores.pki, registrationStore: stores.registration, ca: failingCA{err: errors.New("boom")}, logger: base.logger}
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(registerRequestBody(t, 1, caToken)))
	caFail.handleNodeRegister(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("ca failure status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := stores.registration.Validate(caToken, 1); err != nil {
		t.Fatalf("Validate() after CA failure error = %v", err)
	}

	persistToken, err := stores.registration.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() persistence token error = %v", err)
	}
	persistFail := &NodeHandler{db: stores.db, pkiStore: failingPKIStore{stores.pki}, registrationStore: stores.registration, ca: stores.ca, logger: base.logger}
	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/nodes/1/register", bytes.NewReader(registerRequestBody(t, 1, persistToken)))
	persistFail.handleNodeRegister(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("persistence failure status = %d, body = %s", rec.Code, rec.Body.String())
	}
	if err := stores.registration.Validate(persistToken, 1); err != nil {
		t.Fatalf("Validate() after persistence failure error = %v", err)
	}
}

type nodeHandlerStores struct {
	db           *sql.DB
	pki          *pki.Store
	registration *registration.Store
	ca           *pki.CA
}

func newNodeHandlerTestHarness(t *testing.T) (*NodeHandler, nodeHandlerStores) {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	for _, statement := range []string{
		`PRAGMA foreign_keys = ON`,
		`CREATE TABLE servers (id INTEGER PRIMARY KEY)`,
		`INSERT INTO servers (id) VALUES (1)`,
		`CREATE TABLE ca_certificates (id INTEGER PRIMARY KEY AUTOINCREMENT, fingerprint TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')))`,
		`CREATE TABLE node_certificates (id INTEGER PRIMARY KEY AUTOINCREMENT, server_id INTEGER NOT NULL REFERENCES servers(id), fingerprint TEXT UNIQUE NOT NULL, serial_number TEXT UNIQUE NOT NULL, certificate BLOB NOT NULL, issued_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), expires_at TEXT NOT NULL, revoked_at TEXT)`,
		`CREATE TABLE registration_tokens (id INTEGER PRIMARY KEY AUTOINCREMENT, server_id INTEGER NOT NULL REFERENCES servers(id), token_hash TEXT UNIQUE NOT NULL, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), expires_at TEXT NOT NULL, consumed_at TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec %q: %v", statement, err)
		}
	}
	agePath := filepath.Join(t.TempDir(), "age.key")
	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey() error = %v", err)
	}
	ca, err := pki.LoadOrCreateCA(db, agePath, filepath.Join(t.TempDir(), "ca.key"))
	if err != nil {
		t.Fatalf("LoadOrCreateCA() error = %v", err)
	}
	pkiStore := pki.NewStore(db)
	regStore := registration.NewStore(db)
	h := &NodeHandler{db: db, pkiStore: pkiStore, registrationStore: regStore, ca: ca, logger: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))}
	return h, nodeHandlerStores{db: db, pki: pkiStore, registration: regStore, ca: ca}
}

func registerRequestBody(t *testing.T, serverID int64, token string) []byte {
	t.Helper()
	body, err := json.Marshal(RegisterRequest{Token: token, CSR: string(csrPEM(t, serverID))})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	return body
}

func csrPEM(t *testing.T, serverID int64) []byte {
	t.Helper()
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{Subject: pkix.Name{CommonName: serverCommonName(serverID)}}, newEd25519Signer(t))
	if err != nil {
		t.Fatalf("CreateCertificateRequest() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csr})
}

func newCSR(t *testing.T, serverID int64) *x509.CertificateRequest {
	t.Helper()
	csr, err := x509.ParseCertificateRequest(pemBlockBytes(t, csrPEM(t, serverID)))
	if err != nil {
		t.Fatalf("ParseCertificateRequest() error = %v", err)
	}
	return csr
}

func pemBlockBytes(t *testing.T, pemData []byte) []byte {
	t.Helper()
	block, _ := pem.Decode(pemData)
	if block == nil {
		t.Fatal("expected PEM block")
	}
	return block.Bytes
}

func newEd25519Signer(t *testing.T) ed25519.PrivateKey {
	t.Helper()
	_, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	return key
}

func serverCommonName(serverID int64) string {
	return "server-" + big.NewInt(serverID).String()
}

type failingCA struct{ err error }

func (f failingCA) SignCSR(_ *x509.CertificateRequest, _ int) (*x509.Certificate, error) {
	return nil, f.err
}
func (f failingCA) Certificate() *x509.Certificate { return &x509.Certificate{} }

type failingPKIStore struct{ *pki.Store }

func (f failingPKIStore) SaveNodeCertificateTx(_ context.Context, _ *sql.Tx, _ int64, _ *x509.Certificate) error {
	return errors.New("write failed")
}
