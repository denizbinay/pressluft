package agent

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ServerID          int64  `yaml:"server_id"`
	ControlPlane      string `yaml:"control_plane"`
	CertFile          string `yaml:"cert_file"`
	KeyFile           string `yaml:"key_file"`
	CACertFile        string `yaml:"ca_cert_file"`
	DataDir           string `yaml:"data_dir"`
	RegistrationToken string `yaml:"registration_token"`
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

	return &cfg, nil
}

func (c *Config) SaveConfig(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func (c *Config) IsRegistered() bool {
	_, errCert := os.Stat(c.CertFile)
	_, errKey := os.Stat(c.KeyFile)
	return errCert == nil && errKey == nil
}

func (c *Config) ClearRegistrationToken(configPath string) error {
	c.RegistrationToken = ""
	return c.SaveConfig(configPath)
}
