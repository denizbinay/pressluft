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

	if !strings.Contains(body, "id=\"login-form\"") {
		t.Fatalf("dashboard HTML missing login form")
	}

	if !strings.Contains(body, "Job Timeline") {
		t.Fatalf("dashboard HTML missing job timeline panel")
	}

	if !strings.Contains(body, "id=\"job-timeline\"") {
		t.Fatalf("dashboard HTML missing job timeline target")
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
