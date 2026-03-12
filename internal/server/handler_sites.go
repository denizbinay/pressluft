package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
)

type sitesHandler struct {
	store           *SiteStore
	serverStore     *ServerStore
	jobStore        *orchestrator.Store
	domainStore     *DomainStore
	activityStore   *activity.Store
	activityHandler *activityHandler
}

type deploySiteJobPayload struct {
	SiteID string `json:"site_id"`
}

func (sh *sitesHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/sites" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		sh.handleList(w, r)
	case http.MethodPost:
		sh.handleCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (sh *sitesHandler) routeWithID(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/sites/")
	parts := strings.Split(strings.Trim(tail, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	siteID, err := apitypes.ParseAppID(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid site id")
		return
	}
	if len(parts) == 2 && parts[1] == "activity" {
		if sh.activityHandler == nil {
			http.NotFound(w, r)
			return
		}
		sh.activityHandler.handleSiteActivity(w, r, siteID)
		return
	}
	if len(parts) == 2 && parts[1] == "domains" {
		if sh.domainStore == nil {
			http.NotFound(w, r)
			return
		}
		dh := &domainsHandler{store: sh.domainStore, activityStore: sh.activityStore}
		if r.Method == http.MethodGet {
			dh.handleListBySite(w, r, siteID)
			return
		}
		if r.Method == http.MethodPost {
			dh.handleCreateForSite(w, r, siteID)
			return
		}
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if len(parts) != 1 {
		http.NotFound(w, r)
		return
	}
	sh.routeSiteByID(w, r, siteID)
}

func (sh *sitesHandler) routeSiteByID(w http.ResponseWriter, r *http.Request, siteID string) {
	switch r.Method {
	case http.MethodGet:
		sh.handleGet(w, r, siteID)
	case http.MethodPatch:
		sh.handleUpdate(w, r, siteID)
	case http.MethodDelete:
		sh.handleDelete(w, r, siteID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (sh *sitesHandler) handleList(w http.ResponseWriter, r *http.Request) {
	sites, err := sh.store.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list sites: "+err.Error())
		return
	}
	if sites == nil {
		respondJSON(w, http.StatusOK, []apitypes.StoredSite{})
		return
	}
	payload := make([]apitypes.StoredSite, 0, len(sites))
	for _, site := range sites {
		payload = append(payload, apiStoredSite(site))
	}
	respondJSON(w, http.StatusOK, payload)
}

func (sh *sitesHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req apitypes.CreateSiteRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	status := strings.TrimSpace(req.Status)
	if status == "" {
		status = SiteStatusDraft
	}
	if err := sh.ensureCreateTargetSupported(r, req.ServerID); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := sh.store.Create(r.Context(), CreateSiteInput{
		ServerID:              req.ServerID,
		Name:                  req.Name,
		PrimaryDomain:         req.PrimaryDomain,
		PrimaryHostnameConfig: apiCreateSitePrimaryHostnameConfig(req.PrimaryHostnameConfig),
		Status:                status,
		WordPressPath:         req.WordPressPath,
		PHPVersion:            req.PHPVersion,
		WordPressVersion:      req.WordPressVersion,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "server ") && strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported site status") || strings.Contains(err.Error(), "primary_hostname_config") || strings.Contains(err.Error(), "use either") || strings.Contains(err.Error(), "valid domain name") || strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "cannot") || strings.Contains(err.Error(), "fallback resolver hostnames") || strings.Contains(err.Error(), "IPv4 address"):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to create site: "+err.Error())
		}
		return
	}
	site, err := sh.store.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if sh.activityStore != nil {
		actorType, actorID := activityActorFromRequest(r)
		_, _ = sh.activityStore.Emit(r.Context(), activity.EmitInput{
			EventType:          activity.EventSiteCreated,
			Category:           activity.CategorySite,
			Level:              activity.LevelInfo,
			ResourceType:       activity.ResourceSite,
			ResourceID:         site.ID,
			ParentResourceType: activity.ResourceServer,
			ParentResourceID:   site.ServerID,
			ActorType:          actorType,
			ActorID:            actorID,
			Title:              fmt.Sprintf("Site '%s' created", site.Name),
			Message:            "The site is now tracked as a first-class resource in the control plane.",
		})
	}
	if sh.jobStore != nil && strings.TrimSpace(site.PrimaryDomain) != "" {
		payload, err := json.Marshal(deploySiteJobPayload{SiteID: site.ID})
		if err != nil {
			respondError(w, http.StatusInternalServerError, "failed to queue site deployment: marshal payload")
			return
		}
		job, err := sh.jobStore.CreateJob(r.Context(), orchestrator.CreateJobInput{
			Kind:     string(orchestrator.JobKindDeploySite),
			ServerID: site.ServerID,
			Payload:  string(payload),
		})
		if err != nil {
			_ = sh.store.UpdateDeployment(r.Context(), site.ID, SiteDeploymentStateFailed, "Failed to queue site deployment.", "", "")
			respondError(w, http.StatusInternalServerError, "failed to queue site deployment: "+err.Error())
			return
		}
		_, _ = sh.jobStore.AppendEvent(r.Context(), job.ID, orchestrator.CreateEventInput{
			EventType: orchestrator.JobEventTypeCreated,
			Level:     "info",
			Status:    string(job.Status),
			Message:   "Site deployment accepted and queued",
		})
		_ = sh.store.UpdateDeployment(r.Context(), site.ID, SiteDeploymentStateDeploying, "Site deployment queued.", job.ID, "")
		site, _ = sh.store.GetByID(r.Context(), id)
	}
	respondJSON(w, http.StatusCreated, apiStoredSite(*site))
}

func (sh *sitesHandler) ensureCreateTargetSupported(r *http.Request, serverID string) error {
	if sh.serverStore == nil {
		return nil
	}
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		return err
	}
	if server.Status != platform.ServerStatusReady {
		return fmt.Errorf("selected server must be in ready status before a site can be deployed")
	}
	if server.SetupState != platform.SetupStateReady {
		return fmt.Errorf("selected server setup must be ready before a site can be deployed")
	}
	if strings.TrimSpace(server.ProfileKey) != "nginx-stack" {
		return fmt.Errorf("selected server profile %q is not supported for site deployment", server.ProfileKey)
	}
	return nil
}

func apiCreateSitePrimaryHostnameConfig(in *apitypes.SitePrimaryHostnameConfig) *CreateSitePrimaryHostnameInput {
	if in == nil {
		return nil
	}
	return &CreateSitePrimaryHostnameInput{
		Source:   in.Source,
		Hostname: in.Hostname,
		Label:    in.Label,
		DomainID: in.DomainID,
	}
}

func (sh *sitesHandler) handleGet(w http.ResponseWriter, r *http.Request, siteID string) {
	site, err := sh.store.GetByID(r.Context(), siteID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apiStoredSite(*site))
}

func (sh *sitesHandler) handleUpdate(w http.ResponseWriter, r *http.Request, siteID string) {
	var req apitypes.UpdateSiteRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	site, err := sh.store.Update(r.Context(), siteID, UpdateSiteInput{
		Name:             req.Name,
		PrimaryDomain:    req.PrimaryDomain,
		Status:           req.Status,
		WordPressPath:    req.WordPressPath,
		PHPVersion:       req.PHPVersion,
		WordPressVersion: req.WordPressVersion,
		ServerID:         req.ServerID,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported site status") || strings.Contains(err.Error(), "valid domain name") || strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "cannot"):
			respondError(w, http.StatusBadRequest, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, "failed to update site: "+err.Error())
		}
		return
	}
	if sh.activityStore != nil {
		actorType, actorID := activityActorFromRequest(r)
		_, _ = sh.activityStore.Emit(r.Context(), activity.EmitInput{
			EventType:          activity.EventSiteUpdated,
			Category:           activity.CategorySite,
			Level:              activity.LevelInfo,
			ResourceType:       activity.ResourceSite,
			ResourceID:         site.ID,
			ParentResourceType: activity.ResourceServer,
			ParentResourceID:   site.ServerID,
			ActorType:          actorType,
			ActorID:            actorID,
			Title:              fmt.Sprintf("Site '%s' updated", site.Name),
			Message:            "Site metadata was updated in the control plane.",
		})
	}
	respondJSON(w, http.StatusOK, apiStoredSite(*site))
}

func (sh *sitesHandler) handleDelete(w http.ResponseWriter, r *http.Request, siteID string) {
	site, err := sh.store.GetByID(r.Context(), siteID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := sh.store.Delete(r.Context(), siteID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to delete site: "+err.Error())
		return
	}
	if sh.activityStore != nil {
		actorType, actorID := activityActorFromRequest(r)
		_, _ = sh.activityStore.Emit(r.Context(), activity.EmitInput{
			EventType:          activity.EventSiteDeleted,
			Category:           activity.CategorySite,
			Level:              activity.LevelInfo,
			ResourceType:       activity.ResourceSite,
			ResourceID:         site.ID,
			ParentResourceType: activity.ResourceServer,
			ParentResourceID:   site.ServerID,
			ActorType:          actorType,
			ActorID:            actorID,
			Title:              fmt.Sprintf("Site '%s' deleted", site.Name),
			Message:            "The site was removed from the Pressluft inventory.",
		})
	}
	respondJSON(w, http.StatusOK, apitypes.DeleteSiteResponse{
		SiteID:      apitypes.FormatAppID(site.ID),
		Deleted:     true,
		Description: "Site deleted",
	})
}

func apiStoredSite(in StoredSite) apitypes.StoredSite {
	return apitypes.StoredSite{
		ID:               apitypes.FormatAppID(in.ID),
		ServerID:         apitypes.FormatAppID(in.ServerID),
		ServerName:       in.ServerName,
		Name:             in.Name,
		PrimaryDomain:    in.PrimaryDomain,
		Status:           in.Status,
		DeploymentState:  in.DeploymentState,
		DeploymentStatus: in.DeploymentStatus,
		LastDeployJobID:  apitypes.FormatAppID(in.LastDeployJobID),
		LastDeployedAt:   in.LastDeployedAt,
		WordPressPath:    in.WordPressPath,
		PHPVersion:       in.PHPVersion,
		WordPressVersion: in.WordPressVersion,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
	}
}
