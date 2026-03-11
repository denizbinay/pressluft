package server

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDomainStoreCreateListAndPrimaryAssignment(t *testing.T) {
	db := mustOpenTestDB(t)
	siteStore := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")

	siteID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Northwind", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	baseID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "sandbox.pressluft.test",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipPlatform,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create base domain: %v", err)
	}
	firstID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:       "northwind.sandbox.pressluft.test",
		Kind:           DomainKindDirect,
		Ownership:      DomainOwnershipPlatform,
		Status:         DomainStatusActive,
		SiteID:         siteID,
		ParentDomainID: baseID,
		IsPrimary:      true,
	})
	if err != nil {
		t.Fatalf("create first hostname: %v", err)
	}
	secondID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "www.northwind.example.com",
		Kind:      DomainKindDirect,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusActive,
		SiteID:    siteID,
		IsPrimary: true,
	})
	if err != nil {
		t.Fatalf("create second hostname: %v", err)
	}
	first, err := domainStore.GetByID(context.Background(), firstID)
	if err != nil {
		t.Fatalf("get first hostname: %v", err)
	}
	second, err := domainStore.GetByID(context.Background(), secondID)
	if err != nil {
		t.Fatalf("get second hostname: %v", err)
	}
	if first.IsPrimary {
		t.Fatal("expected first hostname to no longer be primary")
	}
	if !second.IsPrimary {
		t.Fatal("expected second hostname to be primary")
	}
	storedSite, err := siteStore.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if storedSite.PrimaryDomain != "www.northwind.example.com" {
		t.Fatalf("primary_domain = %q, want %q", storedSite.PrimaryDomain, "www.northwind.example.com")
	}
	domains, err := domainStore.ListBySite(context.Background(), siteID)
	if err != nil {
		t.Fatalf("list domains by site: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("site domain count = %d, want 2", len(domains))
	}
}

func TestDomainStoreBackfillsLegacyPrimaryDomains(t *testing.T) {
	db := mustOpenTestDB(t)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID := nextTestPublicID(t, db, "sites")
	if _, err := db.Exec(
		`INSERT INTO sites (id, server_id, name, primary_domain, status, created_at, updated_at) VALUES (?, ?, ?, ?, 'active', ?, ?)`,
		siteID,
		serverID,
		"Legacy Site",
		"legacy.example.test",
		time.Now().UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
	); err != nil {
		t.Fatalf("insert legacy site: %v", err)
	}
	if err := domainStore.BackfillLegacyPrimaryDomains(context.Background()); err != nil {
		t.Fatalf("backfill legacy domains: %v", err)
	}
	domains, err := domainStore.ListBySite(context.Background(), siteID)
	if err != nil {
		t.Fatalf("list domains by site: %v", err)
	}
	if len(domains) != 1 {
		t.Fatalf("backfilled domain count = %d, want 1", len(domains))
	}
	if domains[0].Kind != DomainKindDirect {
		t.Fatalf("kind = %q, want %q", domains[0].Kind, DomainKindDirect)
	}
	if domains[0].Ownership != DomainOwnershipCustomer {
		t.Fatalf("ownership = %q, want %q", domains[0].Ownership, DomainOwnershipCustomer)
	}
	if !domains[0].IsPrimary {
		t.Fatal("expected backfilled domain to be primary")
	}
}

func TestDomainStoreEnsurePlatformBaseDomains(t *testing.T) {
	db := mustOpenTestDB(t)
	domainStore := NewDomainStore(db)

	if err := domainStore.EnsurePlatformBaseDomains(context.Background()); err != nil {
		t.Fatalf("ensure platform base domains: %v", err)
	}
	if err := domainStore.EnsurePlatformBaseDomains(context.Background()); err != nil {
		t.Fatalf("ensure platform base domains second pass: %v", err)
	}

	domains, err := domainStore.List(context.Background())
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if len(domains) != 2 {
		t.Fatalf("domain count = %d, want 2", len(domains))
	}

	byHostname := map[string]StoredDomain{}
	for _, domain := range domains {
		byHostname[domain.Hostname] = domain
	}

	if byHostname["pressluft.bombig.app"].Status != DomainStatusActive {
		t.Fatalf("pressluft.bombig.app status = %q, want %q", byHostname["pressluft.bombig.app"].Status, DomainStatusActive)
	}
	if byHostname["pressluft.dev"].Status != DomainStatusPending {
		t.Fatalf("pressluft.dev status = %q, want %q", byHostname["pressluft.dev"].Status, DomainStatusPending)
	}
}

func TestDomainStoreSetPrimaryHostnameForSiteRejectsAttachedHostnameConflict(t *testing.T) {
	db := mustOpenTestDB(t)
	siteStore := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteOneID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "One", PrimaryDomain: "one.example.test", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create first site: %v", err)
	}
	siteTwoID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Two", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create second site: %v", err)
	}

	err = domainStore.SetPrimaryHostnameForSite(context.Background(), siteTwoID, "one.example.test", DomainOwnershipCustomer)
	if err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("set primary error = %v, want hostname conflict", err)
	}

	siteOne, err := siteStore.GetByID(context.Background(), siteOneID)
	if err != nil {
		t.Fatalf("get first site: %v", err)
	}
	if siteOne.PrimaryDomain != "one.example.test" {
		t.Fatalf("first site primary_domain = %q, want %q", siteOne.PrimaryDomain, "one.example.test")
	}
	siteTwo, err := siteStore.GetByID(context.Background(), siteTwoID)
	if err != nil {
		t.Fatalf("get second site: %v", err)
	}
	if siteTwo.PrimaryDomain != "" {
		t.Fatalf("second site primary_domain = %q, want empty", siteTwo.PrimaryDomain)
	}
}

func TestDomainStoreDeletePromotesReplacementPrimary(t *testing.T) {
	db := mustOpenTestDB(t)
	siteStore := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Northwind", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}
	primaryID, err := domainStore.Create(context.Background(), CreateDomainInput{Hostname: "primary.example.test", Kind: DomainKindDirect, Ownership: DomainOwnershipCustomer, Status: DomainStatusActive, SiteID: siteID, IsPrimary: true})
	if err != nil {
		t.Fatalf("create primary domain: %v", err)
	}
	secondaryID, err := domainStore.Create(context.Background(), CreateDomainInput{Hostname: "secondary.example.test", Kind: DomainKindDirect, Ownership: DomainOwnershipCustomer, Status: DomainStatusActive, SiteID: siteID})
	if err != nil {
		t.Fatalf("create secondary domain: %v", err)
	}

	if err := domainStore.Delete(context.Background(), primaryID); err != nil {
		t.Fatalf("delete primary domain: %v", err)
	}

	secondary, err := domainStore.GetByID(context.Background(), secondaryID)
	if err != nil {
		t.Fatalf("get promoted domain: %v", err)
	}
	if !secondary.IsPrimary {
		t.Fatal("expected secondary domain to be promoted to primary")
	}
	site, err := siteStore.GetByID(context.Background(), siteID)
	if err != nil {
		t.Fatalf("get site: %v", err)
	}
	if site.PrimaryDomain != "secondary.example.test" {
		t.Fatalf("site primary_domain = %q, want %q", site.PrimaryDomain, "secondary.example.test")
	}
}

func TestDomainStoreMovingPrimaryPromotesReplacementOnPreviousSite(t *testing.T) {
	db := mustOpenTestDB(t)
	siteStore := NewSiteStore(db)
	domainStore := NewDomainStore(db)
	serverID := mustInsertServerWithStatus(t, db, "ready")
	siteOneID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "One", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create first site: %v", err)
	}
	siteTwoID, err := siteStore.Create(context.Background(), CreateSiteInput{ServerID: serverID, Name: "Two", Status: SiteStatusDraft})
	if err != nil {
		t.Fatalf("create second site: %v", err)
	}
	movedID, err := domainStore.Create(context.Background(), CreateDomainInput{Hostname: "primary.example.test", Kind: DomainKindDirect, Ownership: DomainOwnershipCustomer, Status: DomainStatusActive, SiteID: siteOneID, IsPrimary: true})
	if err != nil {
		t.Fatalf("create moved domain: %v", err)
	}
	replacementID, err := domainStore.Create(context.Background(), CreateDomainInput{Hostname: "secondary.example.test", Kind: DomainKindDirect, Ownership: DomainOwnershipCustomer, Status: DomainStatusActive, SiteID: siteOneID})
	if err != nil {
		t.Fatalf("create replacement domain: %v", err)
	}
	isPrimary := true
	newSiteID := siteTwoID
	_, err = domainStore.Update(context.Background(), movedID, UpdateDomainInput{SiteID: &newSiteID, IsPrimary: &isPrimary})
	if err != nil {
		t.Fatalf("move primary domain: %v", err)
	}

	replacement, err := domainStore.GetByID(context.Background(), replacementID)
	if err != nil {
		t.Fatalf("get replacement domain: %v", err)
	}
	if !replacement.IsPrimary {
		t.Fatal("expected replacement domain to become primary on previous site")
	}
	siteOne, err := siteStore.GetByID(context.Background(), siteOneID)
	if err != nil {
		t.Fatalf("get first site: %v", err)
	}
	if siteOne.PrimaryDomain != "secondary.example.test" {
		t.Fatalf("first site primary_domain = %q, want %q", siteOne.PrimaryDomain, "secondary.example.test")
	}
	siteTwo, err := siteStore.GetByID(context.Background(), siteTwoID)
	if err != nil {
		t.Fatalf("get second site: %v", err)
	}
	if siteTwo.PrimaryDomain != "primary.example.test" {
		t.Fatalf("second site primary_domain = %q, want %q", siteTwo.PrimaryDomain, "primary.example.test")
	}
}

func TestDomainStoreRejectsDirectDomainOutsideWildcardParent(t *testing.T) {
	db := mustOpenTestDB(t)
	domainStore := NewDomainStore(db)

	parentID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "example.test",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create wildcard domain: %v", err)
	}

	_, err = domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:       "outside.test",
		Kind:           DomainKindDirect,
		Ownership:      DomainOwnershipCustomer,
		Status:         DomainStatusActive,
		ParentDomainID: parentID,
	})
	if err == nil || !strings.Contains(err.Error(), "within the parent wildcard domain") {
		t.Fatalf("create error = %v, want wildcard parent validation failure", err)
	}
}

func TestDomainStoreRejectsDeletingPlatformWildcardDomain(t *testing.T) {
	db := mustOpenTestDB(t)
	domainStore := NewDomainStore(db)

	id, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "pressluft.dev",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipPlatform,
		Status:    DomainStatusActive,
	})
	if err != nil {
		t.Fatalf("create platform wildcard domain: %v", err)
	}

	err = domainStore.Delete(context.Background(), id)
	if err == nil || !strings.Contains(err.Error(), "cannot be deleted") {
		t.Fatalf("delete error = %v, want platform wildcard delete failure", err)
	}
}

func TestDomainStoreRejectsChildDomainForInactiveWildcardParent(t *testing.T) {
	db := mustOpenTestDB(t)
	domainStore := NewDomainStore(db)

	parentID, err := domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:  "agency.dev",
		Kind:      DomainKindWildcard,
		Ownership: DomainOwnershipCustomer,
		Status:    DomainStatusPending,
	})
	if err != nil {
		t.Fatalf("create wildcard domain: %v", err)
	}

	_, err = domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:       "preview.agency.dev",
		Kind:           DomainKindDirect,
		Ownership:      DomainOwnershipCustomer,
		Status:         DomainStatusActive,
		ParentDomainID: parentID,
	})
	if err == nil || !strings.Contains(err.Error(), "active wildcard domain") {
		t.Fatalf("create error = %v, want active wildcard parent failure", err)
	}
}

func TestDomainStoreRejectsChildOwnershipMismatchForWildcardParent(t *testing.T) {
	db := mustOpenTestDB(t)
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

	_, err = domainStore.Create(context.Background(), CreateDomainInput{
		Hostname:       "preview.agency.dev",
		Kind:           DomainKindDirect,
		Ownership:      DomainOwnershipPlatform,
		Status:         DomainStatusActive,
		ParentDomainID: parentID,
	})
	if err == nil || !strings.Contains(err.Error(), "ownership must match") {
		t.Fatalf("create error = %v, want ownership mismatch failure", err)
	}
}
