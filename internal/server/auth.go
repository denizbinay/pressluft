package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"pressluft/internal/activity"
	"pressluft/internal/apitypes"
	"pressluft/internal/auth"
)

const defaultJSONBodyLimit int64 = 64 << 10

type authHandler struct {
	service       *auth.Service
	activityStore *activity.Store
	logger        *slog.Logger
}

func (h *authHandler) handleMe(w http.ResponseWriter, r *http.Request) {
	actor := auth.ActorFromContext(r.Context())
	if !actor.IsAuthenticated() {
		respondError(w, http.StatusUnauthorized, "authentication required")
		return
	}
	respondJSON(w, http.StatusOK, actor)
}

func (h *authHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req apitypes.LoginRequest
	if err := decodeJSONBody(w, r, defaultJSONBodyLimit, &req); err != nil {
		return
	}
	if err := req.Validate(); err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	actor, err := h.service.Login(r.Context(), w, r, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, auth.ErrInvalidCredentials) {
			h.emitSecurityActivity(r, activity.EventSecurityLoginFailed, activity.ActorAPI, "", fmt.Sprintf("Failed login for %s", req.Email), true)
			respondError(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		logger := h.logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("login failed", "error", err)
		respondError(w, http.StatusInternalServerError, "login failed")
		return
	}

	h.emitSecurityActivity(r, activity.EventSecurityLoginSucceeded, activity.ActorUser, actor.ID, fmt.Sprintf("User %s logged in", actor.Email), false)
	respondJSON(w, http.StatusOK, actor)
}

func (h *authHandler) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	actor := auth.ActorFromContext(r.Context())
	if err := h.service.Logout(r.Context(), w, r); err != nil {
		logger := h.logger
		if logger == nil {
			logger = slog.Default()
		}
		logger.Error("logout failed", "error", err)
		respondError(w, http.StatusInternalServerError, "logout failed")
		return
	}
	if actor.IsAuthenticated() {
		h.emitSecurityActivity(r, activity.EventSecurityLogout, activity.ActorUser, actor.ID, fmt.Sprintf("User %s logged out", actor.Email), false)
		h.emitSecurityActivity(r, activity.EventSecuritySessionRevoked, activity.ActorUser, actor.ID, fmt.Sprintf("Session revoked for %s", actor.Email), false)
	}
	respondJSON(w, http.StatusOK, apitypes.StatusResponse{Status: "ok"})
}

func (h *authHandler) emitSecurityActivity(r *http.Request, event activity.EventType, actorType activity.ActorType, actorID, title string, attention bool) {
	if h.activityStore == nil {
		return
	}
	_, _ = h.activityStore.Emit(r.Context(), activity.EmitInput{
		EventType:         event,
		Category:          activity.CategorySecurity,
		Level:             activity.LevelInfo,
		ResourceType:      activity.ResourceAccount,
		ActorType:         actorType,
		ActorID:           actorID,
		Title:             title,
		RequiresAttention: attention,
	})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, limit int64, dst any) error {
	if limit <= 0 {
		limit = defaultJSONBodyLimit
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return err
	}
	return nil
}
