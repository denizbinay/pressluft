package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const CertificateReissueWindow = 14 * 24 * time.Hour

type Config struct {
	ServerID              int64  `yaml:"server_id"`
	ControlPlane          string `yaml:"control_plane"`
	CertFile              string `yaml:"cert_file"`
	KeyFile               string `yaml:"key_file"`
	CACertFile            string `yaml:"ca_cert_file"`
	DataDir               string `yaml:"data_dir"`
	RegistrationToken     string `yaml:"registration_token,omitempty"`
	RegistrationTokenFile string `yaml:"registration_token_file,omitempty"`
	DevWSToken            string `yaml:"dev_ws_token,omitempty"`
	DevWSTokenFile        string `yaml:"dev_ws_token_file,omitempty"`
	path                  string `yaml:"-"`
}

type CertificateStatus int

const (
	CertificateMissing CertificateStatus = iota
	CertificateValid
	CertificateExpiringSoon
	CertificateExpired
	CertificateInvalid
)

type ClientCertificateState struct {
	Status     CertificateStatus
	Leaf       *x509.Certificate
	Underlying error
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	cfg.path = path
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) SaveConfig(path string) error {
	path, err := c.resolveConfigPath(path)
	if err != nil {
		return err
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (c *Config) IsRegistered() bool {
	_, errCert := os.Stat(c.CertFile)
	_, errKey := os.Stat(c.KeyFile)
	return errCert == nil && errKey == nil
}

func (c *Config) ClearRegistrationToken(configPath string) error {
	c.RegistrationToken = ""
	if strings.TrimSpace(c.RegistrationTokenFile) != "" {
		_ = os.Remove(c.RegistrationTokenFile)
	}
	return c.SaveConfig(configPath)
}

func (c *Config) ResolveRegistrationToken() string {
	if token := strings.TrimSpace(c.RegistrationToken); token != "" {
		return token
	}
	if path := strings.TrimSpace(c.RegistrationTokenFile); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

func (c *Config) ResolveDevWSToken() string {
	if token := strings.TrimSpace(c.DevWSToken); token != "" {
		return token
	}
	if path := strings.TrimSpace(c.DevWSTokenFile); path != "" {
		if data, err := os.ReadFile(path); err == nil {
			return strings.TrimSpace(string(data))
		}
	}
	return ""
}

func (c *Config) CertificateState(now time.Time) ClientCertificateState {
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return ClientCertificateState{Status: CertificateMissing, Underlying: err}
		}
		return ClientCertificateState{Status: CertificateInvalid, Underlying: err}
	}

	if len(cert.Certificate) == 0 {
		return ClientCertificateState{Status: CertificateInvalid, Underlying: fmt.Errorf("certificate chain is empty")}
	}

	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return ClientCertificateState{Status: CertificateInvalid, Underlying: err}
	}

	switch {
	case now.After(leaf.NotAfter):
		return ClientCertificateState{Status: CertificateExpired, Leaf: leaf}
	case leaf.NotAfter.Sub(now) <= CertificateReissueWindow:
		return ClientCertificateState{Status: CertificateExpiringSoon, Leaf: leaf}
	default:
		return ClientCertificateState{Status: CertificateValid, Leaf: leaf}
	}
}

func (c *Config) ConfigPath() string {
	return c.path
}

func (c *Config) Validate() error {
	if c.ServerID <= 0 {
		return fmt.Errorf("server_id is required")
	}
	if strings.TrimSpace(c.ControlPlane) == "" {
		return fmt.Errorf("control_plane is required")
	}
	if strings.TrimSpace(c.CertFile) == "" {
		return fmt.Errorf("cert_file is required")
	}
	if strings.TrimSpace(c.KeyFile) == "" {
		return fmt.Errorf("key_file is required")
	}
	if strings.TrimSpace(c.CACertFile) == "" {
		return fmt.Errorf("ca_cert_file is required")
	}
	if strings.TrimSpace(c.DataDir) == "" {
		return fmt.Errorf("data_dir is required")
	}
	if strings.TrimSpace(c.RegistrationToken) != "" && strings.TrimSpace(c.RegistrationTokenFile) != "" {
		return fmt.Errorf("registration_token and registration_token_file are mutually exclusive")
	}
	if strings.TrimSpace(c.DevWSToken) != "" && strings.TrimSpace(c.DevWSTokenFile) != "" {
		return fmt.Errorf("dev_ws_token and dev_ws_token_file are mutually exclusive")
	}
	return nil
}

func (c *Config) resolveConfigPath(path string) (string, error) {
	if path == "" {
		path = c.path
	}
	if path == "" {
		return "", fmt.Errorf("agent config path is required")
	}
	resolved, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	c.path = resolved
	return resolved, nil
}
