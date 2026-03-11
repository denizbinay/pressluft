package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"pressluft/internal/auth"
)

func TestHealthEndpoint(t *testing.T) {
	handler := NewHandler(nil)
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if payload["status"] != "healthy" {
		t.Fatalf("status payload = %v, want %q", payload["status"], "healthy")
	}
}

func TestOperatorRoutesRequireCapabilitiesWhenAuthenticatorConfigured(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	handler := NewHandlerWithOptions(db, nil, nil, nil, HandlerOptions{
		Authenticator: staticAuthenticator{actor: auth.Actor{
			ID:            "user-1",
			Type:          auth.ActorTypeOperator,
			Role:          auth.Role("viewer"),
			Authenticated: true,
		}},
	})

	for _, path := range []string{"/api/activity", "/api/sites", "/api/domains"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		res := httptest.NewRecorder()

		handler.ServeHTTP(res, req)

		if res.Code != http.StatusForbidden {
			t.Fatalf("path %s status = %d, want %d", path, res.Code, http.StatusForbidden)
		}
	}
}

func TestOperatorRoutesRequireAuthenticationWhenAuthenticatorConfigured(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	handler := NewHandlerWithOptions(db, nil, nil, nil, HandlerOptions{
		Authenticator: staticAuthenticator{err: auth.ErrUnauthenticated},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/activity", nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusUnauthorized)
	}
}

type staticAuthenticator struct {
	actor auth.Actor
	err   error
}

func (a staticAuthenticator) Authenticate(*http.Request) (auth.Actor, error) {
	if a.err != nil {
		return auth.AnonymousActor(), a.err
	}
	if !a.actor.IsAuthenticated() && !errors.Is(a.err, auth.ErrUnauthenticated) {
		return auth.AnonymousActor(), auth.ErrUnauthenticated
	}
	return a.actor, nil
}
