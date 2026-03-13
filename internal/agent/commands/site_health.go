package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/agentcommand"
	"pressluft/internal/ws"
)

func SiteHealthSnapshot(ctx context.Context, cmd ws.Command) ws.CommandResult {
	params, err := agentcommand.DecodeSiteHealthPayload(cmd.Payload)
	if err != nil {
		var validationErr *agentcommand.ValidationError
		if errors.As(err, &validationErr) {
			return ws.FailureResult(cmd.ID, validationErr.Code, validationErr.Message, nil, "")
		}
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeInvalidPayload, "invalid site_health_snapshot payload", nil, "")
	}

	publicPath := filepath.ToSlash(filepath.Join(params.SitePath, "public"))
	services := collectServiceSnapshot(ctx)
	checks := make([]agentcommand.SiteHealthCheck, 0, 8)
	recentErrors := make([]string, 0, 8)
	healthy := true

	markCheck := func(name string, err error) {
		check := agentcommand.SiteHealthCheck{Name: name, OK: err == nil}
		if err != nil {
			check.Detail = err.Error()
			healthy = false
		}
		checks = append(checks, check)
	}

	markCheck("wordpress-installed", wpCoreInstalled(ctx, publicPath))
	markCheck("wordpress-home-url", wpOptionMatches(ctx, publicPath, "home", "https://"+params.Hostname))
	markCheck("wordpress-siteurl", wpOptionMatches(ctx, publicPath, "siteurl", "https://"+params.Hostname))
	markCheck("home-page", probeLocalSite(ctx, params.Hostname, "/"))
	markCheck("login-page", probeLocalSite(ctx, params.Hostname, "/wp-login.php"))

	for _, service := range services {
		if service.ActiveState != "active" {
			healthy = false
			checks = append(checks, agentcommand.SiteHealthCheck{
				Name:   "service-" + service.Name,
				OK:     false,
				Detail: fmt.Sprintf("service state is %s", service.ActiveState),
			})
		}
	}

	recentErrors = append(recentErrors, collectJournalErrors(ctx, "php8.3-fpm", 4)...)
	recentErrors = append(recentErrors, collectJournalErrors(ctx, "nginx", 4)...)
	if len(recentErrors) > 8 {
		recentErrors = recentErrors[:8]
	}

	summary := "WordPress runtime checks passed on the managed server."
	if !healthy {
		summary = "One or more runtime checks failed on the managed server."
	}

	result := agentcommand.SiteHealthSnapshot{
		SiteID:       params.SiteID,
		Hostname:     params.Hostname,
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		Healthy:      healthy,
		Summary:      summary,
		Services:     services,
		Checks:       checks,
		RecentErrors: recentErrors,
	}
	payload, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeSerializationFailed, "failed to encode site health snapshot", nil, "")
	}
	return ws.CommandResult{CommandID: cmd.ID, Success: true, Payload: payload}
}

func collectServiceSnapshot(ctx context.Context) []agentcommand.Service {
	services := make([]agentcommand.Service, 0, 4)
	for _, serviceName := range []string{"nginx", "php8.3-fpm", "mariadb", "redis-server"} {
		activeState := strings.TrimSpace(runOutput(ctx, "systemctl", "is-active", serviceName))
		if activeState == "" {
			activeState = "unknown"
		}
		loadState := strings.TrimSpace(runOutput(ctx, "systemctl", "show", serviceName, "--property=LoadState", "--value"))
		if loadState == "" {
			loadState = "unknown"
		}
		description := strings.TrimSpace(runOutput(ctx, "systemctl", "show", serviceName, "--property=Description", "--value"))
		services = append(services, agentcommand.Service{
			Name:        serviceName,
			Description: description,
			ActiveState: activeState,
			LoadState:   loadState,
		})
	}
	return services
}

func wpCoreInstalled(ctx context.Context, publicPath string) error {
	_, err := commandContext(ctx, "wp", "--path="+publicPath, "--allow-root", "core", "is-installed").CombinedOutput()
	if err != nil {
		return fmt.Errorf("wp core is-installed failed")
	}
	return nil
}

func wpOptionMatches(ctx context.Context, publicPath, optionName, want string) error {
	out, err := commandContext(ctx, "wp", "--path="+publicPath, "--allow-root", "option", "get", optionName).CombinedOutput()
	if err != nil {
		return fmt.Errorf("wp option get %s failed", optionName)
	}
	got := strings.TrimSpace(string(out))
	if got != want {
		return fmt.Errorf("expected %s to be %q, got %q", optionName, want, got)
	}
	return nil
}

func probeLocalSite(ctx context.Context, hostname, requestPath string) error {
	url := fmt.Sprintf("https://%s%s", hostname, requestPath)
	out, err := commandContext(ctx, "curl", "--silent", "--show-error", "--fail", "--insecure", "--noproxy", "*", "--resolve", hostname+":443:127.0.0.1", url).CombinedOutput()
	if err != nil {
		return fmt.Errorf("curl probe failed for %s", requestPath)
	}
	body := strings.TrimSpace(string(out))
	if body == "" {
		return fmt.Errorf("empty response body for %s", requestPath)
	}
	lower := strings.ToLower(body)
	for _, marker := range []string{"fatal error", "parse error", "uncaught", "wordpress database error"} {
		if strings.Contains(lower, marker) {
			return fmt.Errorf("fatal marker %q detected for %s", marker, requestPath)
		}
	}
	if !strings.Contains(lower, "<html") && !strings.Contains(lower, "<!doctype html") && !strings.Contains(lower, "wp-login") {
		return fmt.Errorf("response body for %s does not look like html", requestPath)
	}
	return nil
}

func collectJournalErrors(ctx context.Context, unit string, lines int) []string {
	if lines <= 0 {
		return nil
	}
	out, err := commandContext(ctx, "journalctl", "-u", unit, "-n", fmt.Sprintf("%d", lines), "--no-pager").CombinedOutput()
	if err != nil {
		return nil
	}
	parts := strings.Split(strings.TrimSpace(string(out)), "\n")
	entries := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		entries = append(entries, part)
	}
	return entries
}

func runOutput(ctx context.Context, name string, args ...string) string {
	out, err := commandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return string(out)
}
