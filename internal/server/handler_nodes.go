package server

import (
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"pressluft/internal/pki"
	"pressluft/internal/registration"
)

type Handler struct {
	db                *sql.DB
	pkiStore          *pki.Store
	registrationStore *registration.Store
	ca                *pki.CA
	logger            *slog.Logger
}

func NewNodeHandler(db *sql.DB, pkiStore *pki.Store, registrationStore *registration.Store, ca *pki.CA, logger *slog.Logger) *Handler {
	return &Handler{
		db:                db,
		pkiStore:          pkiStore,
		registrationStore: registrationStore,
		ca:                ca,
		logger:            logger,
	}
}

type RegisterRequest struct {
	Token string `json:"token"`
	CSR   string `json:"csr"`
}

type RegisterResponse struct {
	Certificate string `json:"certificate"`
	CACert      string `json:"ca_certificate"`
}

func (h *Handler) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	serverIDStr := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
	serverIDStr = strings.TrimSuffix(serverIDStr, "/register")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid server id", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	if err := h.registrationStore.Consume(req.Token, serverID); err != nil {
		h.logger.Debug("token consumption failed", "server_id", serverID, "error", err)
		http.Error(w, "invalid or expired token", http.StatusUnauthorized)
		return
	}

	existingCert, err := h.pkiStore.GetValidCertForServer(serverID)
	if err != nil && err != sql.ErrNoRows {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	if existingCert != nil {
		http.Error(w, "valid certificate already exists", http.StatusConflict)
		return
	}

	csrBytes := []byte(req.CSR)
	block, _ := pem.Decode(csrBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		http.Error(w, "invalid CSR", http.StatusBadRequest)
		return
	}

	csr, err := pki.ParseCSRFromPEM(csrBytes)
	if err != nil {
		http.Error(w, "failed to parse CSR", http.StatusBadRequest)
		return
	}

	expectedCN := fmt.Sprintf("server-%d", serverID)
	if csr.Subject.CommonName != expectedCN {
		http.Error(w, "CSR CN must match server ID", http.StatusBadRequest)
		return
	}

	cert, err := h.ca.SignCSR(csr, 90)
	if err != nil {
		h.logger.Error("failed to sign CSR", "error", err)
		http.Error(w, "failed to sign certificate", http.StatusInternalServerError)
		return
	}

	if err := h.pkiStore.SaveNodeCertificate(serverID, cert); err != nil {
		h.logger.Error("failed to save certificate", "error", err)
		http.Error(w, "failed to save certificate", http.StatusInternalServerError)
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: h.ca.Certificate().Raw})

	resp := RegisterResponse{
		Certificate: string(certPEM),
		CACert:      string(caCertPEM),
	}

	respondJSON(w, http.StatusOK, resp)
}
