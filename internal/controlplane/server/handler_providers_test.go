package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"pressluft/internal/infra/provider"

	_ "modernc.org/sqlite"
)

var registerProviderTestProviderOnce sync.Once

func registerTestProviderProvider() {
	registerProviderTestProviderOnce.Do(func() {
		provider.Register(&testProviderAdapter{})
	})
}

type testProviderAdapter struct{}

func (t *testProviderAdapter) Info() provider.Info {
	return provider.Info{
		Type:         "test-provider-handler",
		Name:         "Test Provider",
		Abbreviation: "TP",
		Description:  "A test provider for handler tests",
	}
}

func (t *testProviderAdapter) Validate(_ context.Context, token string) (*provider.ValidationResult, error) {
	return &provider.ValidationResult{Valid: true, ReadWrite: true, Message: "ok"}, nil
}

func TestProviderListEmpty(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var providers []provider.StoredProvider
	if err := json.Unmarshal(res.Body.Bytes(), &providers); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(providers) != 0 {
		t.Fatalf("provider count = %d, want 0", len(providers))
	}
}

func TestProviderListReturnsInsertedProviders(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	mustInsertProviderRecord(t, db, "test-provider-handler", "agency-provider", "token-ok")
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var providers []provider.StoredProvider
	if err := json.Unmarshal(res.Body.Bytes(), &providers); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(providers) != 1 {
		t.Fatalf("provider count = %d, want 1", len(providers))
	}
	if providers[0].Name != "agency-provider" {
		t.Fatalf("provider name = %q, want %q", providers[0].Name, "agency-provider")
	}
}

func TestProviderListMethodNotAllowed(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodPut, "/api/providers", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestProviderTypesEndpoint(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers/types", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var types []provider.Info
	if err := json.Unmarshal(res.Body.Bytes(), &types); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("expected at least one provider type")
	}

	// Verify our test provider is in the list
	found := false
	for _, pt := range types {
		if pt.Type == "test-provider-handler" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("test-provider-handler not found in provider types")
	}
}

func TestProviderTypesMethodNotAllowed(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodPost, "/api/providers/types", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusMethodNotAllowed)
	}
}

func TestProviderDeleteNonExistent(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	// Use a valid UUIDv7 format that doesn't exist
	req := httptest.NewRequest(http.MethodDelete, "/api/providers/00000000-0000-7000-8000-000000099999", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusNotFound, res.Body.String())
	}
}

func TestProviderDeleteInvalidID(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodDelete, "/api/providers/not-a-valid-id", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestProviderRouteNotFoundForTrailingPath(t *testing.T) {
	registerTestProviderProvider()

	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	req := httptest.NewRequest(http.MethodGet, "/api/providers/unknown/path", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	// Should get a bad request or not found since "unknown/path" is not a valid ID or sub-route
	if res.Code == http.StatusOK {
		t.Fatalf("expected non-200 status for invalid path, got %d", res.Code)
	}
}
