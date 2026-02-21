package devserver

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPlaceholderMessage(t *testing.T) {
	var logs bytes.Buffer
	logger := log.New(&logs, "", 0)
	server := New(":0", logger)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	server.httpServer.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}

	if got := rr.Body.String(); got != placeholderMessage+"\n" {
		t.Fatalf("body = %q, want %q", got, placeholderMessage+"\n")
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
