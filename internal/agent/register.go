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
	"fmt"
	"net/http"
	"os"
)

type RegisterRequest struct {
	Token string `json:"token"`
	CSR   string `json:"csr"`
}

type RegisterResponse struct {
	Certificate string `json:"certificate"`
	CACert      string `json:"ca_certificate"`
}

func Register(config *Config, configPath string) error {
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
		Token: config.RegistrationToken,
		CSR:   string(csrPEM),
	})
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(
		fmt.Sprintf("%s/api/nodes/%d/register", config.ControlPlane, config.ServerID),
		"application/json",
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return fmt.Errorf("POST registration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status %d", resp.StatusCode)
	}

	var regResp RegisterResponse
	if err := json.NewDecoder(resp.Body).Decode(&regResp); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if err := os.WriteFile(config.KeyFile, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKey}), 0600); err != nil {
		return fmt.Errorf("save private key: %w", err)
	}

	if err := os.WriteFile(config.CertFile, []byte(regResp.Certificate), 0644); err != nil {
		return fmt.Errorf("save certificate: %w", err)
	}

	if err := os.WriteFile(config.CACertFile, []byte(regResp.CACert), 0644); err != nil {
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
