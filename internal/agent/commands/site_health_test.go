package commands

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/shared/ws"
)

func TestSiteHealthSnapshot_ValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		payload  json.RawMessage
		wantCode string
	}{
		{
			name:     "nil payload",
			payload:  nil,
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
		{
			name:     "invalid JSON",
			payload:  json.RawMessage(`{broken}`),
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
		{
			name:     "missing site_id",
			payload:  json.RawMessage(`{"hostname":"example.com","site_path":"/srv/www"}`),
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
		{
			name:     "missing hostname",
			payload:  json.RawMessage(`{"site_id":"s1","site_path":"/srv/www"}`),
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
		{
			name:     "missing site_path",
			payload:  json.RawMessage(`{"site_id":"s1","hostname":"example.com"}`),
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
		{
			name:     "all fields empty strings",
			payload:  json.RawMessage(`{"site_id":"","hostname":"","site_path":""}`),
			wantCode: agentcommand.ErrorCodeInvalidPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SiteHealthSnapshot(context.Background(), ws.Command{ID: "cmd-sh-val", Payload: tt.payload})
			if result.Success {
				t.Fatal("expected failure")
			}
			if result.ErrorCode != tt.wantCode {
				t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, tt.wantCode)
			}
		})
	}
}

func TestSiteHealthSnapshot_AllChecksPass(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	// Mock all external commands to succeed with valid output
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		switch name {
		case "wp":
			// For option get, return matching URL
			for i, arg := range args {
				if arg == "get" && i+1 < len(args) {
					option := args[i+1]
					if option == "home" || option == "siteurl" {
						return exec.Command("echo", "https://example.com")
					}
				}
			}
			// For core is-installed, just succeed
			return exec.Command("true")
		case "curl":
			return exec.Command("echo", "<html><head><title>Test</title></head></html>")
		case "systemctl":
			// Handle is-active, show, etc.
			for _, arg := range args {
				if arg == "is-active" {
					return exec.Command("echo", "active")
				}
				if arg == "--property=LoadState" {
					return exec.Command("echo", "loaded")
				}
				if arg == "--property=Description" {
					return exec.Command("echo", "Test Service")
				}
			}
			return exec.Command("true")
		case "journalctl":
			return exec.Command("true")
		default:
			return exec.Command("true")
		}
	}

	payload, _ := json.Marshal(agentcommand.SiteHealthSnapshotParams{
		SiteID:   "site-1",
		Hostname: "example.com",
		SitePath: "/srv/www/site",
	})

	result := SiteHealthSnapshot(context.Background(), ws.Command{ID: "cmd-sh-pass", Payload: payload})
	if !result.Success {
		t.Fatalf("expected success, got error: %s (code: %s)", result.Error, result.ErrorCode)
	}

	var snapshot agentcommand.SiteHealthSnapshot
	if err := json.Unmarshal(result.Payload, &snapshot); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if snapshot.SiteID != "site-1" {
		t.Errorf("SiteID = %q, want site-1", snapshot.SiteID)
	}
	if snapshot.Hostname != "example.com" {
		t.Errorf("Hostname = %q, want example.com", snapshot.Hostname)
	}
	if !snapshot.Healthy {
		t.Errorf("Healthy = false, want true")
		for _, check := range snapshot.Checks {
			if !check.OK {
				t.Logf("  failed check: %s - %s", check.Name, check.Detail)
			}
		}
	}
	if snapshot.GeneratedAt == "" {
		t.Error("GeneratedAt is empty")
	}
	if snapshot.Summary != "WordPress runtime checks passed on the managed server." {
		t.Errorf("Summary = %q, want healthy summary", snapshot.Summary)
	}
}

func TestSiteHealthSnapshot_UnhealthyService(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		switch name {
		case "wp":
			for i, arg := range args {
				if arg == "get" && i+1 < len(args) {
					option := args[i+1]
					if option == "home" || option == "siteurl" {
						return exec.Command("echo", "https://example.com")
					}
				}
			}
			return exec.Command("true")
		case "curl":
			return exec.Command("echo", "<html>OK</html>")
		case "systemctl":
			for i, arg := range args {
				if arg == "is-active" {
					// Make nginx inactive
					if i+1 < len(args) && args[i+1] == "nginx" {
						return exec.Command("echo", "inactive")
					}
					return exec.Command("echo", "active")
				}
				if arg == "--property=LoadState" {
					return exec.Command("echo", "loaded")
				}
				if arg == "--property=Description" {
					return exec.Command("echo", "Test Service")
				}
			}
			return exec.Command("true")
		case "journalctl":
			return exec.Command("true")
		default:
			return exec.Command("true")
		}
	}

	payload, _ := json.Marshal(agentcommand.SiteHealthSnapshotParams{
		SiteID:   "site-2",
		Hostname: "example.com",
		SitePath: "/srv/www/site",
	})

	result := SiteHealthSnapshot(context.Background(), ws.Command{ID: "cmd-sh-unhealthy", Payload: payload})
	if !result.Success {
		t.Fatalf("expected success result (unhealthy is still success), got error: %s", result.Error)
	}

	var snapshot agentcommand.SiteHealthSnapshot
	if err := json.Unmarshal(result.Payload, &snapshot); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}

	if snapshot.Healthy {
		t.Error("Healthy = true, want false (nginx is inactive)")
	}
	if snapshot.Summary != "One or more runtime checks failed on the managed server." {
		t.Errorf("Summary = %q, want unhealthy summary", snapshot.Summary)
	}
}

func TestSiteHealthSnapshot_PreservesCommandID(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		switch name {
		case "wp":
			for i, arg := range args {
				if arg == "get" && i+1 < len(args) {
					return exec.Command("echo", "https://example.com")
				}
			}
			return exec.Command("true")
		case "curl":
			return exec.Command("echo", "<html>OK</html>")
		case "systemctl":
			for _, arg := range args {
				if arg == "is-active" {
					return exec.Command("echo", "active")
				}
				if arg == "--property=LoadState" {
					return exec.Command("echo", "loaded")
				}
				if arg == "--property=Description" {
					return exec.Command("echo", "Svc")
				}
			}
			return exec.Command("true")
		case "journalctl":
			return exec.Command("true")
		default:
			return exec.Command("true")
		}
	}

	payload, _ := json.Marshal(agentcommand.SiteHealthSnapshotParams{
		SiteID:   "site-1",
		Hostname: "example.com",
		SitePath: "/srv/www/site",
	})
	result := SiteHealthSnapshot(context.Background(), ws.Command{ID: "unique-cmd-id", Payload: payload})
	if result.CommandID != "unique-cmd-id" {
		t.Errorf("CommandID = %q, want unique-cmd-id", result.CommandID)
	}
}

func TestSiteHealthSnapshot_FourExpectedServices(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	var serviceNames []string
	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		if name == "systemctl" {
			for i, arg := range args {
				if arg == "is-active" && i+1 < len(args) {
					serviceNames = append(serviceNames, args[i+1])
					return exec.Command("echo", "active")
				}
				if arg == "--property=LoadState" {
					return exec.Command("echo", "loaded")
				}
				if arg == "--property=Description" {
					return exec.Command("echo", "Svc")
				}
			}
		}
		if name == "wp" {
			for i, arg := range args {
				if arg == "get" && i+1 < len(args) {
					return exec.Command("echo", "https://example.com")
				}
			}
			return exec.Command("true")
		}
		if name == "curl" {
			return exec.Command("echo", "<html>OK</html>")
		}
		return exec.Command("true")
	}

	payload, _ := json.Marshal(agentcommand.SiteHealthSnapshotParams{
		SiteID:   "s1",
		Hostname: "example.com",
		SitePath: "/srv/www/site",
	})
	SiteHealthSnapshot(context.Background(), ws.Command{ID: "cmd-svc-count", Payload: payload})

	expected := map[string]bool{
		"nginx":        false,
		"php8.3-fpm":   false,
		"mariadb":      false,
		"redis-server": false,
	}
	for _, name := range serviceNames {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	for name, found := range expected {
		if !found {
			t.Errorf("expected service %q to be checked, but it was not", name)
		}
	}
}

func TestCollectJournalErrors_ZeroLines(t *testing.T) {
	entries := collectJournalErrors(context.Background(), "nginx", 0)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestCollectJournalErrors_NegativeLines(t *testing.T) {
	entries := collectJournalErrors(context.Background(), "nginx", -1)
	if len(entries) != 0 {
		t.Fatalf("got %d entries, want 0", len(entries))
	}
}

func TestCollectJournalErrors_Success(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("printf", "line1\nline2\nline3\n")
	}

	entries := collectJournalErrors(context.Background(), "nginx", 3)
	if len(entries) != 3 {
		t.Fatalf("got %d entries, want 3", len(entries))
	}
	if entries[0] != "line1" {
		t.Errorf("entries[0] = %q, want line1", entries[0])
	}
}

func TestCollectJournalErrors_CommandFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	entries := collectJournalErrors(context.Background(), "nginx", 4)
	if entries != nil {
		t.Fatalf("got %v, want nil on command failure", entries)
	}
}

func TestCollectJournalErrors_SkipsBlankLines(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("printf", "line1\n\n  \nline2\n")
	}

	entries := collectJournalErrors(context.Background(), "nginx", 4)
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
}

func TestRunOutput_Success(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "hello")
	}

	got := runOutput(context.Background(), "echo", "hello")
	if got != "hello\n" {
		t.Errorf("runOutput = %q, want %q", got, "hello\n")
	}
}

func TestRunOutput_Failure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	got := runOutput(context.Background(), "false")
	if got != "" {
		t.Errorf("runOutput = %q, want empty string on failure", got)
	}
}

func TestWpOptionMatches_Mismatch(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "https://wrong.com")
	}

	err := wpOptionMatches(context.Background(), "/srv/www/public", "home", "https://example.com")
	if err == nil {
		t.Fatal("expected error for mismatched option value")
	}
	if got := err.Error(); got != `expected home to be "https://example.com", got "https://wrong.com"` {
		t.Errorf("error = %q", got)
	}
}

func TestWpOptionMatches_Match(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "https://example.com")
	}

	err := wpOptionMatches(context.Background(), "/srv/www/public", "home", "https://example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWpOptionMatches_CommandFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := wpOptionMatches(context.Background(), "/srv/www/public", "siteurl", "https://example.com")
	if err == nil {
		t.Fatal("expected error when wp command fails")
	}
}

func TestWpCoreInstalled_Success(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	err := wpCoreInstalled(context.Background(), "/srv/www/public")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWpCoreInstalled_Failure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := wpCoreInstalled(context.Background(), "/srv/www/public")
	if err == nil {
		t.Fatal("expected error when wp core is not installed")
	}
}

func TestProbeLocalSite_FatalMarkers(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	markers := []string{
		"<html>Fatal error: something</html>",
		"<html>Parse Error in code</html>",
		"<html>Uncaught Exception</html>",
		"<html>WordPress database error</html>",
	}

	for _, body := range markers {
		commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.Command("echo", body)
		}

		err := probeLocalSite(context.Background(), "example.com", "/")
		if err == nil {
			t.Errorf("expected error for body containing fatal marker: %s", body)
		}
	}
}

func TestProbeLocalSite_EmptyBody(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		// echo with -n to produce empty output; use printf "" instead
		return exec.Command("printf", "")
	}

	err := probeLocalSite(context.Background(), "example.com", "/")
	if err == nil {
		t.Fatal("expected error for empty body")
	}
}

func TestProbeLocalSite_NonHTMLBody(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "just plain text, no html at all")
	}

	err := probeLocalSite(context.Background(), "example.com", "/")
	if err == nil {
		t.Fatal("expected error for non-HTML body")
	}
}

func TestProbeLocalSite_ValidHTML(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "<!doctype html><html><body>Hello</body></html>")
	}

	err := probeLocalSite(context.Background(), "example.com", "/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProbeLocalSite_WpLoginPage(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "wp-login form content")
	}

	err := probeLocalSite(context.Background(), "example.com", "/wp-login.php")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProbeLocalSite_CurlFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	err := probeLocalSite(context.Background(), "example.com", "/")
	if err == nil {
		t.Fatal("expected error when curl fails")
	}
}
