package server

import (
	"database/sql"

	"pressluft/internal/controlplane/server/stores"
)

// Re-export domain types for backward compatibility.
type StoredDomain = stores.StoredDomain
type CreateDomainInput = stores.CreateDomainInput
type UpdateDomainInput = stores.UpdateDomainInput
type DomainStore = stores.DomainStore

// Re-export domain constants for backward compatibility.
const (
	DomainKindHostname   = stores.DomainKindHostname
	DomainKindBaseDomain = stores.DomainKindBaseDomain

	DomainSourceUser             = stores.DomainSourceUser
	DomainSourceFallbackResolver = stores.DomainSourceFallbackResolver

	DomainDNSStatePending  = stores.DomainDNSStatePending
	DomainDNSStateReady    = stores.DomainDNSStateReady
	DomainDNSStateIssue    = stores.DomainDNSStateIssue
	DomainDNSStateDisabled = stores.DomainDNSStateDisabled

	DomainRoutingStateNotConfigured = stores.DomainRoutingStateNotConfigured
	DomainRoutingStatePending       = stores.DomainRoutingStatePending
	DomainRoutingStateReady         = stores.DomainRoutingStateReady
	DomainRoutingStateIssue         = stores.DomainRoutingStateIssue
)

// NewDomainStore creates a new domain store.
func NewDomainStore(db *sql.DB) *DomainStore {
	return stores.NewDomainStore(db)
}
