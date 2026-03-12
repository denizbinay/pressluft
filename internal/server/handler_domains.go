package server

import (
	"fmt"
	"net/http"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
)

type domainsHandler struct {
	store         *DomainStore
	activityStore *activity.Store
}

func (dh *domainsHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/domains" {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		dh.handleList(w, r)
	case http.MethodPost:
		dh.handleCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (dh *domainsHandler) routeWithID(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/domains/")
	parts := strings.Split(strings.Trim(tail, "/"), "/")
	if len(parts) != 1 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}
	domainID, err := apitypes.ParseAppID(parts[0])
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid domain id")
		return
	}
	switch r.Method {
	case http.MethodGet:
		dh.handleGet(w, r, domainID)
	case http.MethodPatch:
		dh.handleUpdate(w, r, domainID)
	case http.MethodDelete:
		dh.handleDelete(w, r, domainID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (dh *domainsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	domains, err := dh.store.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list domains: "+err.Error())
		return
	}
	payload := make([]apitypes.StoredDomain, 0, len(domains))
	for _, domain := range domains {
		payload = append(payload, apiStoredDomain(domain))
	}
	respondJSON(w, http.StatusOK, payload)
}

func (dh *domainsHandler) handleListBySite(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	domains, err := dh.store.ListBySite(r.Context(), siteID)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.Contains(err.Error(), "site_id") || strings.Contains(err.Error(), "not found") {
			status = http.StatusBadRequest
		}
		respondError(w, status, err.Error())
		return
	}
	payload := make([]apitypes.StoredDomain, 0, len(domains))
	for _, domain := range domains {
		payload = append(payload, apiStoredDomain(domain))
	}
	respondJSON(w, http.StatusOK, payload)
}

func (dh *domainsHandler) handleCreateForSite(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req apitypes.CreateDomainRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	req.SiteID = siteID
	if strings.TrimSpace(req.Kind) == "" {
		req.Kind = DomainKindHostname
	}
	if strings.TrimSpace(req.Source) == "" {
		if strings.TrimSpace(req.ParentDomainID) != "" {
			req.Source = DomainSourceUser
		} else {
			req.Source = DomainSourceUser
		}
	}
	if strings.TrimSpace(req.DNSState) == "" && strings.TrimSpace(req.Source) == DomainSourceFallbackResolver {
		req.DNSState = DomainDNSStateReady
	}
	dh.createDomain(w, r, req)
}

func (dh *domainsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req apitypes.CreateDomainRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if strings.TrimSpace(req.Kind) == "" {
		req.Kind = DomainKindHostname
	}
	if strings.TrimSpace(req.Source) == "" {
		req.Source = DomainSourceUser
	}
	if strings.TrimSpace(req.DNSState) == "" && strings.TrimSpace(req.Source) == DomainSourceFallbackResolver {
		req.DNSState = DomainDNSStateReady
	}
	dh.createDomain(w, r, req)
}

func (dh *domainsHandler) createDomain(w http.ResponseWriter, r *http.Request, req apitypes.CreateDomainRequest) {
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := dh.store.Create(r.Context(), CreateDomainInput{
		Hostname:             req.Hostname,
		Kind:                 req.Kind,
		Source:               req.Source,
		DNSState:             req.DNSState,
		RoutingState:         req.RoutingState,
		DNSStatusMessage:     req.DNSStatusMessage,
		RoutingStatusMessage: req.RoutingStatusMessage,
		LastCheckedAt:        req.LastCheckedAt,
		SiteID:               req.SiteID,
		ParentDomainID:       req.ParentDomainID,
		IsPrimary:            req.IsPrimary,
	})
	if err != nil {
		respondDomainError(w, err, "failed to create domain")
		return
	}
	domain, err := dh.store.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	dh.emitDomainActivity(r, activity.EventDomainCreated, domain, fmt.Sprintf("Domain '%s' created", domain.Hostname), "The domain inventory was updated in the control plane.")
	if domain.SiteID != "" {
		dh.emitDomainActivity(r, activity.EventDomainAssigned, domain, fmt.Sprintf("Domain '%s' assigned", domain.Hostname), "The hostname is now attached to a site.")
	}
	respondJSON(w, http.StatusCreated, apiStoredDomain(*domain))
}

func (dh *domainsHandler) handleGet(w http.ResponseWriter, r *http.Request, domainID string) {
	domain, err := dh.store.GetByID(r.Context(), domainID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apiStoredDomain(*domain))
}

func (dh *domainsHandler) handleUpdate(w http.ResponseWriter, r *http.Request, domainID string) {
	var req apitypes.UpdateDomainRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	current, err := dh.store.GetByID(r.Context(), domainID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if current.SiteID != "" && (req.DNSState != nil || req.RoutingState != nil || req.DNSStatusMessage != nil || req.RoutingStatusMessage != nil || req.LastCheckedAt != nil) {
		respondError(w, http.StatusBadRequest, "attached site hostnames manage DNS and routing state automatically")
		return
	}
	domain, err := dh.store.Update(r.Context(), domainID, UpdateDomainInput{
		Hostname:             req.Hostname,
		Kind:                 req.Kind,
		Source:               req.Source,
		DNSState:             req.DNSState,
		RoutingState:         req.RoutingState,
		DNSStatusMessage:     req.DNSStatusMessage,
		RoutingStatusMessage: req.RoutingStatusMessage,
		LastCheckedAt:        req.LastCheckedAt,
		SiteID:               req.SiteID,
		ParentDomainID:       req.ParentDomainID,
		IsPrimary:            req.IsPrimary,
	})
	if err != nil {
		respondDomainError(w, err, "failed to update domain")
		return
	}
	dh.emitDomainActivity(r, activity.EventDomainUpdated, domain, fmt.Sprintf("Domain '%s' updated", domain.Hostname), "Domain metadata was updated in the control plane.")
	respondJSON(w, http.StatusOK, apiStoredDomain(*domain))
}

func (dh *domainsHandler) handleDelete(w http.ResponseWriter, r *http.Request, domainID string) {
	domain, err := dh.store.GetByID(r.Context(), domainID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if err := dh.store.Delete(r.Context(), domainID); err != nil {
		respondDomainError(w, err, "failed to delete domain")
		return
	}
	dh.emitDomainActivity(r, activity.EventDomainDeleted, domain, fmt.Sprintf("Domain '%s' deleted", domain.Hostname), "The domain record was removed from the Pressluft inventory.")
	respondJSON(w, http.StatusOK, apitypes.DeleteDomainResponse{DomainID: apitypes.FormatAppID(domain.ID), Deleted: true, Description: "Domain deleted"})
}

func (dh *domainsHandler) emitDomainActivity(r *http.Request, eventType activity.EventType, domain *StoredDomain, title, message string) {
	if dh.activityStore == nil || domain == nil {
		return
	}
	actorType, actorID := activityActorFromRequest(r)
	input := activity.EmitInput{
		EventType:    eventType,
		Category:     activity.CategoryDomain,
		Level:        activity.LevelInfo,
		ResourceType: activity.ResourceDomain,
		ResourceID:   domain.ID,
		ActorType:    actorType,
		ActorID:      actorID,
		Title:        title,
		Message:      message,
	}
	if domain.SiteID != "" {
		input.ParentResourceType = activity.ResourceSite
		input.ParentResourceID = domain.SiteID
	}
	_, _ = dh.activityStore.Emit(r.Context(), input)
}

func respondDomainError(w http.ResponseWriter, err error, prefix string) {
	switch {
	case strings.Contains(err.Error(), "not found"):
		respondError(w, http.StatusNotFound, err.Error())
	case strings.Contains(err.Error(), "required"), strings.Contains(err.Error(), "unsupported"), strings.Contains(err.Error(), "valid domain name"), strings.Contains(err.Error(), "already exists"), strings.Contains(err.Error(), "cannot"), strings.Contains(err.Error(), "must reference"), strings.Contains(err.Error(), "within the selected base domain"):
		respondError(w, http.StatusBadRequest, err.Error())
	default:
		respondError(w, http.StatusInternalServerError, prefix+": "+err.Error())
	}
}

func apiStoredDomain(in StoredDomain) apitypes.StoredDomain {
	return apitypes.StoredDomain{
		ID:                   apitypes.FormatAppID(in.ID),
		Hostname:             in.Hostname,
		Kind:                 in.Kind,
		Source:               in.Source,
		DNSState:             in.DNSState,
		RoutingState:         in.RoutingState,
		DNSStatusMessage:     in.DNSStatusMessage,
		RoutingStatusMessage: in.RoutingStatusMessage,
		LastCheckedAt:        in.LastCheckedAt,
		SiteID:               apitypes.FormatAppID(in.SiteID),
		SiteName:             in.SiteName,
		ParentDomainID:       apitypes.FormatAppID(in.ParentDomainID),
		ParentHostname:       in.ParentHostname,
		IsPrimary:            in.IsPrimary,
		CreatedAt:            in.CreatedAt,
		UpdatedAt:            in.UpdatedAt,
	}
}
