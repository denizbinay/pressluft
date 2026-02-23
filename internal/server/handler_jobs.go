package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/orchestrator"
)

type jobsHandler struct {
	store *orchestrator.Store
}

type createJobRequest struct {
	Kind     string `json:"kind"`
	ServerID int64  `json:"server_id"`
}

func (jh *jobsHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/jobs" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodPost:
		jh.handleCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
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

	job, err := jh.store.CreateJob(r.Context(), orchestrator.CreateJobInput{
		Kind:     req.Kind,
		ServerID: req.ServerID,
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

	respondJSON(w, http.StatusAccepted, job)
}

func (jh *jobsHandler) handleGet(w http.ResponseWriter, r *http.Request, jobID int64) {
	job, err := jh.store.GetJob(r.Context(), jobID)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, job)
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
