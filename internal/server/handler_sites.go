package server

import (
	"fmt"
	"net/http"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
)

type sitesHandler struct {
	store           *SiteStore
	domainStore     *DomainStore
	activityStore   *activity.Store
	activityHandler *activityHandler
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
	id, err := sh.store.Create(r.Context(), CreateSiteInput{
		ServerID:            req.ServerID,
		Name:                req.Name,
		PrimaryDomain:       req.PrimaryDomain,
		PrimaryDomainConfig: apiCreateSitePrimaryDomainConfig(req.PrimaryDomainConfig),
		Status:              status,
		WordPressPath:       req.WordPressPath,
		PHPVersion:          req.PHPVersion,
		WordPressVersion:    req.WordPressVersion,
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "server ") && strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, err.Error())
		case strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "unsupported site status") || strings.Contains(err.Error(), "primary_domain_config") || strings.Contains(err.Error(), "use either") || strings.Contains(err.Error(), "valid domain name") || strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "cannot"):
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
	respondJSON(w, http.StatusCreated, apiStoredSite(*site))
}

func apiCreateSitePrimaryDomainConfig(in *apitypes.SitePrimaryDomainConfig) *CreateSitePrimaryDomainInput {
	if in == nil {
		return nil
	}
	return &CreateSitePrimaryDomainInput{
		Mode:           in.Mode,
		Hostname:       in.Hostname,
		Label:          in.Label,
		ParentDomainID: in.ParentDomainID,
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
		WordPressPath:    in.WordPressPath,
		PHPVersion:       in.PHPVersion,
		WordPressVersion: in.WordPressVersion,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
	}
}
