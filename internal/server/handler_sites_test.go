package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSitesCreateListGetUpdateDeleteEndpoints(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	handler := NewHandler(db)

	body := map[string]any{
		"server_id":      serverID,
		"name":           "Agency Site",
		"primary_domain": "agency.example.test",
		"status":         "draft",
		"wordpress_path": "/srv/www/agency/current",
		"php_version":    "8.3",
	}
	bodyBytes, _ := json.Marshal(body)
	createReq := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(bodyBytes))
	createReq.Header.Set("Content-Type", "application/json")
	createRes := httptest.NewRecorder()
	handler.ServeHTTP(createRes, createReq)
	if createRes.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", createRes.Code, http.StatusCreated, createRes.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(createRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	siteID, _ := created["id"].(string)
	if siteID == "" {
		t.Fatal("expected created site id")
	}

	listReq := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
	listRes := httptest.NewRecorder()
	handler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", listRes.Code, http.StatusOK)
	}
	var sites []map[string]any
	if err := json.Unmarshal(listRes.Body.Bytes(), &sites); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("site count = %d, want 1", len(sites))
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID, nil)
	getRes := httptest.NewRecorder()
	handler.ServeHTTP(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("get status = %d, want %d", getRes.Code, http.StatusOK)
	}

	updatedName := map[string]any{"name": "Agency Site Live", "status": "active"}
	updateBytes, _ := json.Marshal(updatedName)
	updateReq := httptest.NewRequest(http.MethodPatch, "/api/sites/"+siteID, bytes.NewReader(updateBytes))
	updateReq.Header.Set("Content-Type", "application/json")
	updateRes := httptest.NewRecorder()
	handler.ServeHTTP(updateRes, updateReq)
	if updateRes.Code != http.StatusOK {
		t.Fatalf("update status = %d, want %d; body = %s", updateRes.Code, http.StatusOK, updateRes.Body.String())
	}

	serverSitesReq := httptest.NewRequest(http.MethodGet, "/api/servers/"+serverID+"/sites", nil)
	serverSitesRes := httptest.NewRecorder()
	handler.ServeHTTP(serverSitesRes, serverSitesReq)
	if serverSitesRes.Code != http.StatusOK {
		t.Fatalf("server sites status = %d, want %d", serverSitesRes.Code, http.StatusOK)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/sites/"+siteID, nil)
	deleteRes := httptest.NewRecorder()
	handler.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("delete status = %d, want %d; body = %s", deleteRes.Code, http.StatusOK, deleteRes.Body.String())
	}

	remaining, err := NewSiteStore(db).List(context.Background())
	if err != nil {
		t.Fatalf("list stored sites: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("remaining sites = %d, want 0", len(remaining))
	}
}

func TestSitesEndpointsValidationAndNotFound(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	badCreateReq := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader([]byte(`{"name":"Missing Server"}`)))
	badCreateReq.Header.Set("Content-Type", "application/json")
	badCreateRes := httptest.NewRecorder()
	handler.ServeHTTP(badCreateRes, badCreateReq)
	if badCreateRes.Code != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d", badCreateRes.Code, http.StatusBadRequest)
	}

	notFoundReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+testPublicID(999), nil)
	notFoundRes := httptest.NewRecorder()
	handler.ServeHTTP(notFoundRes, notFoundReq)
	if notFoundRes.Code != http.StatusNotFound {
		t.Fatalf("get status = %d, want %d", notFoundRes.Code, http.StatusNotFound)
	}
}
