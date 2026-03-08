package agent

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRegisterPersistsLoadableKeypairAndClearsToken(t *testing.T) {
	caCert, caKey := newTestCA(t)
	previousClient := registrationHTTPClient
	registrationHTTPClient = &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			if r.Method != http.MethodPost {
				t.Fatalf("method = %q", r.Method)
			}
			if r.URL.String() != "https://control.example.test/api/nodes/42/register" {
				t.Fatalf("url = %q", r.URL.String())
			}
			var req RegisterRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				t.Fatalf("decode request: %v", err)
			}
			csr, err := ParseCSR(req.CSR)
			if err != nil {
				t.Fatalf("ParseCSR() error = %v", err)
			}
			certPEM := signClientCSR(t, caCert, caKey, csr)
			caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCert.Raw})
			payload, err := json.Marshal(RegisterResponse{Certificate: string(certPEM), CACert: string(caPEM)})
			if err != nil {
				t.Fatalf("marshal response: %v", err)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(string(payload))),
			}, nil
		}),
	}
	t.Cleanup(func() { registrationHTTPClient = previousClient })

	dir := t.TempDir()
	configPath := filepath.Join(dir, "agent.yaml")
	cfg := &Config{
		ServerID:          42,
		ControlPlane:      "https://control.example.test",
		CertFile:          filepath.Join(dir, "agent.crt"),
		KeyFile:           filepath.Join(dir, "agent.key"),
		CACertFile:        filepath.Join(dir, "ca.crt"),
		RegistrationToken: "bootstrap-token",
	}
	if err := cfg.SaveConfig(configPath); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	if err := Register(cfg, configPath); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if _, err := LoadClientCert(cfg); err != nil {
		t.Fatalf("LoadClientCert() error = %v", err)
	}
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile(config) error = %v", err)
	}
	if strings.Contains(string(configBytes), "bootstrap-token") {
		t.Fatalf("registration token still present in config: %s", string(configBytes))
	}
	keyBytes, err := os.ReadFile(cfg.KeyFile)
	if err != nil {
		t.Fatalf("ReadFile(key) error = %v", err)
	}
	block, _ := pem.Decode(keyBytes)
	if block == nil || block.Type != "PRIVATE KEY" {
		t.Fatalf("key block = %#v, want PRIVATE KEY", block)
	}
	if _, err := x509.ParsePKCS8PrivateKey(block.Bytes); err != nil {
		t.Fatalf("ParsePKCS8PrivateKey() error = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	if f == nil {
		return nil, fmt.Errorf("round trip function is required")
	}
	return f(r)
}

func ParseCSR(raw string) (*x509.CertificateRequest, error) {
	block, _ := pem.Decode([]byte(raw))
	return x509.ParseCertificateRequest(block.Bytes)
}

func newTestCA(t *testing.T) (*x509.Certificate, ed25519.PrivateKey) {
	t.Helper()
	pub, key, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey() error = %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign,
		IsCA:                  true,
		BasicConstraintsValid: true,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, pub, key)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("ParseCertificate() error = %v", err)
	}
	return cert, key
}

func signClientCSR(t *testing.T, caCert *x509.Certificate, caKey ed25519.PrivateKey, csr *x509.CertificateRequest) []byte {
	t.Helper()
	der, err := x509.CreateCertificate(rand.Reader, &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      csr.Subject,
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		KeyUsage:     x509.KeyUsageDigitalSignature,
	}, caCert, csr.PublicKey, caKey)
	if err != nil {
		t.Fatalf("CreateCertificate() error = %v", err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
}
