package migration

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
var ErrSiteNotFound = errors.New("site not found")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrEnvironmentNotActive = errors.New("environment is not active")

type ImportInput struct {
	SiteID     string
	ArchiveURL string
}

type ImportResult struct {
	JobID         string
	EnvironmentID string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) ImportSite(ctx context.Context, input ImportInput) (ImportResult, error) {
	if err := validateImportInput(input); err != nil {
		return ImportResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return ImportResult{}, err
	}
	releaseID, err := newUUIDv4()
	if err != nil {
		return ImportResult{}, err
	}

	result := ImportResult{JobID: jobID}
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		siteRef, err := loadSiteImportRef(ctx, tx, strings.TrimSpace(input.SiteID))
		if err != nil {
			return err
		}

		targetURL, err := resolveTargetURL(ctx, tx, siteRef)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'restoring', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteRef.EnvironmentID); err != nil {
			return fmt.Errorf("mark environment restoring: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'restoring', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteRef.SiteID); err != nil {
			return fmt.Errorf("mark site restoring: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO releases (id, environment_id, source_type, source_ref, path, health_status, notes, created_at)
			VALUES (?, ?, 'upload', ?, ?, 'unknown', 'site import', ?)
		`, releaseID, siteRef.EnvironmentID, strings.TrimSpace(input.ArchiveURL), pendingReleasePath(releaseID), now); err != nil {
			return fmt.Errorf("insert import release: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"site_id":        siteRef.SiteID,
			"environment_id": siteRef.EnvironmentID,
			"node_id":        siteRef.NodeID,
			"archive_url":    strings.TrimSpace(input.ArchiveURL),
			"release_id":     releaseID,
			"target_url":     targetURL,
		})
		if err != nil {
			return fmt.Errorf("marshal site_import payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "site_import",
			SiteID:        sql.NullString{String: siteRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: siteRef.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: siteRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue site_import job: %w", err)
		}

		result.EnvironmentID = siteRef.EnvironmentID
		return nil
	})
	if err != nil {
		return ImportResult{}, err
	}

	return result, nil
}

type siteImportRef struct {
	SiteID            string
	EnvironmentID     string
	NodeID            string
	SiteStatus        string
	EnvironmentStatus string
	PreviewURL        string
	PrimaryDomainID   sql.NullString
}

func loadSiteImportRef(ctx context.Context, tx *sql.Tx, siteID string) (siteImportRef, error) {
	var ref siteImportRef
	err := tx.QueryRowContext(ctx, `
		SELECT s.id, s.primary_environment_id, s.status, e.status, e.node_id, e.preview_url, e.primary_domain_id
		FROM sites s
		JOIN environments e ON e.id = s.primary_environment_id
		WHERE s.id = ?
		LIMIT 1
	`, siteID).Scan(&ref.SiteID, &ref.EnvironmentID, &ref.SiteStatus, &ref.EnvironmentStatus, &ref.NodeID, &ref.PreviewURL, &ref.PrimaryDomainID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := tx.QueryRowContext(ctx, `SELECT 1 FROM sites WHERE id = ? LIMIT 1`, siteID).Scan(new(int)); err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return siteImportRef{}, ErrSiteNotFound
				}
				return siteImportRef{}, fmt.Errorf("query site for import: %w", err)
			}
			return siteImportRef{}, ErrEnvironmentNotFound
		}
		return siteImportRef{}, fmt.Errorf("query site import reference: %w", err)
	}

	if ref.SiteStatus != "active" || ref.EnvironmentStatus != "active" {
		return siteImportRef{}, ErrEnvironmentNotActive
	}

	return ref, nil
}

func resolveTargetURL(ctx context.Context, tx *sql.Tx, ref siteImportRef) (string, error) {
	if ref.PrimaryDomainID.Valid {
		var hostname string
		err := tx.QueryRowContext(ctx, `
			SELECT hostname
			FROM domains
			WHERE id = ?
			LIMIT 1
		`, ref.PrimaryDomainID.String).Scan(&hostname)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", fmt.Errorf("primary domain %s not found: %w", ref.PrimaryDomainID.String, ErrEnvironmentNotFound)
			}
			return "", fmt.Errorf("query primary domain hostname: %w", err)
		}

		hostname = strings.TrimSpace(hostname)
		if hostname != "" {
			return "https://" + hostname, nil
		}
	}

	targetURL := strings.TrimSpace(ref.PreviewURL)
	if targetURL == "" {
		return "", ErrInvalidInput
	}

	return targetURL, nil
}

func validateImportInput(input ImportInput) error {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.ArchiveURL) == "" {
		return ErrInvalidInput
	}
	if !strings.HasPrefix(input.ArchiveURL, "http://") && !strings.HasPrefix(input.ArchiveURL, "https://") {
		return ErrInvalidInput
	}
	return nil
}

func pendingReleasePath(releaseID string) string {
	return fmt.Sprintf("/var/www/sites/releases/%s", releaseID)
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
