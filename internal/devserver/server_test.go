package devserver

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDashboardShellServedAtRoot(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	if !strings.Contains(body, "Pressluft Operator Console") {
		t.Fatalf("dashboard HTML missing heading")
	}

	for _, token := range []string{"id=\"subsite-nav\"", "href=\"/sites\"", "href=\"/environments\"", "href=\"/backups\"", "href=\"/jobs\""} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard HTML missing %s", token)
		}
	}

	if !strings.Contains(body, "id=\"login-form\"") {
		t.Fatalf("dashboard HTML missing login form")
	}

	if !strings.Contains(body, "Job Timeline") {
		t.Fatalf("dashboard HTML missing job timeline panel")
	}

	if !strings.Contains(body, "id=\"job-timeline\"") {
		t.Fatalf("dashboard HTML missing job timeline target")
	}

	for _, token := range []string{"id=\"site-form\"", "id=\"environment-form\"", "id=\"sites-body\"", "id=\"environments-body\"", "id=\"backup-form\"", "id=\"backups-body\""} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard HTML missing %s", token)
		}
	}
}

func TestDashboardShellServedAtSubsiteRoutes(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	for _, route := range []string{"/", "/sites", "/environments", "/backups", "/jobs"} {
		t.Run(route, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, route, nil)
			rr := httptest.NewRecorder()
			server.httpServer.Handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusOK {
				t.Fatalf("route %s status = %d, want %d", route, rr.Code, http.StatusOK)
			}
		})
	}
}

func TestDashboardContainsConcernScopedMarkers(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	body := rr.Body.String()
	for _, token := range []string{
		"id=\"subsite-overview\"",
		"id=\"metrics\"",
		"id=\"subsite-sites\"",
		"id=\"site-form\"",
		"id=\"sites-body\"",
		"id=\"subsite-environments\"",
		"id=\"environment-form\"",
		"id=\"environments-body\"",
		"id=\"subsite-backups\"",
		"id=\"backup-site\"",
		"id=\"backup-environment\"",
		"id=\"backups-body\"",
		"id=\"subsite-jobs\"",
		"id=\"jobs-body\"",
		"id=\"job-timeline\"",
	} {
		if !strings.Contains(body, token) {
			t.Fatalf("dashboard HTML missing %s", token)
		}
	}
}

func TestDashboardUnknownRouteReturnsNotFound(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	req := httptest.NewRequest(http.MethodGet, "/invalid", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

func TestRequestLogIncludesDeterministicFields(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	line := logs.String()
	for _, token := range []string{"event=request", "method=GET", "path=/", "status=200", "duration_ms="} {
		if !strings.Contains(line, token) {
			t.Fatalf("log %q missing token %q", line, token)
		}
	}
}
