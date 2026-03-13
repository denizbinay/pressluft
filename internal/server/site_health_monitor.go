package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/agentcommand"
	"pressluft/internal/apitypes"
	"pressluft/internal/ws"

	"github.com/google/uuid"
)

type SiteHealthMonitor struct {
	siteStore      *SiteStore
	domainStore    *DomainStore
	activityStore  *activity.Store
	hub            *ws.Hub
	logger         *slog.Logger
	interval       time.Duration
	requestTimeout time.Duration
}

func NewSiteHealthMonitor(siteStore *SiteStore, domainStore *DomainStore, activityStore *activity.Store, hub *ws.Hub, logger *slog.Logger) *SiteHealthMonitor {
	if logger == nil {
		logger = slog.Default()
	}
	return &SiteHealthMonitor{
		siteStore:      siteStore,
		domainStore:    domainStore,
		activityStore:  activityStore,
		hub:            hub,
		logger:         logger,
		interval:       1 * time.Minute,
		requestTimeout: 25 * time.Second,
	}
}

func (m *SiteHealthMonitor) Start(ctx context.Context) {
	if m == nil || m.siteStore == nil {
		return
	}
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()
	m.reconcileAll(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.reconcileAll(ctx)
		}
	}
}

func (m *SiteHealthMonitor) reconcileAll(ctx context.Context) {
	sites, err := m.siteStore.List(ctx)
	if err != nil {
		m.logger.Error("site health reconcile failed to list sites", "error", err)
		return
	}
	for _, site := range sites {
		if site.DeploymentState != SiteDeploymentStateReady {
			continue
		}
		if strings.TrimSpace(site.PrimaryDomain) == "" {
			continue
		}
		siteCtx, cancel := context.WithTimeout(ctx, m.requestTimeout)
		m.reconcileSite(siteCtx, site)
		cancel()
	}
}

func (m *SiteHealthMonitor) reconcileSite(ctx context.Context, site StoredSite) {
	now := time.Now().UTC()
	routingState := DomainRoutingStateReady
	routingMessage := "Hostname routing verified over HTTPS."
	runtimeState := SiteRuntimeHealthStateHealthy
	runtimeMessage := "Public runtime checks passed."

	if err := VerifyPublicSiteRouting(ctx, site.ID, site.PrimaryDomain); err != nil {
		routingState = DomainRoutingStateIssue
		routingMessage = err.Error()
		runtimeState = SiteRuntimeHealthStateIssue
		runtimeMessage = "Public routing check failed: " + err.Error()
	} else if err := VerifyPublicWordPressRuntime(ctx, site.PrimaryDomain); err != nil {
		runtimeState = SiteRuntimeHealthStateIssue
		runtimeMessage = "Public runtime check failed: " + err.Error()
	}

	if m.domainStore != nil {
		if domain, err := m.primaryDomainForSite(ctx, site.ID); err == nil {
			_ = m.domainStore.UpdateRoutingStatus(ctx, domain.ID, routingState, routingMessage, now)
		}
	}

	if m.hub != nil {
		info := m.hub.GetAgentInfo(site.ServerID)
		if info.Connected {
			if snapshot, err := m.fetchSiteHealthSnapshot(ctx, site); err == nil {
				if state, message := RuntimeHealthFromAgentSnapshot(snapshot); state == SiteRuntimeHealthStateIssue {
					runtimeState = state
					runtimeMessage = "Agent diagnostics: " + message
				} else if runtimeState == SiteRuntimeHealthStateHealthy {
					runtimeMessage = "Public and managed-server runtime checks passed."
				}
			} else if runtimeState == SiteRuntimeHealthStateHealthy {
				runtimeState = SiteRuntimeHealthStateUnknown
				runtimeMessage = "Public checks passed, but managed-server diagnostics failed: " + err.Error()
			}
		} else if runtimeState == SiteRuntimeHealthStateHealthy {
			runtimeMessage = "Public checks passed; managed-server diagnostics are currently offline."
		}
	}

	_ = m.siteStore.UpdateRuntimeHealth(ctx, site.ID, runtimeState, runtimeMessage, now.Format(time.RFC3339))
	if site.RuntimeHealthState != runtimeState {
		m.emitTransition(ctx, site, runtimeState, runtimeMessage)
	}
}

func (m *SiteHealthMonitor) fetchSiteHealthSnapshot(ctx context.Context, site StoredSite) (*agentcommand.SiteHealthSnapshot, error) {
	payload, err := json.Marshal(agentcommand.SiteHealthSnapshotParams{
		SiteID:   site.ID,
		Hostname: site.PrimaryDomain,
		SitePath: siteWordPressRootPath(site),
	})
	if err != nil {
		return nil, err
	}
	result, err := m.hub.SendCommandAndWait(ctx, site.ServerID, ws.Command{
		ID:       uuid.NewString(),
		ServerID: apitypes.FormatAppID(site.ServerID),
		Type:     agentcommand.TypeSiteHealth,
		Payload:  payload,
	})
	if err != nil {
		return nil, err
	}
	if !result.Success {
		return nil, errors.New(result.Error)
	}
	var snapshot agentcommand.SiteHealthSnapshot
	if err := json.Unmarshal(result.Payload, &snapshot); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (m *SiteHealthMonitor) primaryDomainForSite(ctx context.Context, siteID string) (*StoredDomain, error) {
	if m.domainStore == nil {
		return nil, fmt.Errorf("domain store not configured")
	}
	domains, err := m.domainStore.ListBySite(ctx, siteID)
	if err != nil {
		return nil, err
	}
	for i := range domains {
		if domains[i].IsPrimary {
			return &domains[i], nil
		}
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("site has no assigned hostname")
	}
	return &domains[0], nil
}

func (m *SiteHealthMonitor) emitTransition(ctx context.Context, site StoredSite, runtimeState, runtimeMessage string) {
	if m.activityStore == nil {
		return
	}
	level := activity.LevelSuccess
	title := fmt.Sprintf("Site '%s' runtime health recovered", site.Name)
	requiresAttention := false
	if runtimeState == SiteRuntimeHealthStateIssue {
		level = activity.LevelWarning
		title = fmt.Sprintf("Site '%s' needs runtime attention", site.Name)
		requiresAttention = true
	}
	_, _ = m.activityStore.Emit(ctx, activity.EmitInput{
		EventType:          activity.EventSiteHealthChanged,
		Category:           activity.CategorySite,
		Level:              level,
		ResourceType:       activity.ResourceSite,
		ResourceID:         site.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   site.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              title,
		Message:            runtimeMessage,
		RequiresAttention:  requiresAttention,
	})
}
