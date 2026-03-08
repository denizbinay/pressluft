package envconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/auth"
	"pressluft/internal/platform"
)

type EnvVarSpec struct {
	Name         string `json:"name"`
	Scope        string `json:"scope"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
	Description  string `json:"description"`
}

type ControlPlaneRuntime struct {
	ExecutionMode          platform.ExecutionMode `json:"execution_mode"`
	ControlPlaneURL        string                 `json:"control_plane_url,omitempty"`
	DBPath                 string                 `json:"db_path"`
	DataDir                string                 `json:"data_dir"`
	AgeKeyPath             string                 `json:"age_key_path"`
	CAKeyPath              string                 `json:"ca_key_path"`
	SessionSecretPath      string                 `json:"session_secret_path"`
	AnsibleDir             string                 `json:"ansible_dir"`
	AnsibleBinary          string                 `json:"ansible_binary"`
	TLSCertFile            string                 `json:"tls_cert_file,omitempty"`
	TLSKeyFile             string                 `json:"tls_key_file,omitempty"`
	SessionIdleTimeout     time.Duration          `json:"session_idle_timeout"`
	SessionAbsoluteTimeout time.Duration          `json:"session_absolute_timeout"`
	SessionCookieSecure    bool                   `json:"session_cookie_secure"`
}

type AgentRuntime struct {
	ExecutionMode platform.ExecutionMode `json:"execution_mode"`
	ConfigPath    string                 `json:"config_path"`
}

func DefaultDataDir() string {
	dataDir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return dataDir
}

func ResolveControlPlaneRuntime(isDevBuild bool, cwd string) (ControlPlaneRuntime, error) {
	executionMode, err := platform.NormalizeControlPlaneExecutionMode(os.Getenv("PRESSLUFT_EXECUTION_MODE"), isDevBuild)
	if err != nil {
		return ControlPlaneRuntime{}, err
	}

	dataDir := DefaultDataDir()
	ansibleDir := strings.TrimSpace(os.Getenv("PRESSLUFT_ANSIBLE_DIR"))
	if ansibleDir == "" {
		if cwd != "" {
			ansibleDir = cwd
		} else {
			ansibleDir = "."
		}
	}

	ansibleBinary := strings.TrimSpace(os.Getenv("PRESSLUFT_ANSIBLE_BIN"))
	if ansibleBinary == "" {
		ansibleBinary = filepath.Join(ansibleDir, ".venv", "bin", "ansible-playbook")
	}

	idleTimeout, absoluteTimeout, err := ResolveSessionTimeouts()
	if err != nil {
		return ControlPlaneRuntime{}, err
	}

	runtime := ControlPlaneRuntime{
		ExecutionMode:          executionMode,
		ControlPlaneURL:        strings.TrimSpace(os.Getenv("PRESSLUFT_CONTROL_PLANE_URL")),
		DBPath:                 resolveDBPath(dataDir),
		DataDir:                dataDir,
		AgeKeyPath:             resolveAgeKeyPath(dataDir),
		CAKeyPath:              resolveCAKeyPath(dataDir),
		SessionSecretPath:      resolveSessionSecretPath(dataDir),
		AnsibleDir:             ansibleDir,
		AnsibleBinary:          ansibleBinary,
		TLSCertFile:            strings.TrimSpace(os.Getenv("PRESSLUFT_TLS_CERT_FILE")),
		TLSKeyFile:             strings.TrimSpace(os.Getenv("PRESSLUFT_TLS_KEY_FILE")),
		SessionIdleTimeout:     idleTimeout,
		SessionAbsoluteTimeout: absoluteTimeout,
		SessionCookieSecure:    ResolveSecureSessionCookies(executionMode),
	}

	return runtime, nil
}

func ResolveAgentRuntime(isDevBuild bool, configPath string) (AgentRuntime, error) {
	executionMode, err := platform.NormalizeAgentExecutionMode(os.Getenv("PRESSLUFT_EXECUTION_MODE"), isDevBuild)
	if err != nil {
		return AgentRuntime{}, err
	}
	return AgentRuntime{
		ExecutionMode: executionMode,
		ConfigPath:    strings.TrimSpace(configPath),
	}, nil
}

func ResolveSessionTimeouts() (time.Duration, time.Duration, error) {
	idle := auth.DefaultSessionIdleTimeout
	absolute := auth.DefaultSessionAbsoluteTimeout
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_IDLE_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("parse PRESSLUFT_SESSION_IDLE_TIMEOUT: %w", err)
		}
		idle = parsed
	}
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("parse PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT: %w", err)
		}
		absolute = parsed
	}
	return idle, absolute, nil
}

func ResolveSecureSessionCookies(mode platform.ExecutionMode) bool {
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_COOKIE_SECURE")); raw != "" {
		return raw == "1" || strings.EqualFold(raw, "true")
	}
	return mode == platform.ExecutionModeProductionBootstrap
}

func ControlPlaneEnvSpec() []EnvVarSpec {
	return []EnvVarSpec{
		{Name: "PRESSLUFT_EXECUTION_MODE", Scope: "control-plane", DefaultValue: string(platform.ExecutionModeSingleNodeLocal), Description: "Execution mode contract for the control plane."},
		{Name: "PRESSLUFT_CONTROL_PLANE_URL", Scope: "control-plane", Description: "Public base URL used for agent bootstrap and reconnect."},
		{Name: "PRESSLUFT_TLS_CERT_FILE", Scope: "control-plane", Required: true, Description: "HTTPS certificate file in production-bootstrap mode."},
		{Name: "PRESSLUFT_TLS_KEY_FILE", Scope: "control-plane", Required: true, Description: "HTTPS private key file in production-bootstrap mode."},
		{Name: "PRESSLUFT_DB", Scope: "control-plane", Description: "SQLite database path."},
		{Name: "PORT", Scope: "control-plane", DefaultValue: "8080", Description: "HTTP listen port."},
		{Name: "XDG_DATA_HOME", Scope: "shared", Description: "Base data directory for derived paths."},
		{Name: "PRESSLUFT_CA_KEY_PATH", Scope: "control-plane", Description: "Encrypted CA private key path."},
		{Name: "PRESSLUFT_AGE_KEY_PATH", Scope: "shared", Description: "age identity path used for local secret encryption."},
		{Name: "PRESSLUFT_SESSION_KEY_PATH", Scope: "control-plane", Description: "Session HMAC secret path."},
		{Name: "PRESSLUFT_SESSION_IDLE_TIMEOUT", Scope: "control-plane", DefaultValue: auth.DefaultSessionIdleTimeout.String(), Description: "Operator session idle timeout."},
		{Name: "PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT", Scope: "control-plane", DefaultValue: auth.DefaultSessionAbsoluteTimeout.String(), Description: "Operator session absolute timeout."},
		{Name: "PRESSLUFT_SESSION_COOKIE_SECURE", Scope: "control-plane", Description: "Override secure-cookie behavior."},
		{Name: "PRESSLUFT_BOOTSTRAP_ADMIN_EMAIL", Scope: "control-plane", Description: "Bootstrap admin email for non-dev mode."},
		{Name: "PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD", Scope: "control-plane", Description: "Bootstrap admin password for non-dev mode."},
		{Name: "PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD_FILE", Scope: "control-plane", Description: "File-based bootstrap admin password source."},
		{Name: "PRESSLUFT_ANSIBLE_DIR", Scope: "control-plane", Description: "Working directory used to resolve ansible paths."},
		{Name: "PRESSLUFT_ANSIBLE_BIN", Scope: "control-plane", Description: "Path to ansible-playbook."},
	}
}

func AgentEnvSpec() []EnvVarSpec {
	return []EnvVarSpec{
		{Name: "PRESSLUFT_EXECUTION_MODE", Scope: "agent", DefaultValue: string(platform.ExecutionModeProductionBootstrap), Description: "Execution mode contract for the agent process."},
	}
}

func resolveDBPath(dataDir string) string {
	if p := strings.TrimSpace(os.Getenv("PRESSLUFT_DB")); p != "" {
		return p
	}
	return filepath.Join(dataDir, "pressluft", "pressluft.db")
}

func resolveAgeKeyPath(dataDir string) string {
	if p := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH")); p != "" {
		return p
	}
	return filepath.Join(dataDir, "pressluft", "age.key")
}

func resolveCAKeyPath(dataDir string) string {
	if p := strings.TrimSpace(os.Getenv("PRESSLUFT_CA_KEY_PATH")); p != "" {
		return p
	}
	return filepath.Join(dataDir, "pressluft", "ca.key")
}

func resolveSessionSecretPath(dataDir string) string {
	if p := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_KEY_PATH")); p != "" {
		return p
	}
	return filepath.Join(dataDir, "pressluft", "session.key")
}
