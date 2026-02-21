package sites

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrSlugConflict = errors.New("site slug already exists")
var ErrNoAvailableNode = errors.New("no available node")
var ErrNodeMissingPublicIP = errors.New("selected node missing public ip")
var ErrNotFound = errors.New("site not found")
var ErrResourceNotFailed = errors.New("resource not failed")
var ErrResetValidationFailed = errors.New("reset validation failed")

type Site struct {
	ID                   string  `json:"id"`
	Name                 string  `json:"name"`
	Slug                 string  `json:"slug"`
	Status               string  `json:"status"`
	PrimaryEnvironmentID *string `json:"primary_environment_id"`
	CreatedAt            string  `json:"created_at"`
	UpdatedAt            string  `json:"updated_at"`
	StateVersion         int     `json:"state_version"`
}

type CreateInput struct {
	Name string
	Slug string
}

type CreateResult struct {
	JobID  string
	SiteID string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	name := strings.TrimSpace(input.Name)
	slug := strings.ToLower(strings.TrimSpace(input.Slug))
	if name == "" || slug == "" {
		return CreateResult{}, ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	siteID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}
	environmentID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}
	jobID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		nodeID, nodePublicIP, err := selectCreateNode(ctx, tx)
		if err != nil {
			return err
		}

		previewDomain, err := loadPreviewDomain(ctx, tx)
		if err != nil {
			return err
		}

		previewURL, err := buildPreviewURL(environmentID, previewDomain, nodePublicIP)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO sites (
				id, name, slug, status, primary_environment_id,
				created_at, updated_at, state_version
			)
			VALUES (?, ?, ?, 'active', ?, ?, ?, 1)
		`, siteID, name, slug, environmentID, now, now); err != nil {
			if isUniqueSiteSlugError(err) {
				return ErrSlugConflict
			}
			return fmt.Errorf("insert site: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO environments (
				id, site_id, name, slug, environment_type, status,
				node_id, source_environment_id, promotion_preset, preview_url,
				primary_domain_id, current_release_id, drift_status, drift_checked_at,
				last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
				created_at, updated_at, state_version
			)
			VALUES (?, ?, 'Production', 'production', 'production', 'active', ?, NULL, 'content-protect', ?, NULL, NULL, 'unknown', NULL, NULL, 1, 1, ?, ?, 1)
		`, environmentID, siteID, nodeID, previewURL, now, now); err != nil {
			return fmt.Errorf("insert production environment: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"site_id":        siteID,
			"environment_id": environmentID,
			"node_id":        nodeID,
		})
		if err != nil {
			return fmt.Errorf("marshal site_create payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:       jobID,
			JobType:     "site_create",
			SiteID:      sql.NullString{String: siteID, Valid: true},
			NodeID:      sql.NullString{String: nodeID, Valid: true},
			PayloadJSON: string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue site_create job: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{JobID: jobID, SiteID: siteID}, nil
}

func (s *Service) List(ctx context.Context) ([]Site, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, slug, status, primary_environment_id, created_at, updated_at, state_version
		FROM sites
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query sites: %w", err)
	}
	defer rows.Close()

	sites := make([]Site, 0)
	for rows.Next() {
		site, err := scanSite(rows)
		if err != nil {
			return nil, err
		}
		sites = append(sites, site)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sites: %w", err)
	}

	return sites, nil
}

func (s *Service) Get(ctx context.Context, id string) (Site, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, name, slug, status, primary_environment_id, created_at, updated_at, state_version
		FROM sites
		WHERE id = ?
		LIMIT 1
	`, id)

	site, err := scanSite(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Site{}, ErrNotFound
		}
		return Site{}, err
	}

	return site, nil
}

func (s *Service) ResetFailed(ctx context.Context, id string) (Site, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Site{}, ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err := store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, `
			SELECT status
			FROM sites
			WHERE id = ?
			LIMIT 1
		`, id).Scan(&status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("query site for reset: %w", err)
		}

		if status != "failed" {
			return ErrResourceNotFailed
		}

		var activeMutationCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM jobs
			WHERE site_id = ?
			  AND status IN ('queued', 'running')
		`, id).Scan(&activeMutationCount); err != nil {
			return fmt.Errorf("query active site jobs for reset: %w", err)
		}
		if activeMutationCount > 0 {
			return ErrResetValidationFailed
		}

		var activeEnvMutationCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM environments
			WHERE site_id = ?
			  AND status IN ('cloning', 'deploying', 'restoring')
		`, id).Scan(&activeEnvMutationCount); err != nil {
			return fmt.Errorf("query active environment states for reset: %w", err)
		}
		if activeEnvMutationCount > 0 {
			return ErrResetValidationFailed
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, id); err != nil {
			return fmt.Errorf("reset site status: %w", err)
		}

		return nil
	})
	if err != nil {
		return Site{}, err
	}

	return s.Get(ctx, id)
}

type siteScanner interface {
	Scan(dest ...any) error
}

func scanSite(scanner siteScanner) (Site, error) {
	var site Site
	var primaryEnvironmentID sql.NullString
	if err := scanner.Scan(
		&site.ID,
		&site.Name,
		&site.Slug,
		&site.Status,
		&primaryEnvironmentID,
		&site.CreatedAt,
		&site.UpdatedAt,
		&site.StateVersion,
	); err != nil {
		return Site{}, err
	}

	if primaryEnvironmentID.Valid {
		site.PrimaryEnvironmentID = &primaryEnvironmentID.String
	}

	return site, nil
}

func selectCreateNode(ctx context.Context, tx *sql.Tx) (string, string, error) {
	var nodeID string
	var publicIP sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, public_ip
		FROM nodes
		WHERE status IN ('active', 'provisioning')
		ORDER BY is_local DESC, created_at ASC
		LIMIT 1
	`).Scan(&nodeID, &publicIP)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrNoAvailableNode
		}
		return "", "", fmt.Errorf("select node for site create: %w", err)
	}

	return nodeID, strings.TrimSpace(publicIP.String), nil
}

func loadPreviewDomain(ctx context.Context, tx *sql.Tx) (string, error) {
	var previewDomain string
	err := tx.QueryRowContext(ctx, `
		SELECT value
		FROM settings
		WHERE key = 'preview_domain'
		LIMIT 1
	`).Scan(&previewDomain)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("load preview domain setting: %w", err)
	}

	return strings.TrimSpace(previewDomain), nil
}

func buildPreviewURL(environmentID, previewDomain, nodePublicIP string) (string, error) {
	idPrefix := environmentID
	if len(idPrefix) > 8 {
		idPrefix = idPrefix[:8]
	}

	if previewDomain != "" {
		return fmt.Sprintf("https://%s.%s", idPrefix, previewDomain), nil
	}

	ip := strings.TrimSpace(nodePublicIP)
	if ip == "" {
		return "", ErrNodeMissingPublicIP
	}

	return fmt.Sprintf("http://%s.%s.sslip.io", idPrefix, strings.ReplaceAll(ip, ".", "-")), nil
}

func isUniqueSiteSlugError(err error) bool {
	return strings.Contains(err.Error(), "UNIQUE constraint failed: sites.slug")
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
