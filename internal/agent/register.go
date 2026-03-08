package agent

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type RegisterRequest struct {
	Token string `json:"token"`
	CSR   string `json:"csr"`
}

type RegisterResponse struct {
	Certificate string `json:"certificate"`
	CACert      string `json:"ca_certificate"`
}

var ErrExistingValidCertificate = errors.New("existing valid certificate already present")

var registrationHTTPClient = http.DefaultClient

func Register(config *Config, configPath string) error {
	state := config.CertificateState(time.Now())
	switch state.Status {
	case CertificateValid:
		return ErrExistingValidCertificate
	case CertificateExpiringSoon:
		if strings.TrimSpace(config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("client certificate expires at %s and no registration token is available for reissue", state.Leaf.NotAfter.UTC().Format(time.RFC3339))
		}
	case CertificateExpired, CertificateMissing:
		if strings.TrimSpace(config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("registration token is required when no usable client certificate is present")
		}
	case CertificateInvalid:
		if strings.TrimSpace(config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("client certificate is invalid: %w", state.Underlying)
		}
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("generate key: %w", err)
	}

	_ = publicKey

	template := &x509.CertificateRequest{
		Subject: pkix.Name{
			CommonName: fmt.Sprintf("server-%d", config.ServerID),
		},
	}

	csrDER, err := x509.CreateCertificateRequest(rand.Reader, template, privateKey)
	if err != nil {
		return fmt.Errorf("create CSR: %w", err)
	}

	csrPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	reqBody, err := json.Marshal(RegisterRequest{
		Token: config.ResolveRegistrationToken(),
		CSR:   string(csrPEM),
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	registrationURL, err := config.registrationURL()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, registrationURL, bytes.NewReader(reqBody))
	if err != nil {
		return fmt.Errorf("build registration request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := registrationHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("POST registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("registration failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(message)))
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	pkcs8Key, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: pkcs8Key})

	if err := writeFileAtomically(config.KeyFile, keyPEM, 0600); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	if err := writeFileAtomically(config.CertFile, []byte(regResp.Certificate), 0644); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}

	if err := writeFileAtomically(config.CACertFile, []byte(regResp.CACert), 0644); err != nil {
		return fmt.Errorf("save CA certificate: %w", err)
	}

	if err := config.ClearRegistrationToken(configPath); err != nil {
		return fmt.Errorf("clear token: %w", err)
	}

	return nil
}

func LoadClientCert(config *Config) (tls.Certificate, error) {
	cert, err := tls.LoadX509KeyPair(config.CertFile, config.KeyFile)
	if err != nil {
		return tls.Certificate{}, err
	}
	return cert, nil
}

func LoadCACertPool(config *Config) (*x509.CertPool, error) {
	caCertData, err := os.ReadFile(config.CACertFile)
	if err != nil {
		return nil, err
	}

	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(caCertData) {
		return nil, fmt.Errorf("failed to add CA cert to pool")
	}

	return pool, nil
}

func writeFileAtomically(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(mode); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, path)
}
