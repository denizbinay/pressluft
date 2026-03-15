package server

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/controlplane/apitypes"
	"pressluft/internal/shared/ws"

	"github.com/google/uuid"
)

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
		result[apitypes.FormatAppID(server.ID)] = storedAgentInfo(server)
	}

	if sh.hub != nil {
		for serverID, info := range sh.hub.GetAllAgentInfo() {
			result[serverID] = info
		}
	}

	respondJSON(w, http.StatusOK, result)
}

// handleAgentStatus returns real-time agent connection status and metrics.
func (sh *serversHandler) handleAgentStatus(w http.ResponseWriter, r *http.Request, serverID string) {
	// First verify the server exists
	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	if sh.hub != nil {
		if _, ok := sh.hub.Get(server.ID); ok {
			info := sh.hub.GetAgentInfo(server.ID)
			respondJSON(w, http.StatusOK, info)
			return
		}
	}

	respondJSON(w, http.StatusOK, storedAgentInfo(*server))
}

// handleListServices returns the list of running services on the server.
// This requires an active agent connection to fetch real-time data.
func (sh *serversHandler) handleListServices(w http.ResponseWriter, r *http.Request, serverID string) {
	// Verify server exists
	_, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	// Check if agent is connected
	if sh.hub == nil {
		respondJSON(w, http.StatusOK, apitypes.ServicesResponse{
			ServerID:       apitypes.FormatAppID(serverID),
			AgentConnected: false,
			Services:       []agentcommand.Service{},
		})
		return
	}

	server, err := sh.serverStore.GetByID(r.Context(), serverID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	info := sh.hub.GetAgentInfo(server.ID)
	if !info.Connected {
		respondJSON(w, http.StatusOK, apitypes.ServicesResponse{
			ServerID:       apitypes.FormatAppID(serverID),
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

	result, err := sh.hub.SendCommandAndWait(ctx, server.ID, cmd)
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
		ServerID:       apitypes.FormatAppID(serverID),
		AgentConnected: true,
		Services:       payload.Services,
	})
}
