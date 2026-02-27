package server

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"pressluft/internal/pki"
	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

type WSHandler struct {
	hub       *ws.Hub
	wsHandler *ws.Handler
	pkiStore  *pki.Store
	logger    *slog.Logger
}

func NewWSHandler(hub *ws.Hub, wsHandler *ws.Handler, pkiStore *pki.Store, logger *slog.Logger) *WSHandler {
	return &WSHandler{
		hub:       hub,
		wsHandler: wsHandler,
		pkiStore:  pkiStore,
		logger:    logger,
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

func parseServerCN(cn string) (int64, error) {
	if !strings.HasPrefix(cn, "server-") {
		return 0, fmt.Errorf("invalid CN format")
	}
	return strconv.ParseInt(cn[7:], 10, 64)
}
