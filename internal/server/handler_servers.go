package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/agentcommand"
	"pressluft/internal/apitypes"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/server/profiles"
	"pressluft/internal/ws"

	"github.com/google/uuid"
)

type serversHandler struct {
	providerStore   *provider.Store
	serverStore     *ServerStore
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
			}
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
	respondJSON(w, http.StatusOK, apiStoredServer(*server))
}

func (sh *serversHandler) handleDeleteServer(w http.ResponseWriter, r *http.Request, serverID int64) {
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
		ServerID:    serverID,
		JobID:       job.ID,
		Status:      platform.ServerStatusDeleting,
		JobStatus:   job.Status,
		Async:       true,
		Description: "Server deletion queued",
	})
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

	respondJSON(w, http.StatusOK, apitypes.ServerCatalogResponse{
		Catalog:  *catalog,
		Profiles: profiles.All(),
	})
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
		ServerID: serverID,
		JobID:    job.ID,
		Status:   platform.ServerStatusPending,
	})
	slog.Default().Info("server action queued", "action", "create_server", "server_id", serverID, "job_id", job.ID, "server_status", platform.ServerStatusPending)
}

func (sh *serversHandler) handleRebuildOptions(w http.ResponseWriter, r *http.Request, serverID int64) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support server provisioning: "+storedProvider.Type)
		return
	}
	imageProvider, ok := provider.GetServerImageProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support image listing: "+storedProvider.Type)
		return
	}

	catalog, err := serverProvider.ListServerCatalog(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider server catalog: "+err.Error())
		return
	}
	architecture, err := resolveServerArchitecture(server.ServerType, catalog)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	images, err := imageProvider.ListServerImages(r.Context(), storedProvider.APIToken, architecture)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider images: "+err.Error())
		return
	}
	if images == nil {
		images = []provider.ServerImageOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.RebuildOptionsResponse{
		ServerID:     server.ID,
		ServerType:   server.ServerType,
		Architecture: architecture,
		Images:       images,
	})
}

func (sh *serversHandler) handleResizeOptions(w http.ResponseWriter, r *http.Request, serverID int64) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
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
	architecture, err := resolveServerArchitecture(server.ServerType, catalog)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}

	serverTypes := filterServerTypes(catalog.ServerTypes, server.Location, architecture)
	if serverTypes == nil {
		serverTypes = []provider.ServerTypeOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.ResizeOptionsResponse{
		ServerID:     server.ID,
		Location:     server.Location,
		ServerType:   server.ServerType,
		Architecture: architecture,
		ServerTypes:  serverTypes,
	})
}

func (sh *serversHandler) handleServerFirewalls(w http.ResponseWriter, r *http.Request, serverID int64) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	firewallProvider, ok := provider.GetFirewallProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support firewall listing: "+storedProvider.Type)
		return
	}

	firewalls, err := firewallProvider.ListFirewalls(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider firewalls: "+err.Error())
		return
	}
	if firewalls == nil {
		firewalls = []provider.FirewallOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.FirewallsResponse{
		ServerID:  server.ID,
		Firewalls: firewalls,
	})
}

func (sh *serversHandler) handleServerVolumes(w http.ResponseWriter, r *http.Request, serverID int64) {
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	storedProvider, err := sh.providerStore.GetByID(r.Context(), server.ProviderID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	volumeProvider, ok := provider.GetVolumeProvider(storedProvider.Type)
	if !ok {
		respondError(w, http.StatusBadRequest, "provider does not support volume listing: "+storedProvider.Type)
		return
	}

	volumes, err := volumeProvider.ListVolumes(r.Context(), storedProvider.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch provider volumes: "+err.Error())
		return
	}
	if volumes == nil {
		volumes = []provider.VolumeOption{}
	}

	respondJSON(w, http.StatusOK, apitypes.VolumesResponse{
		ServerID: server.ID,
		Volumes:  volumes,
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

func validateCreateServerHTTPRequest(req apitypes.CreateServerRequest) error {
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

func resolveServerArchitecture(serverType string, catalog *provider.ServerCatalog) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("provider catalog is empty")
	}
	for _, option := range catalog.ServerTypes {
		if option.Name == serverType {
			arch := strings.TrimSpace(option.Architecture)
			if arch == "" {
				return "", fmt.Errorf("architecture not available for server type %q", serverType)
			}
			return arch, nil
		}
	}
	return "", fmt.Errorf("server type %q not found in provider catalog", serverType)
}

func filterServerTypes(options []provider.ServerTypeOption, location, architecture string) []provider.ServerTypeOption {
	if len(options) == 0 {
		return nil
	}
	location = strings.TrimSpace(location)
	architecture = strings.TrimSpace(architecture)
	filtered := make([]provider.ServerTypeOption, 0, len(options))
	for _, option := range options {
		if option.Architecture != architecture {
			continue
		}
		if location != "" && !containsString(option.AvailableAt, location) {
			continue
		}
		filtered = append(filtered, option)
	}
	return filtered
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

// handleAllAgentStatus returns agent status for all connected servers.
// GET /api/servers/agents
func (sh *serversHandler) handleAllAgentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	servers, err := sh.serverStore.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	result := make(apitypes.AgentStatusMapResponse, len(servers))
	for _, server := range servers {
		result[server.ID] = storedAgentInfo(server)
	}

	if sh.hub != nil {
		for serverID, info := range sh.hub.GetAllAgentInfo() {
			result[serverID] = info
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// handleAgentStatus returns real-time agent connection status and metrics.
func (sh *serversHandler) handleAgentStatus(w http.ResponseWriter, r *http.Request, serverID int64) {
	// First verify the server exists
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	if sh.hub != nil {
		if _, ok := sh.hub.Get(serverID); ok {
			info := sh.hub.GetAgentInfo(serverID)
			respondJSON(w, http.StatusOK, info)
			return
		}
	}

	respondJSON(w, http.StatusOK, storedAgentInfo(*server))
}

// handleListServices returns the list of running services on the server.
// This requires an active agent connection to fetch real-time data.
func (sh *serversHandler) handleListServices(w http.ResponseWriter, r *http.Request, serverID int64) {
	// Verify server exists
	_, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// Check if agent is connected
	if sh.hub == nil {
		respondJSON(w, http.StatusOK, apitypes.ServicesResponse{
			ServerID:       serverID,
			AgentConnected: false,
			Services:       []agentcommand.Service{},
		})
		return
	}

	info := sh.hub.GetAgentInfo(serverID)
	if !info.Connected {
		respondJSON(w, http.StatusOK, apitypes.ServicesResponse{
			ServerID:       serverID,
			AgentConnected: false,
			Services:       []agentcommand.Service{},
		})
		return
	}

	timeout := agentcommand.Timeout(agentcommand.TypeListServices)
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(r.Context(), timeout)
	defer cancel()

	cmd := ws.Command{
		ID:   uuid.NewString(),
		Type: agentcommand.TypeListServices,
	}

	result, err := sh.hub.SendCommandAndWait(ctx, serverID, cmd)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to fetch services: "+err.Error())
		return
	}

	if !result.Success {
		respondError(w, http.StatusBadGateway, "failed to fetch services: "+result.Error)
		return
	}

	var payload struct {
		Services []agentcommand.Service `json:"services"`
	}
	if len(result.Payload) > 0 {
		if err := json.Unmarshal(result.Payload, &payload); err != nil {
			respondError(w, http.StatusBadGateway, "invalid service response")
			return
		}
	}

	respondJSON(w, http.StatusOK, apitypes.ServicesResponse{
		ServerID:       serverID,
		AgentConnected: true,
		Services:       payload.Services,
	})
}

func storedAgentInfo(server StoredServer) ws.AgentInfo {
	status := platform.NodeStatusUnknown
	if server.NodeStatus != platform.NodeStatus("") {
		status = server.NodeStatus
	}
	info := ws.AgentInfo{
		Connected: status == platform.NodeStatusOnline,
		Status:    status,
		Version:   server.NodeVersion,
	}
	if server.NodeLastSeen != "" {
		if parsed, err := time.Parse(time.RFC3339, server.NodeLastSeen); err == nil {
			info.LastSeen = parsed
		}
	}
	return info
}

func apiStoredServer(in StoredServer) apitypes.StoredServer {
	return apitypes.StoredServer{
		ID:               in.ID,
		ProviderID:       in.ProviderID,
		ProviderType:     in.ProviderType,
		ProviderServerID: in.ProviderServerID,
		IPv4:             in.IPv4,
		IPv6:             in.IPv6,
		Name:             in.Name,
		Location:         in.Location,
		ServerType:       in.ServerType,
		Image:            in.Image,
		ProfileKey:       in.ProfileKey,
		Status:           in.Status,
		SetupState:       in.SetupState,
		SetupLastError:   in.SetupLastError,
		ActionID:         in.ActionID,
		ActionStatus:     in.ActionStatus,
		HasKey:           in.HasKey,
		NodeStatus:       in.NodeStatus,
		NodeLastSeen:     in.NodeLastSeen,
		NodeVersion:      in.NodeVersion,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
	}
}
