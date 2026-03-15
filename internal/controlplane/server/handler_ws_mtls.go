//go:build !dev

package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"pressluft/internal/agent/agentauth"
	"pressluft/internal/infra/pki"
	"pressluft/internal/shared/idutil"
	"pressluft/internal/shared/ws"

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
	if r.TLS == nil {
		http.Error(w, "TLS required", http.StatusBadRequest)
		return
	}

	if len(r.TLS.VerifiedChains) == 0 {
		http.Error(w, "valid client certificate required", http.StatusUnauthorized)
		return
	}

	leaf := r.TLS.VerifiedChains[0][0]

	serverID, err := parseServerCN(leaf.Subject.CommonName)
	if err != nil {
		h.logger.Debug("invalid certificate CN", "cn", leaf.Subject.CommonName, "error", err)
		http.Error(w, "invalid certificate CN", http.StatusUnauthorized)
		return
	}

	if h.pkiStore.IsRevoked(leaf.SerialNumber.String()) {
		http.Error(w, "certificate revoked", http.StatusUnauthorized)
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

func parseServerCN(cn string) (string, error) {
	if strings.HasPrefix(cn, "server:") {
		serverID, err := idutil.Normalize(strings.TrimSpace(cn[7:]))
		if err != nil {
			return "", err
		}
		return serverID, nil
	}
	return "", fmt.Errorf("invalid CN format")
}
