//go:build dev

package server

import (
	"log/slog"
	"net/http"
	"strings"

	"pressluft/internal/agentauth"
	"pressluft/internal/pki"
	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

type WSHandler struct {
	hub        *ws.Hub
	wsHandler  *ws.Handler
	pkiStore   *pki.Store
	tokenStore *agentauth.Store
	logger     *slog.Logger
}

func NewWSHandler(hub *ws.Hub, wsHandler *ws.Handler, pkiStore *pki.Store, tokenStore *agentauth.Store, logger *slog.Logger) *WSHandler {
	return &WSHandler{
		hub:        hub,
		wsHandler:  wsHandler,
		pkiStore:   pkiStore,
		tokenStore: tokenStore,
		logger:     logger,
	}
}

func (h *WSHandler) handleAgentWebSocket(w http.ResponseWriter, r *http.Request) {
	token := extractDevToken(r)
	if token == "" {
		http.Error(w, "missing dev token", http.StatusUnauthorized)
		return
	}

	if h.tokenStore == nil {
		http.Error(w, "dev token store not configured", http.StatusInternalServerError)
		return
	}

	serverID, err := h.tokenStore.ValidateAndLookupServerID(token)
	if err != nil {
		h.logger.Debug("invalid dev token", "error", err)
		http.Error(w, "invalid dev token", http.StatusUnauthorized)
		return
	}

	wsConn, err := websocket.Accept(w, r, nil)
	if err != nil {
		h.logger.Error("websocket accept failed", "error", err)
		return
	}

	conn := ws.NewConn(wsConn, serverID)
	h.hub.Register(conn)

	h.wsHandler.HandleConnection(r.Context(), conn)
}

func extractDevToken(r *http.Request) string {
	token := strings.TrimSpace(r.Header.Get("X-Pressluft-Dev-Token"))
	if token != "" {
		return token
	}

	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if authorization == "" {
		return ""
	}

	const prefix = "Bearer "
	if !strings.HasPrefix(authorization, prefix) {
		return ""
	}

	return strings.TrimSpace(authorization[len(prefix):])
}
