package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"pressluft/internal/orchestrator"
	"pressluft/internal/provider"
	"pressluft/internal/server/profiles"
)

type serversHandler struct {
	providerStore *provider.Store
	serverStore   *ServerStore
	jobStore      *orchestrator.Store
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
		// Check for nested paths like /api/servers/{id}/jobs
		parts := strings.Split(tail, "/")
		if len(parts) == 0 {
			http.NotFound(w, r)
			return
		}

		serverID, err := strconv.ParseInt(parts[0], 10, 64)
		if err != nil || serverID <= 0 {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			sh.routeServerByID(w, r, serverID)
			return
		}

		// Handle nested routes
		if len(parts) == 2 && parts[1] == "jobs" {
			if r.Method != http.MethodGet {
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
				return
			}
			sh.handleListServerJobs(w, r, serverID)
			return
		}

		http.NotFound(w, r)
	}
}

func (sh *serversHandler) routeServerByID(w http.ResponseWriter, r *http.Request, serverID int64) {
	switch r.Method {
	case http.MethodGet:
		sh.handleGetServer(w, r, serverID)
	case http.MethodDelete:
		sh.handleDeleteServer(w, r, serverID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (sh *serversHandler) handleGetServer(w http.ResponseWriter, r *http.Request, serverID int64) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, server)
}

func (sh *serversHandler) handleDeleteServer(w http.ResponseWriter, r *http.Request, serverID int64) {
	if err := sh.serverStore.Delete(r.Context(), serverID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (sh *serversHandler) handleListServerJobs(w http.ResponseWriter, r *http.Request, serverID int64) {
	jobs, err := sh.jobStore.ListJobsByServer(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, jobs)
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

	profile, ok := profiles.Get(req.ProfileKey)
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported profile_key: "+req.ProfileKey)
		return
	}

	storedProvider, err := sh.providerStore.GetByID(r.Context(), req.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	if _, ok := provider.GetServerProvider(storedProvider.Type); !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}

	// Create server record in pending state
	// Image is derived from the profile, not user input
	serverID, err := sh.serverStore.Create(r.Context(), CreateServerNodeInput{
		ProviderID:   storedProvider.ID,
		ProviderType: storedProvider.Type,
		Name:         req.Name,
		Location:     req.Location,
		ServerType:   req.ServerType,
		Image:        profile.Image,
		ProfileKey:   req.ProfileKey,
		Status:       "pending",
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to create server record: "+err.Error())
		return
	}

	// Create a provisioning job instead of calling provider directly
	job, err := sh.jobStore.CreateJob(r.Context(), orchestrator.CreateJobInput{
		Kind:     "provision_server",
		ServerID: serverID,
		Payload:  "",
	})
	if err != nil {
		// Rollback: mark server as failed since we couldn't create the job
		_ = sh.serverStore.UpdateStatus(r.Context(), serverID, "failed")
		respondError(w, http.StatusInternalServerError, "failed to create provisioning job: "+err.Error())
		return
	}

	// Emit initial job event
	_, _ = sh.jobStore.AppendEvent(r.Context(), job.ID, orchestrator.CreateEventInput{
		EventType: "job_created",
		Level:     "info",
		Status:    string(job.Status),
		Message:   "Server provisioning job queued",
	})

	respondJSON(w, http.StatusAccepted, map[string]any{
		"server_id": serverID,
		"job_id":    job.ID,
		"status":    "pending",
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
	if strings.TrimSpace(req.ProfileKey) == "" {
		return fmt.Errorf("profile_key is required")
	}
	return nil
}
