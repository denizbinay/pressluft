package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pressluft/internal/provider"
	"pressluft/internal/server/profiles"
)

type serversHandler struct {
	providerStore *provider.Store
	serverStore   *ServerStore
}

func (sh *serversHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/servers" {
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

func (sh *serversHandler) routeWithPath(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/servers/")

	switch tail {
	case "catalog":
		sh.handleCatalog(w, r)
	case "profiles":
		sh.handleProfiles(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (sh *serversHandler) handleList(w http.ResponseWriter, r *http.Request) {
	servers, err := sh.serverStore.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list servers: "+err.Error())
		return
	}
	if servers == nil {
		servers = []StoredServer{}
	}
	respondJSON(w, http.StatusOK, servers)
}

func (sh *serversHandler) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, profiles.All())
}

func (sh *serversHandler) handleCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providerID, err := parseProviderIDQuery(r)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	storedProvider, err := sh.providerStore.GetByID(r.Context(), providerID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}

	catalog, err := serverProvider.ListServerCatalog(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider server catalog: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, map[string]any{
		"catalog":  catalog,
		"profiles": profiles.All(),
	})
}

type createServerRequest struct {
	ProviderID int64  `json:"provider_id"`
	Name       string `json:"name"`
	Location   string `json:"location"`
	ServerType string `json:"server_type"`
	Image      string `json:"image"`
	ProfileKey string `json:"profile_key"`
}

func (sh *serversHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := validateCreateServerHTTPRequest(req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	if _, ok := profiles.Get(req.ProfileKey); !ok {
		respondError(w, http.StatusBadRequest, "unsupported profile_key: "+req.ProfileKey)
		return
	}

	storedProvider, err := sh.providerStore.GetByID(r.Context(), req.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}

	serverID, err := sh.serverStore.Create(r.Context(), CreateServerNodeInput{
		ProviderID:   storedProvider.ID,
		ProviderType: storedProvider.Type,
		Name:         req.Name,
		Location:     req.Location,
		ServerType:   req.ServerType,
		Image:        req.Image,
		ProfileKey:   req.ProfileKey,
		Status:       "pending",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create server record: "+err.Error())
		return
	}

	result, createErr := serverProvider.CreateServer(r.Context(), storedProvider.APIToken, provider.CreateServerRequest{
		Name:       req.Name,
		Location:   req.Location,
		ServerType: req.ServerType,
		Image:      req.Image,
		ProfileKey: req.ProfileKey,
		Labels: map[string]string{
			"pressluft_profile": req.ProfileKey,
		},
	})
	if createErr != nil {
		_ = sh.serverStore.UpdateProvisioning(r.Context(), serverID, "", "", "", "failed")
		respondError(w, http.StatusBadGateway, "server provisioning failed: "+createErr.Error())
		return
	}

	status := normalizeProvisionStatus(result.Status)
	if err := sh.serverStore.UpdateProvisioning(
		r.Context(),
		serverID,
		result.ProviderServerID,
		result.ActionID,
		result.Status,
		status,
	); err != nil {
		respondError(w, http.StatusInternalServerError, "failed to update server provisioning state: "+err.Error())
		return
	}

	respondJSON(w, http.StatusAccepted, map[string]any{
		"id":                 serverID,
		"status":             status,
		"provider_server_id": result.ProviderServerID,
		"action_id":          result.ActionID,
		"action_status":      result.Status,
	})
}

func parseProviderIDQuery(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("provider_id"))
	if raw == "" {
		return 0, fmt.Errorf("provider_id is required")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("provider_id must be a positive integer")
	}
	return id, nil
}

func validateCreateServerHTTPRequest(req createServerRequest) error {
	if req.ProviderID <= 0 {
		return fmt.Errorf("provider_id must be a positive integer")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Location) == "" {
		return fmt.Errorf("location is required")
	}
	if strings.TrimSpace(req.ServerType) == "" {
		return fmt.Errorf("server_type is required")
	}
	if strings.TrimSpace(req.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if strings.TrimSpace(req.ProfileKey) == "" {
		return fmt.Errorf("profile_key is required")
	}
	return nil
}

func normalizeProvisionStatus(actionStatus string) string {
	status := strings.ToLower(strings.TrimSpace(actionStatus))
	switch status {
	case "success":
		return "ready"
	case "error":
		return "failed"
	case "running":
		return "provisioning"
	default:
		return "provisioning"
	}
}
