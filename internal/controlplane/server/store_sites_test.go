package server

import (
	"context"
	"strings"
	"testing"
)

func TestSiteStoreCreateListAndGet(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	siteID, err := store.Create(context.Background(), CreateSiteInput{
		ServerID:            serverID,
		Name:                "Agency Brochure",
		WordPressAdminEmail: "owner@example.test",
		PrimaryDomain:       "brochure.example.test",
		Status:              SiteStatusDraft,
		WordPressPath:       "/srv/www/brochure/current",
		PHPVersion:          "8.3",
		WordPressVersion:    "6.8",
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	site, err := store.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if site.ServerID != serverID {
		t.Fatalf("server_id = %q, want %q", site.ServerID, serverID)
	}
	if site.ServerName != "server-under-test" {
		t.Fatalf("server_name = %q, want %q", site.ServerName, "server-under-test")
	}

	sites, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list sites: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("site count = %d, want 1", len(sites))
	}
	if sites[0].Name != "Agency Brochure" {
		t.Fatalf("name = %q, want %q", sites[0].Name, "Agency Brochure")
	}
}

func TestSiteStoreCreateWithUserBaseDomainPrimaryHostname(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	baseID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname: "agency.dev",
		Kind:     DomainKindBaseDomain,
		Source:   DomainSourceUser,
		DNSState: DomainDNSStateReady,
	})
	if err != nil {
		t.Fatalf("create base domain: %v", err)
	}

	siteID, err := store.Create(context.Background(), CreateSiteInput{
		ServerID:            serverID,
		Name:                "Agency Brochure",
		WordPressAdminEmail: "owner@example.test",
		PrimaryHostnameConfig: &CreateSitePrimaryHostnameInput{
			Source:   DomainSourceUser,
			Label:    "Northwind Live",
			DomainID: baseID,
		},
		Status: SiteStatusDraft,
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	site, err := store.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if site.PrimaryDomain != "northwind-live.agency.dev" {
		t.Fatalf("primary_domain = %q, want %q", site.PrimaryDomain, "northwind-live.agency.dev")
	}
	domains, err := domainStore.ListBySite(context.Background(), siteID)
	if err != nil {
		t.Fatalf("list site domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("site domains = %d, want 1", len(domains))
	}
	if domains[0].ParentDomainID != baseID {
		t.Fatalf("parent_domain_id = %q, want %q", domains[0].ParentDomainID, baseID)
	}
	if domains[0].Kind != DomainKindHostname {
		t.Fatalf("kind = %q, want %q", domains[0].Kind, DomainKindHostname)
	}
	if domains[0].Source != DomainSourceUser {
		t.Fatalf("source = %q, want %q", domains[0].Source, DomainSourceUser)
	}
	if domains[0].DNSState != DomainDNSStateReady {
		t.Fatalf("dns_state = %q, want %q", domains[0].DNSState, DomainDNSStateReady)
	}
}

func TestSiteStoreCreateWithPendingBaseDomainKeepsChildPending(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	baseID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname: "agency.dev",
		Kind:     DomainKindBaseDomain,
		Source:   DomainSourceUser,
		DNSState: DomainDNSStatePending,
	})
	if err != nil {
		t.Fatalf("create base domain: %v", err)
	}

	siteID, err := store.Create(context.Background(), CreateSiteInput{
		ServerID:            serverID,
		Name:                "Agency Brochure",
		WordPressAdminEmail: "owner@example.test",
		PrimaryHostnameConfig: &CreateSitePrimaryHostnameInput{
			Source:   DomainSourceUser,
			Label:    "preview-42",
			DomainID: baseID,
		},
		Status: SiteStatusDraft,
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	domains, err := domainStore.ListBySite(context.Background(), siteID)
	if err != nil {
		t.Fatalf("list site domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("site domains = %d, want 1", len(domains))
	}
	if domains[0].Hostname != "preview-42.agency.dev" {
		t.Fatalf("hostname = %q, want %q", domains[0].Hostname, "preview-42.agency.dev")
	}
	if domains[0].DNSState != DomainDNSStatePending {
		t.Fatalf("dns_state = %q, want %q", domains[0].DNSState, DomainDNSStatePending)
	}
}

func TestSiteStoreCreateWithFallbackResolverPrimaryHostname(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	siteID, err := store.Create(context.Background(), CreateSiteInput{
		ServerID:            serverID,
		Name:                "Agency Brochure",
		WordPressAdminEmail: "owner@example.test",
		PrimaryHostnameConfig: &CreateSitePrimaryHostnameInput{
			Source: DomainSourceFallbackResolver,
			Label:  "Northwind Live",
		},
		Status: SiteStatusDraft,
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	site, err := store.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if site.PrimaryDomain != "northwind-live.203-0-113-10.sslip.io" {
		t.Fatalf("primary_domain = %q, want %q", site.PrimaryDomain, "northwind-live.203-0-113-10.sslip.io")
	}
	domains, err := domainStore.ListBySite(context.Background(), siteID)
	if err != nil {
		t.Fatalf("list site domains: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("site domains = %d, want 1", len(domains))
	}
	if domains[0].Source != DomainSourceFallbackResolver {
		t.Fatalf("source = %q, want %q", domains[0].Source, DomainSourceFallbackResolver)
	}
	if domains[0].DNSState != DomainDNSStateReady {
		t.Fatalf("dns_state = %q, want %q", domains[0].DNSState, DomainDNSStateReady)
	}
}

func TestSiteStoreListByServer(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	otherServerID := mustInsertServerWithStatus(t, db, "ready")

	_, _ = store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "One", WordPressAdminEmail: "one@example.test", Status: SiteStatusDraft})
	_, _ = store.Create(context.Background(), CreateSiteInput{ServerID: otherServerID, Name: "Two", WordPressAdminEmail: "two@example.test", Status: SiteStatusActive})

	sites, err := store.ListByServer(context.Background(), serverID)
	if err != nil {
		t.Fatalf("list by server: %v", err)
	}
	if len(sites) != 1 {
		t.Fatalf("site count = %d, want 1", len(sites))
	}
	if sites[0].Name != "One" {
		t.Fatalf("name = %q, want %q", sites[0].Name, "One")
	}
}

func TestSiteStoreUpdate(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Original", WordPressAdminEmail: "owner@example.test", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	updatedName := "Client Store"
	updatedStatus := SiteStatusActive
	updatedDomain := "store.example.test"
	updatedPath := "/srv/www/store/current"
	updatedPHP := "8.2"
	updatedWP := "6.7"
	updatedAdminEmail := "site-owner@example.test"

	site, err := store.Update(context.Background(), siteID, UpdateSiteInput{
		Name:                &updatedName,
		WordPressAdminEmail: &updatedAdminEmail,
		Status:              &updatedStatus,
		PrimaryDomain:       &updatedDomain,
		WordPressPath:       &updatedPath,
		PHPVersion:          &updatedPHP,
		WordPressVersion:    &updatedWP,
	})
	if err != nil {
		t.Fatalf("update site: %v", err)
	}
	if site.Status != SiteStatusActive {
		t.Fatalf("status = %q, want %q", site.Status, SiteStatusActive)
	}
	if site.PrimaryDomain != updatedDomain {
		t.Fatalf("primary_domain = %q, want %q", site.PrimaryDomain, updatedDomain)
	}
	if site.WordPressAdminEmail != updatedAdminEmail {
		t.Fatalf("wordpress_admin_email = %q, want %q", site.WordPressAdminEmail, updatedAdminEmail)
	}
}

func TestSiteStoreUpdateRollsBackWhenPrimaryHostnameAssignmentFails(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID, err := store.Create(context.Background(), CreateSiteInput{
		ServerID:            serverID,
		Name:                "Original",
		WordPressAdminEmail: "owner@example.test",
		PrimaryDomain:       "original.example.test",
		Status:              SiteStatusDraft,
	})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	_, err = domainStore.Create(context.Background(), CreateDomainInput{
		Hostname: "agency.example.test",
		Kind:     DomainKindBaseDomain,
		Source:   DomainSourceUser,
		DNSState: DomainDNSStateReady,
	})
	if err != nil {
		t.Fatalf("create base domain: %v", err)
	}
	updatedName := "Changed"
	updatedStatus := SiteStatusActive
	invalidPrimary := "agency.example.test"

	_, err = store.Update(context.Background(), siteID, UpdateSiteInput{
		Name:          &updatedName,
		Status:        &updatedStatus,
		PrimaryDomain: &invalidPrimary,
	})
	if err == nil || !strings.Contains(err.Error(), "cannot be used as a site primary hostname") {
		t.Fatalf("update error = %v, want primary hostname assignment failure", err)
	}

	site, err := store.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site after failed update: %v", err)
	}
	if site.Name != "Original" {
		t.Fatalf("name after failed update = %q, want %q", site.Name, "Original")
	}
	if site.Status != SiteStatusDraft {
		t.Fatalf("status after failed update = %q, want %q", site.Status, SiteStatusDraft)
	}
	if site.PrimaryDomain != "original.example.test" {
		t.Fatalf("primary_domain after failed update = %q, want %q", site.PrimaryDomain, "original.example.test")
	}
}

func TestSiteStoreDelete(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Delete Me", WordPressAdminEmail: "owner@example.test", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	if err := store.Delete(context.Background(), siteID); err != nil {
		t.Fatalf("delete site: %v", err)
	}
	if _, err := store.GetByID(context.Background(), siteID); err == nil {
		t.Fatal("expected deleted site lookup to fail")
	}
}

func TestSiteStoreValidationAndNotFound(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	if _, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "", WordPressAdminEmail: "owner@example.test", Status: SiteStatusDraft}); err == nil {
		t.Fatal("expected create validation error")
	}
	if _, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Bad", WordPressAdminEmail: "owner@example.test", Status: "wrong"}); err == nil {
		t.Fatal("expected status validation error")
	}
	if _, err := store.GetByID(context.Background(), testPublicID(999)); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
	if err := store.Delete(context.Background(), testPublicID(999)); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
}
