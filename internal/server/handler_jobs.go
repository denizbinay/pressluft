package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
	"pressluft/internal/orchestrator"
)

type jobsHandler struct {
	store         *orchestrator.Store
	serverStore   *ServerStore
	activityStore *activity.Store
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
	var req apitypes.CreateJobRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	slog.Default().Info("job action requested", "job_kind", req.Kind, "server_id", req.ServerID)
	payload, err := validateJobPayload(req)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	var job orchestrator.Job
	if req.ServerID > 0 && jh.serverStore != nil {
		dispatchPolicy, ok := orchestrator.DispatchPolicyForKind(req.Kind)
		if !ok {
			respondError(w, http.StatusBadRequest, "unsupported job kind: "+req.Kind)
			return
		}
		if dispatchPolicy.QueueServer {
			_, job, err = jh.serverStore.QueueServerJob(r.Context(), QueueServerJobInput{
				ServerID: req.ServerID,
				Kind:     req.Kind,
				Payload:  payload,
			})
		} else {
			job, err = jh.store.CreateJob(r.Context(), orchestrator.CreateJobInput{
				Kind:     req.Kind,
				ServerID: req.ServerID,
				Payload:  payload,
			})
		}
	} else {
		job, err = jh.store.CreateJob(r.Context(), orchestrator.CreateJobInput{
			Kind:     req.Kind,
			ServerID: req.ServerID,
			Payload:  payload,
		})
	}
	if err != nil {
		statusCode := http.StatusBadRequest
		if err == ErrServerDeleting || err == ErrServerDeleted || strings.Contains(err.Error(), ErrServerActionConflict.Error()) {
			statusCode = http.StatusConflict
		}
		respondError(w, statusCode, "failed to create job: "+err.Error())
		return
	}

	_, _ = jh.store.AppendEvent(r.Context(), job.ID, orchestrator.CreateEventInput{
		EventType: orchestrator.JobEventTypeCreated,
		Level:     "info",
		Status:    string(job.Status),
		Message:   "Job accepted and queued",
	})

	// Emit activity for job creation
	if jh.activityStore != nil {
		actorType, actorID := activityActorFromRequest(r)
		input := activity.EmitInput{
			EventType:    activity.EventJobCreated,
			Category:     activity.CategoryJob,
			Level:        activity.LevelInfo,
			ResourceType: activity.ResourceJob,
			ResourceID:   job.ID,
			ActorType:    actorType,
			ActorID:      actorID,
			Title:        fmt.Sprintf("%s job queued", orchestrator.JobKindLabel(req.Kind)),
		}
		if req.ServerID > 0 {
			input.ParentResourceType = activity.ResourceServer
			input.ParentResourceID = req.ServerID
		}
		_, _ = jh.activityStore.Emit(r.Context(), input)
	}

	respondJSON(w, http.StatusAccepted, job)
	slog.Default().Info("job action queued", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID, "job_status", job.Status)
}

// jobKindLabel returns a human-readable label for a job kind.
func jobKindLabel(kind string) string {
	return orchestrator.JobKindLabel(kind)
}

func validateJobPayload(req apitypes.CreateJobRequest) (string, error) {
	return orchestrator.ValidatePayload(req.Kind, req.Payload, req.ServerID)
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
