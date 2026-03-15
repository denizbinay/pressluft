package server

import (
	"database/sql"

	"pressluft/internal/controlplane/server/stores"
)

// Re-export site types for backward compatibility.
type StoredSite = stores.StoredSite
type CreateSiteInput = stores.CreateSiteInput
type CreateSitePrimaryHostnameInput = stores.CreateSitePrimaryHostnameInput
type UpdateSiteInput = stores.UpdateSiteInput
type SiteStore = stores.SiteStore

// Re-export site constants for backward compatibility.
const (
	SiteStatusDraft     = stores.SiteStatusDraft
	SiteStatusActive    = stores.SiteStatusActive
	SiteStatusAttention = stores.SiteStatusAttention
	SiteStatusArchived  = stores.SiteStatusArchived

	SiteDeploymentStatePending   = stores.SiteDeploymentStatePending
	SiteDeploymentStateDeploying = stores.SiteDeploymentStateDeploying
	SiteDeploymentStateReady     = stores.SiteDeploymentStateReady
	SiteDeploymentStateFailed    = stores.SiteDeploymentStateFailed

	SiteRuntimeHealthStatePending = stores.SiteRuntimeHealthStatePending
	SiteRuntimeHealthStateHealthy = stores.SiteRuntimeHealthStateHealthy
	SiteRuntimeHealthStateIssue   = stores.SiteRuntimeHealthStateIssue
	SiteRuntimeHealthStateUnknown = stores.SiteRuntimeHealthStateUnknown
)

// Re-export site functions for backward compatibility.
var (
	NewSiteStore                    = stores.NewSiteStore
	AllSiteStatuses                 = stores.AllSiteStatuses
	AllSiteDeploymentStates         = stores.AllSiteDeploymentStates
	AllSiteRuntimeHealthStates      = stores.AllSiteRuntimeHealthStates
	NormalizeSiteStatus             = stores.NormalizeSiteStatus
	NormalizeSiteDeploymentState    = stores.NormalizeSiteDeploymentState
	NormalizeSiteRuntimeHealthState = stores.NormalizeSiteRuntimeHealthState
)

// NewSiteStoreFromDB is an alias for NewSiteStore.
// Deprecated: Use NewSiteStore directly.
func NewSiteStoreFromDB(db *sql.DB) *SiteStore {
	return stores.NewSiteStore(db)
}
