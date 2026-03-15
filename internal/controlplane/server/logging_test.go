package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithRequestLoggingPreservesStatus(t *testing.T) {
	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})

	handler := WithRequestLogging(next, logger)
	req := httptest.NewRequest(http.MethodGet, "/brew", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusTeapot {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusTeapot)
	}

	if logs.Len() == 0 {
		t.Fatal("expected request log output")
	}
}
