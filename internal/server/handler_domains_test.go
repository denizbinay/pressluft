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

func TestDomainsCreateAssignListDeleteEndpoints(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	handler := NewHandler(db)
	activityStore := activity.NewStore(db)

	siteID, err := NewSiteStore(db).Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Agency Site", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	createBaseBody, _ := json.Marshal(map[string]any{"hostname": "sandbox.pressluft.test", "kind": "wildcard", "ownership": "platform"})
	createBaseReq := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(createBaseBody))
	createBaseReq.Header.Set("Content-Type", "application/json")
	createBaseRes := httptest.NewRecorder()
	handler.ServeHTTP(createBaseRes, createBaseReq)
	if createBaseRes.Code != http.StatusCreated {
		t.Fatalf("create base status = %d, want %d; body = %s", createBaseRes.Code, http.StatusCreated, createBaseRes.Body.String())
	}
	var createdBase map[string]any
	if err := json.Unmarshal(createBaseRes.Body.Bytes(), &createdBase); err != nil {
		t.Fatalf("decode base response: %v", err)
	}
	baseID, _ := createdBase["id"].(string)

	assignBody, _ := json.Marshal(map[string]any{"hostname": "agency.sandbox.pressluft.test", "parent_domain_id": baseID, "is_primary": true})
	assignReq := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/domains", bytes.NewReader(assignBody))
	assignReq.Header.Set("Content-Type", "application/json")
	assignRes := httptest.NewRecorder()
	handler.ServeHTTP(assignRes, assignReq)
	if assignRes.Code != http.StatusCreated {
		t.Fatalf("assign domain status = %d, want %d; body = %s", assignRes.Code, http.StatusCreated, assignRes.Body.String())
	}
	var created map[string]any
	if err := json.Unmarshal(assignRes.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode assigned domain response: %v", err)
	}
	domainID, _ := created["id"].(string)

	listReq := httptest.NewRequest(http.MethodGet, "/api/sites/"+siteID+"/domains", nil)
	listRes := httptest.NewRecorder()
	handler.ServeHTTP(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("list site domains status = %d, want %d", listRes.Code, http.StatusOK)
	}

	activities, _, err := activityStore.ListForSite(context.Background(), siteID, activity.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list activity: %v", err)
	}
	if len(activities) == 0 {
		t.Fatal("expected domain activity entries")
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/domains/"+domainID, nil)
	deleteRes := httptest.NewRecorder()
	handler.ServeHTTP(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusOK {
		t.Fatalf("delete domain status = %d, want %d; body = %s", deleteRes.Code, http.StatusOK, deleteRes.Body.String())
	}
}

func TestDomainsCreateDefaultsToCustomerDirectInventory(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	handler := NewHandler(db)

	body, _ := json.Marshal(map[string]any{"hostname": "agency.example.test"})
	req := httptest.NewRequest(http.MethodPost, "/api/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d; body = %s", res.Code, http.StatusCreated, res.Body.String())
	}

	var created StoredDomain
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Kind != DomainKindDirect {
		t.Fatalf("kind = %q, want %q", created.Kind, DomainKindDirect)
	}
	if created.Ownership != DomainOwnershipCustomer {
		t.Fatalf("ownership = %q, want %q", created.Ownership, DomainOwnershipCustomer)
	}
}

func TestDomainsCreateForSiteDerivesOwnershipFromWildcardParent(t *testing.T) {
	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, "ready")
	handler := NewHandler(db)

	siteID, err := NewSiteStore(db).Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Agency Site", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	parentID, err := NewDomainStore(db).Create(context.Background(), CreateDomainInput{
		Hostname:  "agency.dev",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create wildcard domain: %v", err)
	}

	body, _ := json.Marshal(map[string]any{"hostname": "preview.agency.dev", "parent_domain_id": parentID, "is_primary": true})
	req := httptest.NewRequest(http.MethodPost, "/api/sites/"+siteID+"/domains", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("assign domain status = %d, want %d; body = %s", res.Code, http.StatusCreated, res.Body.String())
	}

	var created StoredDomain
	if err := json.Unmarshal(res.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode assigned domain response: %v", err)
	}
	if created.Ownership != DomainOwnershipCustomer {
		t.Fatalf("ownership = %q, want %q", created.Ownership, DomainOwnershipCustomer)
	}
}
