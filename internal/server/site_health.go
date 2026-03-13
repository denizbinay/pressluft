package server

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"pressluft/internal/agentcommand"
)

func VerifyPublicSiteRouting(ctx context.Context, siteID, hostname string) error {
	resp, bodyPreview, err := fetchPublicSite(ctx, hostname, "/")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected HTTPS status %d", resp.StatusCode)
	}
	if got := strings.TrimSpace(resp.Header.Get("X-Pressluft-Site-ID")); got != strings.TrimSpace(siteID) {
		return fmt.Errorf("hostname did not route to the expected site")
	}
	if strings.TrimSpace(bodyPreview) == "" {
		return fmt.Errorf("hostname returned an empty response body")
	}
	return nil
}

func VerifyPublicWordPressRuntime(ctx context.Context, hostname string) error {
	if err := verifyPublicWordPressPath(ctx, hostname, "/"); err != nil {
		return err
	}
	if err := verifyPublicWordPressPath(ctx, hostname, "/wp-login.php"); err != nil {
		return err
	}
	return nil
}

func RuntimeHealthFromAgentSnapshot(snapshot *agentcommand.SiteHealthSnapshot) (string, string) {
	if snapshot == nil {
		return SiteRuntimeHealthStateUnknown, "Managed server diagnostics are unavailable."
	}
	if snapshot.Healthy {
		return SiteRuntimeHealthStateHealthy, strings.TrimSpace(snapshot.Summary)
	}
	message := strings.TrimSpace(snapshot.Summary)
	for _, check := range snapshot.Checks {
		if !check.OK {
			if detail := strings.TrimSpace(check.Detail); detail != "" {
				message = fmt.Sprintf("%s: %s", check.Name, detail)
			} else {
				message = fmt.Sprintf("%s failed on the managed server.", check.Name)
			}
			break
		}
	}
	if message == "" {
		message = "Managed server diagnostics reported a runtime issue."
	}
	return SiteRuntimeHealthStateIssue, message
}

func verifyPublicWordPressPath(ctx context.Context, hostname, path string) error {
	resp, bodyPreview, err := fetchPublicSite(ctx, hostname, path)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("unexpected HTTPS status %d for %s", resp.StatusCode, path)
	}
	if err := verifyWordPressBody(path, resp.Header.Get("Content-Type"), bodyPreview); err != nil {
		return err
	}
	return nil
}

func fetchPublicSite(ctx context.Context, hostname, path string) (*http.Response, string, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	if path == "" {
		path = "/"
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+hostname+path, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	buf, err := io.ReadAll(io.LimitReader(resp.Body, 65536))
	if err != nil {
		return nil, "", err
	}
	resp.Body.Close()
	resp.Body = io.NopCloser(strings.NewReader(string(buf)))
	return resp, string(buf), nil
}

func verifyWordPressBody(path, contentType, body string) error {
	trimmed := strings.TrimSpace(body)
	if trimmed == "" {
		return fmt.Errorf("empty response body for %s", path)
	}
	lowerBody := strings.ToLower(trimmed)
	for _, marker := range []string{"fatal error", "parse error", "uncaught", "wordpress database error"} {
		if strings.Contains(lowerBody, marker) {
			return fmt.Errorf("fatal marker %q detected for %s", marker, path)
		}
	}
	lowerContentType := strings.ToLower(strings.TrimSpace(contentType))
	if lowerContentType != "" && !strings.Contains(lowerContentType, "text/html") && !strings.Contains(lowerContentType, "application/xhtml+xml") {
		return fmt.Errorf("unexpected content type %q for %s", contentType, path)
	}
	if !strings.Contains(lowerBody, "<html") && !strings.Contains(lowerBody, "<!doctype html") && !strings.Contains(lowerBody, "wp-login") && !strings.Contains(lowerBody, "wp-content") {
		return fmt.Errorf("response body for %s does not look like WordPress HTML", path)
	}
	return nil
}
