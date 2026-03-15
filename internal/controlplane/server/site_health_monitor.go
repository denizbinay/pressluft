package server

import (
	"log/slog"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/controlplane/server/health"
	"pressluft/internal/shared/ws"
)

// SiteHealthMonitor is a re-export of the health.SiteHealthMonitor type.
type SiteHealthMonitor = health.SiteHealthMonitor

// NewSiteHealthMonitor creates a new site health monitor.
func NewSiteHealthMonitor(siteStore *SiteStore, domainStore *DomainStore, activityStore *activity.Store, hub *ws.Hub, logger *slog.Logger) *SiteHealthMonitor {
	return health.NewSiteHealthMonitor(siteStore, domainStore, activityStore, hub, logger)
}
