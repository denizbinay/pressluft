package envconfig

import (
	"path/filepath"
	"testing"

	"pressluft/internal/auth"
	"pressluft/internal/platform"
)

func TestResolveControlPlaneRuntimeDefaults(t *testing.T) {
	t.Setenv("PRESSLUFT_EXECUTION_MODE", "")
	t.Setenv("PRESSLUFT_CONTROL_PLANE_URL", "")
	t.Setenv("PRESSLUFT_DB", "")
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", "")
	t.Setenv("PRESSLUFT_CA_KEY_PATH", "")
	t.Setenv("PRESSLUFT_SESSION_KEY_PATH", "")
	t.Setenv("PRESSLUFT_ANSIBLE_DIR", "")
	t.Setenv("PRESSLUFT_ANSIBLE_BIN", "")
	t.Setenv("PRESSLUFT_SESSION_IDLE_TIMEOUT", "")
	t.Setenv("PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT", "")
	t.Setenv("PRESSLUFT_SESSION_COOKIE_SECURE", "")
	t.Setenv("XDG_DATA_HOME", "/tmp/pressluft-data")

	runtime, err := ResolveControlPlaneRuntime(false, "/repo/root")
	if err != nil {
		t.Fatalf("ResolveControlPlaneRuntime() error = %v", err)
	}

	if runtime.ExecutionMode != platform.ExecutionModeSingleNodeLocal {
		t.Fatalf("execution mode = %q, want %q", runtime.ExecutionMode, platform.ExecutionModeSingleNodeLocal)
	}
	if runtime.DBPath != filepath.Join("/tmp/pressluft-data", "pressluft", "pressluft.db") {
		t.Fatalf("db path = %q", runtime.DBPath)
	}
	if runtime.AnsibleDir != "/repo/root" {
		t.Fatalf("ansible dir = %q, want /repo/root", runtime.AnsibleDir)
	}
	if runtime.AnsibleBinary != filepath.Join("/repo/root", ".venv", "bin", "ansible-playbook") {
		t.Fatalf("ansible binary = %q", runtime.AnsibleBinary)
	}
	if runtime.SessionIdleTimeout != auth.DefaultSessionIdleTimeout {
		t.Fatalf("idle timeout = %v, want %v", runtime.SessionIdleTimeout, auth.DefaultSessionIdleTimeout)
	}
	if runtime.SessionAbsoluteTimeout != auth.DefaultSessionAbsoluteTimeout {
		t.Fatalf("absolute timeout = %v, want %v", runtime.SessionAbsoluteTimeout, auth.DefaultSessionAbsoluteTimeout)
	}
	if runtime.SessionCookieSecure {
		t.Fatal("expected secure cookies to default to false outside production bootstrap")
	}
}

func TestResolveControlPlaneRuntimeProductionSettings(t *testing.T) {
	t.Setenv("PRESSLUFT_EXECUTION_MODE", string(platform.ExecutionModeProductionBootstrap))
	t.Setenv("PRESSLUFT_CONTROL_PLANE_URL", "https://control.example.test")
	t.Setenv("PRESSLUFT_TLS_CERT_FILE", "/certs/control.crt")
	t.Setenv("PRESSLUFT_TLS_KEY_FILE", "/certs/control.key")
	t.Setenv("PRESSLUFT_SESSION_IDLE_TIMEOUT", "15m")
	t.Setenv("PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT", "24h")
	t.Setenv("PRESSLUFT_SESSION_COOKIE_SECURE", "")

	runtime, err := ResolveControlPlaneRuntime(false, "/repo/root")
	if err != nil {
		t.Fatalf("ResolveControlPlaneRuntime() error = %v", err)
	}

	if runtime.ExecutionMode != platform.ExecutionModeProductionBootstrap {
		t.Fatalf("execution mode = %q, want %q", runtime.ExecutionMode, platform.ExecutionModeProductionBootstrap)
	}
	if !runtime.SessionCookieSecure {
		t.Fatal("expected secure cookies in production bootstrap mode")
	}
	if runtime.TLSCertFile != "/certs/control.crt" || runtime.TLSKeyFile != "/certs/control.key" {
		t.Fatalf("tls files = (%q, %q)", runtime.TLSCertFile, runtime.TLSKeyFile)
	}
}

func TestResolveControlPlaneRuntimeRejectsInvalidTimeouts(t *testing.T) {
	t.Setenv("PRESSLUFT_SESSION_IDLE_TIMEOUT", "not-a-duration")

	if _, err := ResolveControlPlaneRuntime(false, "/repo/root"); err == nil {
		t.Fatal("expected invalid timeout to fail")
	}
}

func TestResolveAgentRuntimeModes(t *testing.T) {
	t.Setenv("PRESSLUFT_EXECUTION_MODE", "")
	runtime, err := ResolveAgentRuntime(false, "/etc/pressluft/agent.yaml")
	if err != nil {
		t.Fatalf("ResolveAgentRuntime() error = %v", err)
	}
	if runtime.ExecutionMode != platform.ExecutionModeProductionBootstrap {
		t.Fatalf("execution mode = %q, want %q", runtime.ExecutionMode, platform.ExecutionModeProductionBootstrap)
	}
	if runtime.ConfigPath != "/etc/pressluft/agent.yaml" {
		t.Fatalf("config path = %q", runtime.ConfigPath)
	}

	t.Setenv("PRESSLUFT_EXECUTION_MODE", string(platform.ExecutionModeSingleNodeLocal))
	if _, err := ResolveAgentRuntime(false, "/etc/pressluft/agent.yaml"); err == nil {
		t.Fatal("expected unsupported non-dev agent mode to fail")
	}
}

func TestResolveSecureSessionCookiesOverride(t *testing.T) {
	t.Setenv("PRESSLUFT_SESSION_COOKIE_SECURE", "true")
	if !ResolveSecureSessionCookies(platform.ExecutionModeSingleNodeLocal) {
		t.Fatal("expected explicit secure-cookie override to win")
	}
}
