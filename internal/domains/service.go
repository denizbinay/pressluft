package domains

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrEnvironmentNotActive = errors.New("environment is not active")
var ErrDomainNotFound = errors.New("domain not found")
var ErrDomainConflict = errors.New("domain conflict")
var ErrNodeMissingPublicIP = errors.New("selected node missing public ip")

const DNSMismatchErrorCode = "DOMAIN_DNS_MISMATCH"

type Domain struct {
	ID            string `json:"id"`
	EnvironmentID string `json:"environment_id"`
	Hostname      string `json:"hostname"`
	TLSStatus     string `json:"tls_status"`
	TLSIssuer     string `json:"tls_issuer"`
	CreatedAt     string `json:"created_at"`
	UpdatedAt     string `json:"updated_at"`
}

type AddInput struct {
	EnvironmentID string
	Hostname      string
}

type AddResult struct {
	JobID    string
	DomainID string
}

type RemoveResult struct {
	JobID string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ListByEnvironment(ctx context.Context, environmentID string) ([]Domain, error) {
	if strings.TrimSpace(environmentID) == "" {
		return nil, ErrInvalidInput
	}

	if err := assertEnvironmentExists(ctx, s.db, environmentID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, environment_id, hostname, tls_status, tls_issuer, created_at, updated_at
		FROM domains
		WHERE environment_id = ?
		ORDER BY created_at ASC
	`, strings.TrimSpace(environmentID))
	if err != nil {
		return nil, fmt.Errorf("query domains: %w", err)
	}
	defer rows.Close()

	domains := make([]Domain, 0)
	for rows.Next() {
		var domain Domain
		if err := rows.Scan(&domain.ID, &domain.EnvironmentID, &domain.Hostname, &domain.TLSStatus, &domain.TLSIssuer, &domain.CreatedAt, &domain.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan domain row: %w", err)
		}
		domains = append(domains, domain)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domains: %w", err)
	}

	return domains, nil
}

func (s *Service) Add(ctx context.Context, input AddInput) (AddResult, error) {
	if err := validateAddInput(input); err != nil {
		return AddResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	domainID, err := newUUIDv4()
	if err != nil {
		return AddResult{}, err
	}
	jobID, err := newUUIDv4()
	if err != nil {
		return AddResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		envRef, err := loadEnvironmentForMutation(ctx, tx, strings.TrimSpace(input.EnvironmentID))
		if err != nil {
			return err
		}

		if strings.TrimSpace(envRef.NodePublicIP) == "" {
			return ErrNodeMissingPublicIP
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO domains (id, environment_id, hostname, tls_status, tls_issuer, created_at, updated_at)
			VALUES (?, ?, ?, 'pending', 'letsencrypt', ?, ?)
		`, domainID, envRef.EnvironmentID, strings.ToLower(strings.TrimSpace(input.Hostname)), now, now); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed: domains.hostname") {
				return ErrDomainConflict
			}
			return fmt.Errorf("insert domain: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":  envRef.EnvironmentID,
			"domain_id":       domainID,
			"domain_hostname": strings.ToLower(strings.TrimSpace(input.Hostname)),
			"node_public_ip":  strings.TrimSpace(envRef.NodePublicIP),
		})
		if err != nil {
			return fmt.Errorf("marshal domain_add payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "domain_add",
			SiteID:        sql.NullString{String: envRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: envRef.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: envRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue domain_add job: %w", err)
		}

		return nil
	})
	if err != nil {
		return AddResult{}, err
	}

	return AddResult{JobID: jobID, DomainID: domainID}, nil
}

func (s *Service) Remove(ctx context.Context, domainID string) (RemoveResult, error) {
	if strings.TrimSpace(domainID) == "" {
		return RemoveResult{}, ErrInvalidInput
	}

	jobID, err := newUUIDv4()
	if err != nil {
		return RemoveResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		domainRef, err := loadDomainForMutation(ctx, tx, strings.TrimSpace(domainID))
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":  domainRef.EnvironmentID,
			"domain_id":       domainRef.DomainID,
			"domain_hostname": domainRef.Hostname,
			"preview_url":     domainRef.PreviewURL,
		})
		if err != nil {
			return fmt.Errorf("marshal domain_remove payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "domain_remove",
			SiteID:        sql.NullString{String: domainRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: domainRef.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: domainRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue domain_remove job: %w", err)
		}

		return nil
	})
	if err != nil {
		return RemoveResult{}, err
	}

	return RemoveResult{JobID: jobID}, nil
}

func (s *Service) MarkAddSucceeded(ctx context.Context, domainID, environmentID, jobID string) error {
	if strings.TrimSpace(domainID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE domains
			SET tls_status = 'active', updated_at = ?
			WHERE id = ? AND environment_id = ?
		`, now, domainID, environmentID); err != nil {
			return fmt.Errorf("mark domain active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET primary_domain_id = CASE WHEN primary_domain_id IS NULL THEN ? ELSE primary_domain_id END,
			    updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, domainID, now, environmentID); err != nil {
			return fmt.Errorf("set primary domain: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark domain_add job succeeded: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkAddFailed(ctx context.Context, domainID, jobID, errorCode, errorMessage string) error {
	if strings.TrimSpace(domainID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	if strings.TrimSpace(errorCode) == "" {
		errorCode = "DOMAIN_ADD_FAILED"
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "domain add failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE domains
			SET tls_status = 'failed', updated_at = ?
			WHERE id = ?
		`, now, domainID); err != nil {
			return fmt.Errorf("mark domain failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
			WHERE id = ?
		`, now, strings.TrimSpace(errorCode), truncateMessage(errorMessage), now, jobID); err != nil {
			return fmt.Errorf("mark domain_add job failed: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkRemoveSucceeded(ctx context.Context, domainID, environmentID, jobID string) error {
	if strings.TrimSpace(domainID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `DELETE FROM domains WHERE id = ? AND environment_id = ?`, domainID, environmentID); err != nil {
			return fmt.Errorf("delete domain: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET primary_domain_id = CASE WHEN primary_domain_id = ? THEN NULL ELSE primary_domain_id END,
			    updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, domainID, now, environmentID); err != nil {
			return fmt.Errorf("clear primary domain: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark domain_remove job succeeded: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkRemoveFailed(ctx context.Context, jobID, errorCode, errorMessage string) error {
	if strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	if strings.TrimSpace(errorCode) == "" {
		errorCode = "DOMAIN_REMOVE_FAILED"
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "domain remove failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := s.db.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, now, strings.TrimSpace(errorCode), truncateMessage(errorMessage), now, jobID); err != nil {
		return fmt.Errorf("mark domain_remove job failed: %w", err)
	}

	return nil
}

func DNSMismatchError(hostname, expectedNodeIP string) (string, string) {
	host := strings.ToLower(strings.TrimSpace(hostname))
	ip := strings.TrimSpace(expectedNodeIP)
	if host == "" {
		host = "<unknown-hostname>"
	}
	if ip == "" {
		ip = "<unknown-node-ip>"
	}
	return DNSMismatchErrorCode, fmt.Sprintf("dns mismatch: %s does not resolve to node ip %s", host, ip)
}

type environmentRef struct {
	SiteID        string
	EnvironmentID string
	NodeID        string
	NodePublicIP  string
	Status        string
}

func loadEnvironmentForMutation(ctx context.Context, tx *sql.Tx, environmentID string) (environmentRef, error) {
	var ref environmentRef
	err := tx.QueryRowContext(ctx, `
		SELECT e.site_id, e.id, e.node_id, COALESCE(n.public_ip, ''), e.status
		FROM environments e
		JOIN nodes n ON n.id = e.node_id
		WHERE e.id = ?
		LIMIT 1
	`, environmentID).Scan(&ref.SiteID, &ref.EnvironmentID, &ref.NodeID, &ref.NodePublicIP, &ref.Status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return environmentRef{}, ErrEnvironmentNotFound
		}
		return environmentRef{}, fmt.Errorf("query environment for domain mutation: %w", err)
	}

	if ref.Status != "active" {
		return environmentRef{}, ErrEnvironmentNotActive
	}

	return ref, nil
}

type domainRef struct {
	DomainID          string
	EnvironmentID     string
	SiteID            string
	NodeID            string
	Hostname          string
	PreviewURL        string
	EnvironmentStatus string
}

func loadDomainForMutation(ctx context.Context, tx *sql.Tx, domainID string) (domainRef, error) {
	var ref domainRef
	err := tx.QueryRowContext(ctx, `
		SELECT d.id, d.environment_id, e.site_id, e.node_id, d.hostname, e.preview_url, e.status
		FROM domains d
		JOIN environments e ON e.id = d.environment_id
		WHERE d.id = ?
		LIMIT 1
	`, domainID).Scan(&ref.DomainID, &ref.EnvironmentID, &ref.SiteID, &ref.NodeID, &ref.Hostname, &ref.PreviewURL, &ref.EnvironmentStatus)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domainRef{}, ErrDomainNotFound
		}
		return domainRef{}, fmt.Errorf("query domain for mutation: %w", err)
	}

	if ref.EnvironmentStatus != "active" {
		return domainRef{}, ErrEnvironmentNotActive
	}

	return ref, nil
}

func assertEnvironmentExists(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, environmentID string) error {
	var exists int
	if err := q.QueryRowContext(ctx, `SELECT 1 FROM environments WHERE id = ? LIMIT 1`, strings.TrimSpace(environmentID)).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEnvironmentNotFound
		}
		return fmt.Errorf("query environment: %w", err)
	}

	return nil
}

func validateAddInput(input AddInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return ErrInvalidInput
	}

	hostname := strings.ToLower(strings.TrimSpace(input.Hostname))
	if hostname == "" {
		return ErrInvalidInput
	}
	if strings.Contains(hostname, " ") {
		return ErrInvalidInput
	}
	if net.ParseIP(hostname) != nil {
		return ErrInvalidInput
	}

	if strings.Contains(hostname, ".") && !strings.HasPrefix(hostname, ".") && !strings.HasSuffix(hostname, ".") {
		return nil
	}

	return ErrInvalidInput
}

func truncateMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "domain mutation failed"
	}
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func newUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}
