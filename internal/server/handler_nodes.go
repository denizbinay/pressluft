package server

import (
	"context"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/pki"
	"pressluft/internal/registration"
)

const nodeCertificateReissueWindow = 14 * 24 * time.Hour
const nodeRegistrationBodyLimit int64 = 256 << 10

type nodePKIStore interface {
	GetValidCertForServer(serverID int64) (*pki.NodeCertificate, error)
	GetValidCertForServerTx(ctx context.Context, tx *sql.Tx, serverID int64) (*pki.NodeCertificate, error)
	SaveNodeCertificateTx(ctx context.Context, tx *sql.Tx, serverID int64, cert *x509.Certificate) error
	RevokeCertificateTx(ctx context.Context, tx *sql.Tx, serialNumber string) error
}

type nodeRegistrationStore interface {
	Validate(plaintext string, serverID int64) error
	ConsumeTx(ctx context.Context, tx *sql.Tx, plaintext string, serverID int64) error
}

type nodeCertificateAuthority interface {
	SignCSR(csr *x509.CertificateRequest, validityDays int) (*x509.Certificate, error)
	Certificate() *x509.Certificate
}

type NodeHandler struct {
	db                *sql.DB
	pkiStore          nodePKIStore
	registrationStore nodeRegistrationStore
	ca                nodeCertificateAuthority
	logger            *slog.Logger
}

func NewNodeHandler(db *sql.DB, pkiStore *pki.Store, registrationStore *registration.Store, ca *pki.CA, logger *slog.Logger) *NodeHandler {
	return &NodeHandler{
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

func (h *NodeHandler) handleNodeRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	serverIDStr := strings.TrimPrefix(r.URL.Path, "/api/nodes/")
	serverIDStr = strings.TrimSuffix(serverIDStr, "/register")
	serverID, err := strconv.ParseInt(serverIDStr, 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid server id")
		return
	}
	h.logger.Info("agent registration request received", "server_id", serverID)

	r.Body = http.MaxBytesReader(w, r.Body, nodeRegistrationBodyLimit)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	var req RegisterRequest
	if err := json.Unmarshal(body, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.registrationStore.Validate(req.Token, serverID); err != nil {
		h.handleRegistrationTokenError(w, serverID, err)
		return
	}

	existingCert, err := h.pkiStore.GetValidCertForServer(serverID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existingCert != nil && !shouldAllowReissue(existingCert, time.Now().UTC()) {
		respondError(w, http.StatusConflict, "valid certificate already exists")
		return
	}

	csrBytes := []byte(req.CSR)
	block, _ := pem.Decode(csrBytes)
	if block == nil || block.Type != "CERTIFICATE REQUEST" {
		respondError(w, http.StatusBadRequest, "invalid CSR")
		return
	}

	csr, err := pki.ParseCSRFromPEM(csrBytes)
	if err != nil {
		respondError(w, http.StatusBadRequest, "failed to parse CSR")
		return
	}
	if err := csr.CheckSignature(); err != nil {
		respondError(w, http.StatusBadRequest, "invalid CSR signature")
		return
	}

	expectedCN := fmt.Sprintf("server-%d", serverID)
	if csr.Subject.CommonName != expectedCN {
		respondError(w, http.StatusBadRequest, "CSR CN must match server ID")
		return
	}

	cert, err := h.ca.SignCSR(csr, 90)
	if err != nil {
		h.logger.Error("agent registration signing failed", "server_id", serverID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to sign certificate")
		return
	}

	tx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to begin registration transaction")
		return
	}
	defer tx.Rollback()

	existingCertTx, err := h.pkiStore.GetValidCertForServerTx(r.Context(), tx, serverID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		respondError(w, http.StatusInternalServerError, "internal error")
		return
	}
	if existingCertTx != nil && !shouldAllowReissue(existingCertTx, time.Now().UTC()) {
		respondError(w, http.StatusConflict, "valid certificate already exists")
		return
	}

	if err := h.registrationStore.ConsumeTx(r.Context(), tx, req.Token, serverID); err != nil {
		h.handleRegistrationTokenError(w, serverID, err)
		return
	}

	if existingCertTx != nil {
		if err := h.pkiStore.RevokeCertificateTx(r.Context(), tx, existingCertTx.SerialNumber); err != nil {
			h.logger.Error("agent registration certificate revoke failed", "server_id", serverID, "serial", existingCertTx.SerialNumber, "error", err)
			respondError(w, http.StatusInternalServerError, "failed to rotate certificate")
			return
		}
	}

	if err := h.pkiStore.SaveNodeCertificateTx(r.Context(), tx, serverID, cert); err != nil {
		h.logger.Error("agent registration certificate persistence failed", "server_id", serverID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to save certificate")
		return
	}

	if err := tx.Commit(); err != nil {
		h.logger.Error("agent registration transaction commit failed", "server_id", serverID, "error", err)
		respondError(w, http.StatusInternalServerError, "failed to persist certificate")
		return
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: h.ca.Certificate().Raw})

	resp := RegisterResponse{
		Certificate: string(certPEM),
		CACert:      string(caCertPEM),
	}
	h.logger.Info("agent registration completed", "server_id", serverID, "serial", cert.SerialNumber.String(), "expires_at", cert.NotAfter.UTC().Format(time.RFC3339))

	respondJSON(w, http.StatusOK, resp)
}

func shouldAllowReissue(cert *pki.NodeCertificate, now time.Time) bool {
	if cert == nil {
		return false
	}
	return cert.ExpiresAt.Sub(now) <= nodeCertificateReissueWindow
}

func (h *NodeHandler) handleRegistrationTokenError(w http.ResponseWriter, serverID int64, err error) {
	h.logger.Warn("agent registration token rejected", "server_id", serverID, "error", err)
	switch {
	case errors.Is(err, registration.ErrExpiredToken):
		respondError(w, http.StatusUnauthorized, "registration token expired")
	case errors.Is(err, registration.ErrConsumedToken):
		respondError(w, http.StatusUnauthorized, "registration token already consumed")
	case errors.Is(err, registration.ErrInvalidToken):
		respondError(w, http.StatusUnauthorized, "registration token invalid")
	default:
		respondError(w, http.StatusInternalServerError, "registration token lookup failed")
	}
}
