package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
)

type activityHandler struct {
	store *activity.Store
}

func (ah *activityHandler) route(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/activity" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ah.handleList(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (ah *activityHandler) routeWithID(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/activity/")
	parts := strings.Split(strings.Trim(tail, "/"), "/")
	if len(parts) == 0 || strings.TrimSpace(parts[0]) == "" {
		http.NotFound(w, r)
		return
	}

	// Handle special paths first
	switch parts[0] {
	case "stream":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ah.handleStream(w, r)
		return
	case "read-all":
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ah.handleMarkAllRead(w, r)
		return
	case "unread-count":
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ah.handleUnreadCount(w, r)
		return
	}

	activityID := strings.TrimSpace(parts[0])
	if activityID == "" {
		respondError(w, http.StatusBadRequest, "activity id is required")
		return
	}

	// /api/activity/{id}
	if len(parts) == 1 {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ah.handleGet(w, r, activityID)
		return
	}

	// /api/activity/{id}/read
	if len(parts) == 2 && parts[1] == "read" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		ah.handleMarkRead(w, r, activityID)
		return
	}

	http.NotFound(w, r)
}

func (ah *activityHandler) handleList(w http.ResponseWriter, r *http.Request) {
	filter := parseActivityFilter(r)

	activities, nextCursor, err := ah.store.List(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := apitypes.ActivityListResponse{
		Data:       apitypes.APIActivities(activities),
		NextCursor: nextCursor,
	}
	respondJSON(w, http.StatusOK, response)
}

func (ah *activityHandler) handleGet(w http.ResponseWriter, r *http.Request, id string) {
	act, err := ah.store.GetByID(r.Context(), id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apitypes.APIActivity(act))
}

func (ah *activityHandler) handleMarkRead(w http.ResponseWriter, r *http.Request, id string) {
	if err := ah.store.MarkRead(r.Context(), id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			respondError(w, http.StatusNotFound, err.Error())
			return
		}
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	act, err := ah.store.GetByID(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apitypes.APIActivity(act))
}

func (ah *activityHandler) handleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	filter := parseActivityFilter(r)

	if err := ah.store.MarkAllRead(r.Context(), filter); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, apitypes.StatusResponse{Status: "ok"})
}

func (ah *activityHandler) handleUnreadCount(w http.ResponseWriter, r *http.Request) {
	filter := parseActivityFilter(r)
	// Default to requiring attention for unread count
	if filter.RequiresAttention == nil {
		t := true
		filter.RequiresAttention = &t
	}

	count, err := ah.store.CountUnread(r.Context(), filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondJSON(w, http.StatusOK, apitypes.UnreadCountResponse{Count: count})
}

func (ah *activityHandler) handleStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, http.StatusInternalServerError, "streaming not supported")
		return
	}

	// Get starting point
	sinceID := strings.TrimSpace(r.URL.Query().Get("since_id"))

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Send a comment to establish connection immediately
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	currentID := sinceID
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			activities, err := ah.store.ListSince(r.Context(), currentID, 100)
			if err != nil {
				fmt.Fprintf(w, "event: error\ndata: {\"error\":%q}\n\n", err.Error())
				flusher.Flush()
				return
			}

			if len(activities) == 0 {
				fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
				continue
			}

			for _, act := range activities {
				body, err := json.Marshal(apitypes.APIActivity(act))
				if err != nil {
					continue
				}
				fmt.Fprintf(w, "id: %s\n", act.ID)
				fmt.Fprint(w, "event: activity\n")
				fmt.Fprintf(w, "data: %s\n\n", body)
				currentID = act.ID
			}
			flusher.Flush()
		}
	}
}

// handleServerActivity lists activity for a specific server (convenience endpoint)
func (ah *activityHandler) handleServerActivity(w http.ResponseWriter, r *http.Request, serverID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	filter := parseActivityFilter(r)

	activities, nextCursor, err := ah.store.ListForServer(r.Context(), serverID, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	response := apitypes.ActivityListResponse{
		Data:       apitypes.APIActivities(activities),
		NextCursor: nextCursor,
	}
	respondJSON(w, http.StatusOK, response)
}

func (ah *activityHandler) handleSiteActivity(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	filter := parseActivityFilter(r)
	activities, nextCursor, err := ah.store.ListForSite(r.Context(), siteID, filter)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, apitypes.ActivityListResponse{
		Data:       apitypes.APIActivities(activities),
		NextCursor: nextCursor,
	})
}

func parseActivityFilter(r *http.Request) activity.ListFilter {
	filter := activity.ListFilter{}

	// Cursor
	if raw := strings.TrimSpace(r.URL.Query().Get("cursor")); raw != "" {
		filter.Cursor = raw
	}

	// Limit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 {
			filter.Limit = parsed
		}
	}

	// Category
	if raw := strings.TrimSpace(r.URL.Query().Get("category")); raw != "" {
		filter.Category = activity.Category(raw)
	}

	// Resource type and ID
	if raw := strings.TrimSpace(r.URL.Query().Get("resource_type")); raw != "" {
		filter.ResourceType = activity.ResourceType(raw)
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("resource_id")); raw != "" {
		filter.ResourceID = raw
	}

	// Parent resource type and ID
	if raw := strings.TrimSpace(r.URL.Query().Get("parent_resource_type")); raw != "" {
		filter.ParentResourceType = activity.ResourceType(raw)
	}
	if raw := strings.TrimSpace(r.URL.Query().Get("parent_resource_id")); raw != "" {
		filter.ParentResourceID = raw
	}

	// Requires attention
	if raw := strings.TrimSpace(r.URL.Query().Get("requires_attention")); raw != "" {
		if raw == "true" || raw == "1" {
			t := true
			filter.RequiresAttention = &t
		} else if raw == "false" || raw == "0" {
			f := false
			filter.RequiresAttention = &f
		}
	}

	// Unread only
	if raw := strings.TrimSpace(r.URL.Query().Get("unread_only")); raw == "true" || raw == "1" {
		filter.UnreadOnly = true
	}

	return filter
}
