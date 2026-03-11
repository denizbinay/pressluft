package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"pressluft/internal/activity"
)

func TestSitesCreateListGetUpdateDeleteEndpoints(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	handler := NewHandler(db)
	activityStore := activity.NewStore(db)

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

	activities, _, err := activityStore.List(context.Background(), activity.ListFilter{Category: activity.CategorySite, Limit: 10})
	if err != nil {
		t.Fatalf("list activity after create: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("site activity count after create = %d, want 1", len(activities))
	}
	if activities[0].EventType != activity.EventSiteCreated {
		t.Fatalf("create event type = %q, want %q", activities[0].EventType, activity.EventSiteCreated)
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

	activities, _, err = activityStore.List(context.Background(), activity.ListFilter{Category: activity.CategorySite, Limit: 10})
	if err != nil {
		t.Fatalf("list activity after get: %v", err)
	}
	if len(activities) != 1 {
		t.Fatalf("site activity count after get = %d, want 1", len(activities))
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

	activities, _, err = activityStore.List(context.Background(), activity.ListFilter{Category: activity.CategorySite, Limit: 10})
	if err != nil {
		t.Fatalf("list activity after update: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("site activity count after update = %d, want 2", len(activities))
	}
	if activities[0].EventType != activity.EventSiteUpdated {
		t.Fatalf("latest event type after update = %q, want %q", activities[0].EventType, activity.EventSiteUpdated)
	}
	if activities[0].ResourceID != siteID {
		t.Fatalf("updated event resource_id = %q, want %q", activities[0].ResourceID, siteID)
	}
	if activities[0].ParentResourceID != serverID {
		t.Fatalf("updated event parent_resource_id = %q, want %q", activities[0].ParentResourceID, serverID)
	}
	if activities[0].Title != "Site 'Agency Site Live' updated" {
		t.Fatalf("updated event title = %q", activities[0].Title)
	}
	if activities[0].Message != "Site metadata was updated in the control plane." {
		t.Fatalf("updated event message = %q", activities[0].Message)
	}

	getAgainReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID, nil)
	getAgainRes := httptest.NewRecorder()
	handler.ServeHTTP(getAgainRes, getAgainReq)
	if getAgainRes.Code != http.StatusOK {
		t.Fatalf("second get status = %d, want %d", getAgainRes.Code, http.StatusOK)
	}

	activities, _, err = activityStore.List(context.Background(), activity.ListFilter{Category: activity.CategorySite, Limit: 10})
	if err != nil {
		t.Fatalf("list activity after second get: %v", err)
	}
	if len(activities) != 2 {
		t.Fatalf("site activity count after second get = %d, want 2", len(activities))
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

func TestSitesCreateWithWildcardPrimaryDomainConfig(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	domainStore := NewDomainStore(db)
	baseID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "pressluft.dev",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipPlatform,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create wildcard domain: %v", err)
	}
	handler := NewHandler(db)

	body := map[string]any{
		"server_id": serverID,
		"name":      "Sandbox Site",
		"status":    "draft",
		"primary_domain_config": map[string]any{
			"mode":             "wildcard",
			"label":            "client preview",
			"parent_domain_id": baseID,
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", res.Code, http.StatusCreated, res.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created["primary_domain"] != "client-preview.pressluft.dev" {
		t.Fatalf("primary_domain = %v, want %q", created["primary_domain"], "client-preview.pressluft.dev")
	}
}

func TestSitesCreateWithCustomerWildcardPrimaryDomainConfig(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	domainStore := NewDomainStore(db)
	parentID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "agency.dev",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create wildcard domain: %v", err)
	}
	handler := NewHandler(db)

	body := map[string]any{
		"server_id": serverID,
		"name":      "Customer Wildcard Site",
		"status":    "draft",
		"primary_domain_config": map[string]any{
			"mode":             "wildcard",
			"label":            "staging",
			"parent_domain_id": parentID,
		},
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", res.Code, http.StatusCreated, res.Body.String())
	}

	var created map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created["primary_domain"] != "staging.agency.dev" {
		t.Fatalf("primary_domain = %v, want %q", created["primary_domain"], "staging.agency.dev")
	}
}

func TestSitesCreateReturnsBadRequestForDuplicatePrimaryDomain(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	domainStore := NewDomainStore(db)
	_, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "agency.example.test",
		Kind:      DomainKindDirect,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create inventory domain: %v", err)
	}
	handler := NewHandler(db)

	body := map[string]any{
		"server_id":      serverID,
		"name":           "Agency Site",
		"primary_domain": "agency.example.test",
		"status":         "draft",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("create status = %d, want %d; body = %s", res.Code, http.StatusBadRequest, res.Body.String())
	}
}

func TestSitesUpdateReturnsBadRequestForInvalidOrConflictingPrimaryDomain(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	handler := NewHandler(db)

	createBody := map[string]any{
		"server_id": serverID,
		"name":      "Agency Site",
		"status":    "draft",
	}
	createBytes, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/api/sites", bytes.NewReader(createBytes))
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

	invalidBody := map[string]any{"primary_domain": "not a domain"}
	invalidBytes, _ := json.Marshal(invalidBody)
	invalidReq := httptest.NewRequest(http.MethodPatch, "/api/sites/"+siteID, bytes.NewReader(invalidBytes))
	invalidReq.Header.Set("Content-Type", "application/json")
	invalidRes := httptest.NewRecorder()
	handler.ServeHTTP(invalidRes, invalidReq)
	if invalidRes.Code != http.StatusBadRequest {
		t.Fatalf("invalid update status = %d, want %d; body = %s", invalidRes.Code, http.StatusBadRequest, invalidRes.Body.String())
	}

	conflictSiteID, err := NewSiteStore(db).Create(context.Background(), CreateSiteInput{
		ServerID:      serverID,
		Name:          "Conflicting Site",
		PrimaryDomain: "conflict.example.test",
		Status:        SiteStatusDraft,
	})
	if err != nil {
		t.Fatalf("create conflicting site: %v", err)
	}
	if conflictSiteID == "" {
		t.Fatal("expected conflicting site id")
	}
	conflictBody := map[string]any{"primary_domain": "conflict.example.test"}
	conflictBytes, _ := json.Marshal(conflictBody)
	conflictReq := httptest.NewRequest(http.MethodPatch, "/api/sites/"+siteID, bytes.NewReader(conflictBytes))
	conflictReq.Header.Set("Content-Type", "application/json")
	conflictRes := httptest.NewRecorder()
	handler.ServeHTTP(conflictRes, conflictReq)
	if conflictRes.Code != http.StatusBadRequest {
		t.Fatalf("conflict update status = %d, want %d; body = %s", conflictRes.Code, http.StatusBadRequest, conflictRes.Body.String())
	}
}
