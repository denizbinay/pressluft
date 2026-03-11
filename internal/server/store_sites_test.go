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
		ServerID:         serverID,
		Name:             "Agency Brochure",
		PrimaryDomain:    "brochure.example.test",
		Status:           SiteStatusDraft,
		WordPressPath:    "/srv/www/brochure/current",
		PHPVersion:       "8.3",
		WordPressVersion: "6.8",
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

func TestSiteStoreListByServer(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	otherServerID := mustInsertServerWithStatus(t, db, "ready")

	_, _ = store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "One", Status: SiteStatusDraft})
	_, _ = store.Create(context.Background(), CreateSiteInput{ServerID: otherServerID, Name: "Two", Status: SiteStatusActive})

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
	siteID, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Original", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	updatedName := "Client Store"
	updatedStatus := SiteStatusActive
	updatedDomain := "store.example.test"
	updatedPath := "/srv/www/store/current"
	updatedPHP := "8.2"
	updatedWP := "6.7"

	site, err := store.Update(context.Background(), siteID, UpdateSiteInput{
		Name:             &updatedName,
		Status:           &updatedStatus,
		PrimaryDomain:    &updatedDomain,
		WordPressPath:    &updatedPath,
		PHPVersion:       &updatedPHP,
		WordPressVersion: &updatedWP,
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
}

func TestSiteStoreDelete(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewSiteStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Delete Me", Status: SiteStatusDraft})
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

	if _, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "", Status: SiteStatusDraft}); err == nil {
		t.Fatal("expected create validation error")
	}
	if _, err := store.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Bad", Status: "wrong"}); err == nil {
		t.Fatal("expected status validation error")
	}
	if _, err := store.GetByID(context.Background(), testPublicID(999)); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
	if err := store.Delete(context.Background(), testPublicID(999)); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("err = %v, want not found", err)
	}
}
