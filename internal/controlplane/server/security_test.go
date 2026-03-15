package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestRateLimitKeyUsesNormalizedRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/auth/login", nil)
	req.RemoteAddr = "192.0.2.15:54321"
	req.Header.Set("X-Forwarded-For", "198.51.100.20")

	if got, want := requestRateLimitKey(req), "192.0.2.15"; got != want {
		t.Fatalf("requestRateLimitKey() = %q, want %q", got, want)
	}
}

func TestWithRateLimitIgnoresPortAndSpoofedForwardedIP(t *testing.T) {
	limiter := newRateLimiter(1, time.Minute)
	handler := withRateLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}), limiter, "auth-login")

	first := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	first.RemoteAddr = "192.0.2.15:1001"
	first.Header.Set("X-Forwarded-For", "198.51.100.20")
	firstRec := httptest.NewRecorder()
	handler.ServeHTTP(firstRec, first)
	if firstRec.Code != http.StatusNoContent {
		t.Fatalf("first status = %d, want %d", firstRec.Code, http.StatusNoContent)
	}

	second := httptest.NewRequest(http.MethodPost, "/api/auth/login", nil)
	second.RemoteAddr = "192.0.2.15:2002"
	second.Header.Set("X-Forwarded-For", "203.0.113.44")
	secondRec := httptest.NewRecorder()
	handler.ServeHTTP(secondRec, second)
	if secondRec.Code != http.StatusTooManyRequests {
		t.Fatalf("second status = %d, want %d", secondRec.Code, http.StatusTooManyRequests)
	}
}
