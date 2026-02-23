package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"pressluft/internal/provider"
)

type providerHandler struct {
	store *provider.Store
}

// route dispatches /api/providers based on HTTP method.
func (ph *providerHandler) route(w http.ResponseWriter, r *http.Request) {
	// Exact match only — requests with trailing path segments go to routeWithID.
	if r.URL.Path != "/api/providers" {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodGet:
		ph.handleList(w, r)
	case http.MethodPost:
		ph.handleCreate(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// routeWithID dispatches /api/providers/{id} and /api/providers/validate|types.
func (ph *providerHandler) routeWithID(w http.ResponseWriter, r *http.Request) {
	tail := strings.TrimPrefix(r.URL.Path, "/api/providers/")

	switch tail {
	case "validate":
		ph.handleValidate(w, r)
		return
	case "types":
		ph.handleTypes(w, r)
		return
	}

	id, err := strconv.ParseInt(tail, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid provider id")
		return
	}

	switch r.Method {
	case http.MethodDelete:
		ph.handleDelete(w, r, id)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// --- handlers ---------------------------------------------------------------

type createProviderRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	APIToken string `json:"api_token"`
}

func (ph *providerHandler) handleCreate(w http.ResponseWriter, r *http.Request) {
	var req createProviderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Type == "" || req.Name == "" || req.APIToken == "" {
		respondError(w, http.StatusBadRequest, "type, name, and api_token are required")
		return
	}

	p := provider.Get(req.Type)
	if p == nil {
		respondError(w, http.StatusBadRequest, "unsupported provider type: "+req.Type)
		return
	}

	// Validate the token before persisting.
	result, err := p.Validate(r.Context(), req.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "failed to validate token: "+err.Error())
		return
	}
	if !result.Valid {
		respondJSON(w, http.StatusUnprocessableEntity, result)
		return
	}

	id, err := ph.store.Create(r.Context(), req.Type, req.Name, req.APIToken)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to save provider: "+err.Error())
		return
	}

	respondJSON(w, http.StatusCreated, map[string]any{
		"id":         id,
		"validation": result,
	})
}

func (ph *providerHandler) handleList(w http.ResponseWriter, r *http.Request) {
	providers, err := ph.store.List(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to list providers: "+err.Error())
		return
	}
	if providers == nil {
		providers = []provider.StoredProvider{}
	}
	respondJSON(w, http.StatusOK, providers)
}

func (ph *providerHandler) handleDelete(w http.ResponseWriter, r *http.Request, id int64) {
	if err := ph.store.Delete(r.Context(), id); err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type validateRequest struct {
	Type     string `json:"type"`
	APIToken string `json:"api_token"`
}

func (ph *providerHandler) handleValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req validateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p := provider.Get(req.Type)
	if p == nil {
		respondError(w, http.StatusBadRequest, "unsupported provider type: "+req.Type)
		return
	}

	result, err := p.Validate(r.Context(), req.APIToken)
	if err != nil {
		respondError(w, http.StatusBadGateway, "validation failed: "+err.Error())
		return
	}

	respondJSON(w, http.StatusOK, result)
}

func (ph *providerHandler) handleTypes(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	respondJSON(w, http.StatusOK, provider.All())
}
