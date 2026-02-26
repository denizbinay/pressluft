package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/orchestrator"
)

type jobsHandler struct {
	store         *orchestrator.Store
	activityStore *activity.Store
}

type createJobRequest struct {
	Kind     string          `json:"kind"`
	ServerID int64           `json:"server_id"`
	Payload  json.RawMessage `json:"payload"`
}

type rebuildServerPayload struct {
	ServerName  string `json:"server_name"`
	ServerImage string `json:"server_image"`
}

type resizeServerPayload struct {
	ServerType  string `json:"server_type"`
	UpgradeDisk *bool  `json:"upgrade_disk"`
}

type updateFirewallsPayload struct {
	Firewalls []string `json:"firewalls"`
}

type manageVolumePayload struct {
	VolumeName string `json:"volume_name"`
	SizeGB     int    `json:"size_gb"`
	Location   string `json:"location"`
	State      string `json:"state"`
	Automount  *bool  `json:"automount"`
}

func (jh *jobsHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/jobs" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		jh.handleList(w, r)
	case http.MethodPost:
		jh.handleCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (jh *jobsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	jobs, err := jh.store.ListAllJobs(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, jobs)
}

func (jh *jobsHandler) routeWithID(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/jobs/")
	parts := strings.Split(strings.Trim(tail, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}

	jobID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || jobID <= 0 {
		respondError(w, http.StatusBadRequest, "job id must be a positive integer")
		return
	}

	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jh.handleGet(w, r, jobID)
		return
	}

	if len(parts) == 2 && parts[1] == "events" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jh.handleEventStream(w, r, jobID)
		return
	}

	// /api/jobs/{id}/events/history - get all events as JSON (for completed jobs)
	if len(parts) == 3 && parts[1] == "events" && parts[2] == "history" {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jh.handleEventHistory(w, r, jobID)
		return
	}

	http.NotFound(w, r)
}

func (jh *jobsHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(req.Kind) == "" {
		req.Kind = "provision_server"
	}
	payload, err := validateJobPayload(req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	job, err := jh.store.CreateJob(r.Context(), orchestrator.CreateJobInput{
		Kind:     req.Kind,
		ServerID: req.ServerID,
		Payload:  payload,
	})
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to create job: "+err.Error())
		return
	}

	_, _ = jh.store.AppendEvent(r.Context(), job.ID, orchestrator.CreateEventInput{
		EventType: "job_created",
		Level:     "info",
		Status:    string(job.Status),
		Message:   "Job accepted and queued",
	})

	// Emit activity for job creation
	if jh.activityStore != nil {
		input := activity.EmitInput{
			EventType:    activity.EventJobCreated,
			Category:     activity.CategoryJob,
			Level:        activity.LevelInfo,
			ResourceType: activity.ResourceJob,
			ResourceID:   job.ID,
			ActorType:    activity.ActorUser,
			Title:        fmt.Sprintf("%s job queued", jobKindLabel(req.Kind)),
		}
		if req.ServerID > 0 {
			input.ParentResourceType = activity.ResourceServer
			input.ParentResourceID = req.ServerID
		}
		_, _ = jh.activityStore.Emit(r.Context(), input)
	}

	respondJSON(w, http.StatusAccepted, job)
}

// jobKindLabel returns a human-readable label for a job kind.
func jobKindLabel(kind string) string {
	labels := map[string]string{
		"provision_server": "Server provisioning",
		"delete_server":    "Server deletion",
		"rebuild_server":   "Server rebuild",
		"resize_server":    "Server resize",
		"update_firewalls": "Firewall update",
		"manage_volume":    "Volume management",
	}
	if label, ok := labels[kind]; ok {
		return label
	}
	return kind
}

func validateJobPayload(req createJobRequest) (string, error) {
	payload := strings.TrimSpace(string(req.Payload))
	payloadBytes := bytes.TrimSpace(req.Payload)
	if len(payloadBytes) == 0 || string(payloadBytes) == "null" {
		payloadBytes = []byte("{}")
	}

	switch req.Kind {
	case "rebuild_server":
		if req.ServerID <= 0 {
			return "", fmt.Errorf("server_id is required for rebuild_server job")
		}
		if payload == "" || payload == "null" {
			return "", nil
		}
		var parsed rebuildServerPayload
		if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
			return "", fmt.Errorf("invalid rebuild_server payload: %w", err)
		}
	case "resize_server":
		if req.ServerID <= 0 {
			return "", fmt.Errorf("server_id is required for resize_server job")
		}
		var parsed resizeServerPayload
		if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
			return "", fmt.Errorf("invalid resize_server payload: %w", err)
		}
		if strings.TrimSpace(parsed.ServerType) == "" {
			return "", fmt.Errorf("server_type is required for resize_server job")
		}
		if parsed.UpgradeDisk == nil {
			return "", fmt.Errorf("upgrade_disk is required for resize_server job")
		}
	case "update_firewalls":
		if req.ServerID <= 0 {
			return "", fmt.Errorf("server_id is required for update_firewalls job")
		}
		var parsed updateFirewallsPayload
		if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
			return "", fmt.Errorf("invalid update_firewalls payload: %w", err)
		}
		firewalls := make([]string, 0, len(parsed.Firewalls))
		for _, fw := range parsed.Firewalls {
			fw = strings.TrimSpace(fw)
			if fw != "" {
				firewalls = append(firewalls, fw)
			}
		}
		if len(firewalls) == 0 {
			return "", fmt.Errorf("firewalls payload must contain at least one firewall")
		}
	case "manage_volume":
		if req.ServerID <= 0 {
			return "", fmt.Errorf("server_id is required for manage_volume job")
		}
		var parsed manageVolumePayload
		if err := json.Unmarshal(payloadBytes, &parsed); err != nil {
			return "", fmt.Errorf("invalid manage_volume payload: %w", err)
		}
		volumeName := strings.TrimSpace(parsed.VolumeName)
		state := strings.TrimSpace(parsed.State)
		if volumeName == "" {
			return "", fmt.Errorf("volume_name is required for manage_volume job")
		}
		if state != "present" && state != "absent" {
			return "", fmt.Errorf("state must be present or absent for manage_volume job")
		}
		if state == "present" {
			if parsed.Automount == nil {
				return "", fmt.Errorf("automount is required for manage_volume job when state=present")
			}
			if parsed.SizeGB <= 0 {
				return "", fmt.Errorf("size_gb is required for manage_volume job when state=present")
			}
		}
	}

	if payload == "null" {
		payload = ""
	}
	return payload, nil
}

func (jh *jobsHandler) handleGet(w http.ResponseWriter, r *http.Request, jobID int64) {
	job, err := jh.store.GetJob(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, job)
}

func (jh *jobsHandler) handleEventHistory(w http.ResponseWriter, r *http.Request, jobID int64) {
	events, err := jh.store.ListAllEvents(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, events)
}

func (jh *jobsHandler) handleEventStream(w http.ResponseWriter, r *http.Request, jobID int64) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	startSeq := int64(0)
	if raw := strings.TrimSpace(r.URL.Query().Get("since_seq")); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil && parsed >= 0 {
			startSeq = parsed
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send a comment to establish connection immediately
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	currentSeq := startSeq
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			events, err := jh.store.ListEvents(r.Context(), jobID, currentSeq, 100)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":%q}\n\n", err.Error())
				flusher.Flush()
				return
			}

			if len(events) == 0 {
				fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
				continue
			}

			for _, evt := range events {
				body, err := json.Marshal(evt)
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "id: %d\n", evt.Seq)
				fmt.Fprint(w, "event: job_event\n")
				fmt.Fprintf(w, "data: %s\n\n", body)
				currentSeq = evt.Seq
			}
			flusher.Flush()
		}
	}
}
