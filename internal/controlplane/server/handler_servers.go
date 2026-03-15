package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/controlplane/apitypes"
	"pressluft/internal/controlplane/server/profiles"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/ws"
)

type serversHandler struct {
	providerStore   *provider.Store
	serverStore     *ServerStore
	siteStore       *SiteStore
	jobStore        *orchestrator.Store
	activityStore   *activity.Store
	activityHandler *activityHandler
	hub             *ws.Hub
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
	case "agents":
		sh.handleAllAgentStatus(w, r)
	default:
		// Check for nested paths like /api/servers/{id}/jobs
		parts := strings.Split(tail, "/")
		if len(parts) == 0 {
			http.NotFound(w, r)
			return
		}

		serverID, err := apitypes.ParseAppID(parts[0])
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if len(parts) == 1 {
			sh.routeServerByID(w, r, serverID)
			return
		}

		// Handle nested routes
		if len(parts) == 2 {
			switch parts[1] {
			case "jobs":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleListServerJobs(w, r, serverID)
				return
			case "rebuild-options":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleRebuildOptions(w, r, serverID)
				return
			case "resize-options":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleResizeOptions(w, r, serverID)
				return
			case "firewalls":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleServerFirewalls(w, r, serverID)
				return
			case "volumes":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleServerVolumes(w, r, serverID)
				return
			case "activity":
				if sh.activityHandler != nil {
					sh.activityHandler.handleServerActivity(w, r, serverID)
					return
				}
			case "agent-status":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleAgentStatus(w, r, serverID)
				return
			case "services":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleListServices(w, r, serverID)
				return
			case "sites":
				if r.Method != http.MethodGet {
					http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
					return
				}
				sh.handleListSites(w, r, serverID)
				return
			}
		}

		http.NotFound(w, r)
	}
}

func (sh *serversHandler) routeServerByID(w http.ResponseWriter, r *http.Request, serverID string) {
	switch r.Method {
	case http.MethodGet:
		sh.handleGetServer(w, r, serverID)
	case http.MethodDelete:
		sh.handleDeleteServer(w, r, serverID)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (sh *serversHandler) handleGetServer(w http.ResponseWriter, r *http.Request, serverID string) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apiStoredServer(*server))
}

func (sh *serversHandler) handleDeleteServer(w http.ResponseWriter, r *http.Request, serverID string) {
	slog.Default().Info("server action requested", "action", "delete_server", "server_id", serverID)
	serverRecord, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	_, job, err := sh.serverStore.QueueServerJob(r.Context(), QueueServerJobInput{
		ServerID: serverID,
		Kind:     string(orchestrator.JobKindDeleteServer),
		Payload:  fmt.Sprintf(`{"server_name":%q}`, serverRecord.Name),
	})
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "not found"):
			respondError(w, http.StatusNotFound, err.Error())
		case err == ErrServerDeleting || err == ErrServerDeleted || strings.Contains(err.Error(), ErrServerActionConflict.Error()):
			respondError(w, http.StatusConflict, err.Error())
		default:
			respondError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	_, _ = sh.jobStore.AppendEvent(r.Context(), job.ID, orchestrator.CreateEventInput{
		EventType: "job_created",
		Level:     "info",
		Status:    string(job.Status),
		Message:   "Server deletion job queued",
	})

	if sh.activityStore != nil {
		title := fmt.Sprintf("Server '%s' deletion requested", serverRecord.Name)
		actorType, actorID := activityActorFromRequest(r)
		_, _ = sh.activityStore.Emit(r.Context(), activity.EmitInput{
			EventType:    activity.EventServerStatusChanged,
			Category:     activity.CategoryServer,
			Level:        activity.LevelInfo,
			ResourceType: activity.ResourceServer,
			ResourceID:   serverID,
			ActorType:    actorType,
			ActorID:      actorID,
			Title:        title,
			Message:      "Delete runs asynchronously through the orchestrator until provider-side removal succeeds or fails.",
		})
	}
	slog.Default().Info("server action queued", "action", "delete_server", "server_id", serverID, "job_id", job.ID, "server_status", platform.ServerStatusDeleting)

	respondJSON(w, http.StatusAccepted, apitypes.DeleteServerResponse{
		ServerID:    apitypes.FormatAppID(serverID),
		JobID:       job.ID,
		Status:      platform.ServerStatusDeleting,
		JobStatus:   job.Status,
		Async:       true,
		Description: "Server deletion queued",
	})
}

func (sh *serversHandler) handleListServerJobs(w http.ResponseWriter, r *http.Request, serverID string) {
	jobs, err := sh.jobStore.ListJobsByServer(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apitypes.APIJobs(jobs))
}

func (sh *serversHandler) handleList(w http.ResponseWriter, r *http.Request) {
	servers, err := sh.serverStore.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list servers: "+err.Error())
		return
	}
	if servers == nil {
		respondJSON(w, http.StatusOK, []apitypes.StoredServer{})
		return
	}
	payload := make([]apitypes.StoredServer, 0, len(servers))
	for _, srv := range servers {
		payload = append(payload, apiStoredServer(srv))
	}
	respondJSON(w, http.StatusOK, payload)
}

func (sh *serversHandler) handleProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, profiles.All())
}

func (sh *serversHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req apitypes.CreateServerRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	slog.Default().Info("server action requested", "action", "create_server", "provider_id", req.ProviderID, "server_name", req.Name, "profile_key", req.ProfileKey)

	if err := validateCreateServerHTTPRequest(req); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	profile, ok := profiles.Get(req.ProfileKey)
	if !ok {
		respondError(w, http.StatusBadRequest, "unsupported profile_key: "+req.ProfileKey)
		return
	}
	if !profile.Selectable() {
		reason := strings.TrimSpace(profile.SupportReason)
		if reason == "" {
			reason = "profile is not selectable in the current platform contract"
		}
		respondError(w, http.StatusBadRequest, reason)
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
		Status:       platform.ServerStatusPending,
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
		_ = sh.serverStore.UpdateStatus(r.Context(), serverID, platform.ServerStatusFailed)
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

	// Emit activity for server creation
	if sh.activityStore != nil {
		actorType, actorID := activityActorFromRequest(r)
		_, _ = sh.activityStore.Emit(r.Context(), activity.EmitInput{
			EventType:    activity.EventServerCreated,
			Category:     activity.CategoryServer,
			Level:        activity.LevelInfo,
			ResourceType: activity.ResourceServer,
			ResourceID:   serverID,
			ActorType:    actorType,
			ActorID:      actorID,
			Title:        fmt.Sprintf("Server '%s' created", req.Name),
		})
	}

	respondJSON(w, http.StatusAccepted, apitypes.CreateServerResponse{
		ServerID: apitypes.FormatAppID(serverID),
		JobID:    job.ID,
		Status:   platform.ServerStatusPending,
	})
	slog.Default().Info("server action queued", "action", "create_server", "server_id", serverID, "job_id", job.ID, "server_status", platform.ServerStatusPending)
}

func (sh *serversHandler) handleListSites(w http.ResponseWriter, r *http.Request, serverID string) {
	if _, err := sh.serverStore.GetByID(r.Context(), serverID); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	if sh.siteStore == nil {
		respondJSON(w, http.StatusOK, []apitypes.StoredSite{})
		return
	}
	sites, err := sh.siteStore.ListByServer(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	payload := make([]apitypes.StoredSite, 0, len(sites))
	for _, site := range sites {
		payload = append(payload, apiStoredSite(site))
	}
	respondJSON(w, http.StatusOK, payload)
}
