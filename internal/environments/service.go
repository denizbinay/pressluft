package environments

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
var ErrBackupNotFound = errors.New("backup not found")
var ErrBackupNotCompleted = errors.New("backup is not completed")
var ErrNoAvailableNode = errors.New("no available node")
var ErrNodeMissingPublicIP = errors.New("selected node missing public ip")
var ErrResourceNotFailed = errors.New("resource not failed")
var ErrResetValidationFailed = errors.New("reset validation failed")

const (
	preUpdateBackupFreshnessWindow = 60 * time.Minute
	backupRetentionDays            = 30
)

type Environment struct {
	ID                  string  `json:"id"`
	SiteID              string  `json:"site_id"`
	Name                string  `json:"name"`
	Slug                string  `json:"slug"`
	EnvironmentType     string  `json:"environment_type"`
	Status              string  `json:"status"`
	NodeID              string  `json:"node_id"`
	SourceEnvironmentID *string `json:"source_environment_id"`
	PromotionPreset     string  `json:"promotion_preset"`
	PreviewURL          string  `json:"preview_url"`
	PrimaryDomainID     *string `json:"primary_domain_id"`
	CurrentReleaseID    *string `json:"current_release_id"`
	DriftStatus         string  `json:"drift_status"`
	DriftCheckedAt      *string `json:"drift_checked_at"`
	LastDriftCheckID    *string `json:"last_drift_check_id"`
	FastCGICacheEnabled bool    `json:"fastcgi_cache_enabled"`
	RedisCacheEnabled   bool    `json:"redis_cache_enabled"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
	StateVersion        int     `json:"state_version"`
}

type CreateInput struct {
	SiteID              string
	Name                string
	Slug                string
	EnvironmentType     string
	SourceEnvironmentID *string
	PromotionPreset     string
}

type CreateResult struct {
	JobID         string
	EnvironmentID string
}

type DeployInput struct {
	EnvironmentID string
	SourceType    string
	SourceRef     string
}

type DeployResult struct {
	JobID     string
	ReleaseID string
}

type UpdatesInput struct {
	EnvironmentID string
	Scope         string
}

type UpdatesResult struct {
	JobID           string
	PreUpdateBackup string
}

type RestoreInput struct {
	EnvironmentID string
	BackupID      string
}

type RestoreResult struct {
	JobID            string
	PreRestoreBackup string
}

type CacheToggleInput struct {
	EnvironmentID       string
	FastCGICacheEnabled *bool
	RedisCacheEnabled   *bool
}

type CacheToggleResult struct {
	JobID string
}

type CachePurgeInput struct {
	EnvironmentID string
}

type CachePurgeResult struct {
	JobID string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	if err := validateCreateInput(input); err != nil {
		return CreateResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	environmentID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}
	jobID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := assertSiteExists(ctx, tx, input.SiteID); err != nil {
			return err
		}

		if err := assertSourceEnvironment(ctx, tx, input.SiteID, input.SourceEnvironmentID); err != nil {
			return err
		}

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

		var sourceID any = nil
		if input.SourceEnvironmentID != nil {
			sourceID = *input.SourceEnvironmentID
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO environments (
				id, site_id, name, slug, environment_type, status,
				node_id, source_environment_id, promotion_preset, preview_url,
				primary_domain_id, current_release_id, drift_status, drift_checked_at,
				last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
				created_at, updated_at, state_version
			)
			VALUES (?, ?, ?, ?, ?, 'cloning', ?, ?, ?, ?, NULL, NULL, 'unknown', NULL, NULL, 1, 1, ?, ?, 1)
		`, environmentID, input.SiteID, strings.TrimSpace(input.Name), strings.ToLower(strings.TrimSpace(input.Slug)), input.EnvironmentType, nodeID, sourceID, input.PromotionPreset, previewURL, now, now); err != nil {
			if strings.Contains(err.Error(), "UNIQUE constraint failed: environments.site_id, environments.slug") {
				return fmt.Errorf("environment slug already exists: %w", ErrInvalidInput)
			}
			return fmt.Errorf("insert environment: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'cloning', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.SiteID); err != nil {
			return fmt.Errorf("update site cloning status: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"site_id":               input.SiteID,
			"environment_id":        environmentID,
			"node_id":               nodeID,
			"source_environment_id": strings.TrimSpace(valueOrEmpty(input.SourceEnvironmentID)),
		})
		if err != nil {
			return fmt.Errorf("marshal env_create payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_create",
			SiteID:        sql.NullString{String: input.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: nodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue env_create job: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{JobID: jobID, EnvironmentID: environmentID}, nil
}

func (s *Service) ListBySite(ctx context.Context, siteID string) ([]Environment, error) {
	if strings.TrimSpace(siteID) == "" {
		return nil, ErrInvalidInput
	}

	if err := assertSiteExists(ctx, s.db, siteID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
		       promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
		       drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
		       created_at, updated_at, state_version
		FROM environments
		WHERE site_id = ?
		ORDER BY created_at ASC
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("query environments for site: %w", err)
	}
	defer rows.Close()

	environments := make([]Environment, 0)
	for rows.Next() {
		environment, err := scanEnvironment(rows)
		if err != nil {
			return nil, err
		}
		environments = append(environments, environment)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate environments: %w", err)
	}

	return environments, nil
}

func (s *Service) Get(ctx context.Context, id string) (Environment, error) {
	if strings.TrimSpace(id) == "" {
		return Environment{}, ErrInvalidInput
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
		       promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
		       drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
		       created_at, updated_at, state_version
		FROM environments
		WHERE id = ?
		LIMIT 1
	`, id)

	environment, err := scanEnvironment(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Environment{}, ErrEnvironmentNotFound
		}
		return Environment{}, err
	}

	return environment, nil
}

func (s *Service) Deploy(ctx context.Context, input DeployInput) (DeployResult, error) {
	if err := validateDeployInput(input); err != nil {
		return DeployResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return DeployResult{}, err
	}
	releaseID, err := newUUIDv4()
	if err != nil {
		return DeployResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		environmentID := strings.TrimSpace(input.EnvironmentID)
		environmentRef, err := loadEnvironmentForMutation(ctx, tx, environmentID)
		if err != nil {
			return err
		}

		if err := markEnvironmentAndSiteDeploying(ctx, tx, environmentID, environmentRef, now); err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO releases (id, environment_id, source_type, source_ref, path, health_status, notes, created_at)
			VALUES (?, ?, ?, ?, ?, 'unknown', NULL, ?)
		`, releaseID, environmentID, input.SourceType, strings.TrimSpace(input.SourceRef), pendingReleasePath(releaseID), now); err != nil {
			return fmt.Errorf("insert deploy release: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id": environmentID,
			"release_id":     releaseID,
			"source_type":    input.SourceType,
			"source_ref":     strings.TrimSpace(input.SourceRef),
		})
		if err != nil {
			return fmt.Errorf("marshal env_deploy payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_deploy",
			SiteID:        sql.NullString{String: environmentRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: environmentRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue env_deploy job: %w", err)
		}

		return nil
	})
	if err != nil {
		return DeployResult{}, err
	}

	return DeployResult{JobID: jobID, ReleaseID: releaseID}, nil
}

func (s *Service) Updates(ctx context.Context, input UpdatesInput) (UpdatesResult, error) {
	if err := validateUpdatesInput(input); err != nil {
		return UpdatesResult{}, err
	}

	nowTime := time.Now().UTC()
	now := nowTime.Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return UpdatesResult{}, err
	}

	result := UpdatesResult{JobID: jobID}
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		environmentID := strings.TrimSpace(input.EnvironmentID)
		environmentRef, err := loadEnvironmentForMutation(ctx, tx, environmentID)
		if err != nil {
			return err
		}

		if err := markEnvironmentAndSiteDeploying(ctx, tx, environmentID, environmentRef, now); err != nil {
			return err
		}

		backupID, err := ensureFreshPreUpdateBackup(ctx, tx, environmentID, nowTime)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":          environmentID,
			"scope":                   input.Scope,
			"pre_update_backup_id":    backupID,
			"pre_update_backup_fresh": "true",
		})
		if err != nil {
			return fmt.Errorf("marshal env_update payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_update",
			SiteID:        sql.NullString{String: environmentRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: environmentRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue env_update job: %w", err)
		}

		result.PreUpdateBackup = backupID
		return nil
	})
	if err != nil {
		return UpdatesResult{}, err
	}

	return result, nil
}

func (s *Service) Restore(ctx context.Context, input RestoreInput) (RestoreResult, error) {
	if err := validateRestoreInput(input); err != nil {
		return RestoreResult{}, err
	}

	nowTime := time.Now().UTC()
	now := nowTime.Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return RestoreResult{}, err
	}

	result := RestoreResult{JobID: jobID}
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		environmentID := strings.TrimSpace(input.EnvironmentID)
		environmentRef, err := loadEnvironmentForMutation(ctx, tx, environmentID)
		if err != nil {
			return err
		}

		backupID, err := assertRestorableBackup(ctx, tx, environmentID, input.BackupID)
		if err != nil {
			return err
		}

		if err := markEnvironmentAndSiteRestoring(ctx, tx, environmentID, environmentRef, now); err != nil {
			return err
		}

		preRestoreBackupID, err := ensureFreshPreRestoreBackup(ctx, tx, environmentID, backupID, nowTime)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":           environmentID,
			"backup_id":                backupID,
			"pre_restore_backup_id":    preRestoreBackupID,
			"pre_restore_backup_fresh": "true",
		})
		if err != nil {
			return fmt.Errorf("marshal env_restore payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_restore",
			SiteID:        sql.NullString{String: environmentRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: environmentRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue env_restore job: %w", err)
		}

		result.PreRestoreBackup = preRestoreBackupID
		return nil
	})
	if err != nil {
		return RestoreResult{}, err
	}

	return result, nil
}

func (s *Service) ToggleCache(ctx context.Context, input CacheToggleInput) (CacheToggleResult, error) {
	if err := validateCacheToggleInput(input); err != nil {
		return CacheToggleResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return CacheToggleResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		environmentID := strings.TrimSpace(input.EnvironmentID)
		environmentRef, err := loadEnvironmentForMutation(ctx, tx, environmentID)
		if err != nil {
			return err
		}

		updateClauses := []string{"updated_at = ?", "state_version = state_version + 1"}
		updateArgs := []any{now}
		payload := map[string]any{"environment_id": environmentID}

		if input.FastCGICacheEnabled != nil {
			updateClauses = append(updateClauses, "fastcgi_cache_enabled = ?")
			updateArgs = append(updateArgs, boolToSQLiteInt(*input.FastCGICacheEnabled))
			payload["fastcgi_cache_enabled"] = *input.FastCGICacheEnabled
		}

		if input.RedisCacheEnabled != nil {
			updateClauses = append(updateClauses, "redis_cache_enabled = ?")
			updateArgs = append(updateArgs, boolToSQLiteInt(*input.RedisCacheEnabled))
			payload["redis_cache_enabled"] = *input.RedisCacheEnabled
		}

		updateArgs = append(updateArgs, environmentID)
		query := "UPDATE environments SET " + strings.Join(updateClauses, ", ") + " WHERE id = ?"
		if _, err := tx.ExecContext(ctx, query, updateArgs...); err != nil {
			return fmt.Errorf("update environment cache settings: %w", err)
		}

		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("marshal env_cache_toggle payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_cache_toggle",
			SiteID:        sql.NullString{String: environmentRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: environmentRef.NodeID, Valid: true},
			PayloadJSON:   string(payloadJSON),
		}); err != nil {
			return fmt.Errorf("enqueue env_cache_toggle job: %w", err)
		}

		return nil
	})
	if err != nil {
		return CacheToggleResult{}, err
	}

	return CacheToggleResult{JobID: jobID}, nil
}

func (s *Service) PurgeCache(ctx context.Context, input CachePurgeInput) (CachePurgeResult, error) {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return CachePurgeResult{}, ErrInvalidInput
	}

	jobID, err := newUUIDv4()
	if err != nil {
		return CachePurgeResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		environmentID := strings.TrimSpace(input.EnvironmentID)
		environmentRef, err := loadEnvironmentForMutation(ctx, tx, environmentID)
		if err != nil {
			return err
		}

		payloadJSON, err := json.Marshal(map[string]any{
			"environment_id":        environmentID,
			"fastcgi_cache_enabled": environmentRef.FastCGICacheEnabled,
			"redis_cache_enabled":   environmentRef.RedisCacheEnabled,
		})
		if err != nil {
			return fmt.Errorf("marshal cache_purge payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "cache_purge",
			SiteID:        sql.NullString{String: environmentRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: environmentID, Valid: true},
			NodeID:        sql.NullString{String: environmentRef.NodeID, Valid: true},
			PayloadJSON:   string(payloadJSON),
		}); err != nil {
			return fmt.Errorf("enqueue cache_purge job: %w", err)
		}

		return nil
	})
	if err != nil {
		return CachePurgeResult{}, err
	}

	return CachePurgeResult{JobID: jobID}, nil
}

func (s *Service) ResetFailed(ctx context.Context, environmentID string) (Environment, error) {
	environmentID = strings.TrimSpace(environmentID)
	if environmentID == "" {
		return Environment{}, ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err := store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var siteID string
		var status string
		if err := tx.QueryRowContext(ctx, `
			SELECT site_id, status
			FROM environments
			WHERE id = ?
			LIMIT 1
		`, environmentID).Scan(&siteID, &status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrEnvironmentNotFound
			}
			return fmt.Errorf("query environment for reset: %w", err)
		}

		if status != "failed" {
			return ErrResourceNotFailed
		}

		var activeJobCount int
		if err := tx.QueryRowContext(ctx, `
			SELECT COUNT(1)
			FROM jobs
			WHERE environment_id = ?
			  AND status IN ('queued', 'running')
		`, environmentID).Scan(&activeJobCount); err != nil {
			return fmt.Errorf("query active environment jobs for reset: %w", err)
		}
		if activeJobCount > 0 {
			return ErrResetValidationFailed
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, environmentID); err != nil {
			return fmt.Errorf("reset environment status: %w", err)
		}

		var siteStatus string
		if err := tx.QueryRowContext(ctx, `
			SELECT status
			FROM sites
			WHERE id = ?
			LIMIT 1
		`, siteID).Scan(&siteStatus); err != nil {
			return fmt.Errorf("query site for environment reset: %w", err)
		}

		if siteStatus == "failed" {
			var otherFailedCount int
			if err := tx.QueryRowContext(ctx, `
				SELECT COUNT(1)
				FROM environments
				WHERE site_id = ?
				  AND id <> ?
				  AND status = 'failed'
			`, siteID, environmentID).Scan(&otherFailedCount); err != nil {
				return fmt.Errorf("query other failed environments for reset: %w", err)
			}

			var activeMutationCount int
			if err := tx.QueryRowContext(ctx, `
				SELECT COUNT(1)
				FROM environments
				WHERE site_id = ?
				  AND status IN ('cloning', 'deploying', 'restoring')
			`, siteID).Scan(&activeMutationCount); err != nil {
				return fmt.Errorf("query active environment mutations for reset: %w", err)
			}

			if otherFailedCount == 0 && activeMutationCount == 0 {
				if _, err := tx.ExecContext(ctx, `
					UPDATE sites
					SET status = 'active', updated_at = ?, state_version = state_version + 1
					WHERE id = ?
				`, now, siteID); err != nil {
					return fmt.Errorf("reset parent site status: %w", err)
				}
			}
		}

		return nil
	})
	if err != nil {
		return Environment{}, err
	}

	return s.Get(ctx, environmentID)
}

func (s *Service) MarkDeployOrUpdateSucceeded(ctx context.Context, siteID, environmentID, jobID, releaseID string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if strings.TrimSpace(releaseID) != "" {
			if _, err := tx.ExecContext(ctx, `
				UPDATE environments
				SET current_release_id = ?, status = 'active', updated_at = ?, state_version = state_version + 1
				WHERE id = ? AND site_id = ?
			`, strings.TrimSpace(releaseID), now, environmentID, siteID); err != nil {
				return fmt.Errorf("mark environment active with release: %w", err)
			}
		} else {
			if _, err := tx.ExecContext(ctx, `
				UPDATE environments
				SET status = 'active', updated_at = ?, state_version = state_version + 1
				WHERE id = ? AND site_id = ?
			`, now, environmentID, siteID); err != nil {
				return fmt.Errorf("mark environment active: %w", err)
			}
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark site active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark deploy/update job succeeded: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkDeployOrUpdateFailed(ctx context.Context, siteID, environmentID, jobID, errorCode, errorMessage string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	if strings.TrimSpace(errorCode) == "" {
		errorCode = "ENV_MUTATION_FAILED"
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "environment mutation failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, environmentID, siteID); err != nil {
			return fmt.Errorf("mark environment failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark site failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
			WHERE id = ?
		`, now, strings.TrimSpace(errorCode), truncateMessage(errorMessage), now, jobID); err != nil {
			return fmt.Errorf("mark deploy/update job failed: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkRestoreSucceeded(ctx context.Context, siteID, environmentID, jobID string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, environmentID, siteID); err != nil {
			return fmt.Errorf("mark restored environment active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark restored site active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark restore job succeeded: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkRestoreFailed(ctx context.Context, siteID, environmentID, jobID, errorCode, errorMessage string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	if strings.TrimSpace(errorCode) == "" {
		errorCode = "ENV_RESTORE_FAILED"
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "environment restore failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, environmentID, siteID); err != nil {
			return fmt.Errorf("mark restored environment failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark restored site failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
			WHERE id = ?
		`, now, strings.TrimSpace(errorCode), truncateMessage(errorMessage), now, jobID); err != nil {
			return fmt.Errorf("mark restore job failed: %w", err)
		}

		return nil
	})
}

type scanner interface {
	Scan(dest ...any) error
}

func scanEnvironment(scanner scanner) (Environment, error) {
	var env Environment
	var sourceID sql.NullString
	var primaryDomainID sql.NullString
	var currentReleaseID sql.NullString
	var driftCheckedAt sql.NullString
	var lastDriftCheckID sql.NullString
	var fastcgi int
	var redis int

	if err := scanner.Scan(
		&env.ID,
		&env.SiteID,
		&env.Name,
		&env.Slug,
		&env.EnvironmentType,
		&env.Status,
		&env.NodeID,
		&sourceID,
		&env.PromotionPreset,
		&env.PreviewURL,
		&primaryDomainID,
		&currentReleaseID,
		&env.DriftStatus,
		&driftCheckedAt,
		&lastDriftCheckID,
		&fastcgi,
		&redis,
		&env.CreatedAt,
		&env.UpdatedAt,
		&env.StateVersion,
	); err != nil {
		return Environment{}, err
	}

	if sourceID.Valid {
		env.SourceEnvironmentID = &sourceID.String
	}
	if primaryDomainID.Valid {
		env.PrimaryDomainID = &primaryDomainID.String
	}
	if currentReleaseID.Valid {
		env.CurrentReleaseID = &currentReleaseID.String
	}
	if driftCheckedAt.Valid {
		env.DriftCheckedAt = &driftCheckedAt.String
	}
	if lastDriftCheckID.Valid {
		env.LastDriftCheckID = &lastDriftCheckID.String
	}
	env.FastCGICacheEnabled = fastcgi == 1
	env.RedisCacheEnabled = redis == 1

	return env, nil
}

func validateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.Slug) == "" {
		return ErrInvalidInput
	}

	if input.EnvironmentType != "staging" && input.EnvironmentType != "clone" {
		return ErrInvalidInput
	}
	if input.PromotionPreset != "content-protect" && input.PromotionPreset != "commerce-protect" {
		return ErrInvalidInput
	}
	if input.SourceEnvironmentID == nil || strings.TrimSpace(*input.SourceEnvironmentID) == "" {
		return ErrInvalidInput
	}

	return nil
}

func validateDeployInput(input DeployInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.SourceRef) == "" {
		return ErrInvalidInput
	}
	if input.SourceType != "git" && input.SourceType != "upload" {
		return ErrInvalidInput
	}

	return nil
}

func validateUpdatesInput(input UpdatesInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return ErrInvalidInput
	}

	switch input.Scope {
	case "core", "plugins", "themes", "all":
		return nil
	default:
		return ErrInvalidInput
	}
}

func validateRestoreInput(input RestoreInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.BackupID) == "" {
		return ErrInvalidInput
	}

	return nil
}

func validateCacheToggleInput(input CacheToggleInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return ErrInvalidInput
	}

	if input.FastCGICacheEnabled == nil && input.RedisCacheEnabled == nil {
		return ErrInvalidInput
	}

	return nil
}

type environmentMutationRef struct {
	SiteID              string
	NodeID              string
	Status              string
	FastCGICacheEnabled bool
	RedisCacheEnabled   bool
}

func loadEnvironmentForMutation(ctx context.Context, tx *sql.Tx, environmentID string) (environmentMutationRef, error) {
	var ref environmentMutationRef
	var fastcgi int
	var redis int
	err := tx.QueryRowContext(ctx, `
		SELECT site_id, node_id, status, fastcgi_cache_enabled, redis_cache_enabled
		FROM environments
		WHERE id = ?
		LIMIT 1
	`, environmentID).Scan(&ref.SiteID, &ref.NodeID, &ref.Status, &fastcgi, &redis)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return environmentMutationRef{}, ErrEnvironmentNotFound
		}
		return environmentMutationRef{}, fmt.Errorf("query environment for mutation: %w", err)
	}

	if ref.Status != "active" {
		return environmentMutationRef{}, ErrEnvironmentNotActive
	}

	ref.FastCGICacheEnabled = fastcgi == 1
	ref.RedisCacheEnabled = redis == 1

	return ref, nil
}

func boolToSQLiteInt(value bool) int {
	if value {
		return 1
	}

	return 0
}

func markEnvironmentAndSiteDeploying(ctx context.Context, tx *sql.Tx, environmentID string, ref environmentMutationRef, now string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE environments
		SET status = 'deploying', updated_at = ?, state_version = state_version + 1
		WHERE id = ?
	`, now, environmentID); err != nil {
		return fmt.Errorf("mark environment deploying: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE sites
		SET status = 'deploying', updated_at = ?, state_version = state_version + 1
		WHERE id = ?
	`, now, ref.SiteID); err != nil {
		return fmt.Errorf("mark site deploying: %w", err)
	}

	return nil
}

func markEnvironmentAndSiteRestoring(ctx context.Context, tx *sql.Tx, environmentID string, ref environmentMutationRef, now string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE environments
		SET status = 'restoring', updated_at = ?, state_version = state_version + 1
		WHERE id = ?
	`, now, environmentID); err != nil {
		return fmt.Errorf("mark environment restoring: %w", err)
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE sites
		SET status = 'restoring', updated_at = ?, state_version = state_version + 1
		WHERE id = ?
	`, now, ref.SiteID); err != nil {
		return fmt.Errorf("mark site restoring: %w", err)
	}

	return nil
}

func ensureFreshPreUpdateBackup(ctx context.Context, tx *sql.Tx, environmentID string, now time.Time) (string, error) {
	var backupID string
	var completedAt sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, completed_at
		FROM backups
		WHERE environment_id = ? AND status = 'completed'
		ORDER BY completed_at DESC
		LIMIT 1
	`, environmentID).Scan(&backupID, &completedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("query latest completed backup: %w", err)
	}

	if err == nil && completedAt.Valid {
		completed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(completedAt.String))
		if parseErr == nil && now.Sub(completed) <= preUpdateBackupFreshnessWindow {
			return backupID, nil
		}
	}

	newBackupID, err := newUUIDv4()
	if err != nil {
		return "", err
	}
	retentionUntil := now.Add(backupRetentionDays * 24 * time.Hour).Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'pending', 's3', ?, ?, NULL, NULL, ?, NULL)
	`, newBackupID, environmentID, buildStoragePath(environmentID, newBackupID), retentionUntil, nowStr); err != nil {
		return "", fmt.Errorf("insert pre-update backup record: %w", err)
	}

	return newBackupID, nil
}

func assertRestorableBackup(ctx context.Context, tx *sql.Tx, environmentID, backupID string) (string, error) {
	var resolvedBackupID string
	var status string
	err := tx.QueryRowContext(ctx, `
		SELECT id, status
		FROM backups
		WHERE id = ? AND environment_id = ?
		LIMIT 1
	`, strings.TrimSpace(backupID), environmentID).Scan(&resolvedBackupID, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrBackupNotFound
		}
		return "", fmt.Errorf("query restore backup: %w", err)
	}

	if status != "completed" {
		return "", ErrBackupNotCompleted
	}

	return resolvedBackupID, nil
}

func ensureFreshPreRestoreBackup(ctx context.Context, tx *sql.Tx, environmentID, restoreBackupID string, now time.Time) (string, error) {
	var backupID string
	var completedAt sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT id, completed_at
		FROM backups
		WHERE environment_id = ? AND status = 'completed' AND backup_scope = 'full' AND id != ?
		ORDER BY completed_at DESC
		LIMIT 1
	`, environmentID, restoreBackupID).Scan(&backupID, &completedAt)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return "", fmt.Errorf("query latest completed pre-restore backup: %w", err)
	}

	if err == nil && completedAt.Valid {
		completed, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(completedAt.String))
		if parseErr == nil && now.Sub(completed) <= preUpdateBackupFreshnessWindow {
			return backupID, nil
		}
	}

	newBackupID, err := newUUIDv4()
	if err != nil {
		return "", err
	}
	retentionUntil := now.Add(backupRetentionDays * 24 * time.Hour).Format(time.RFC3339)
	nowStr := now.Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'pending', 's3', ?, ?, NULL, NULL, ?, NULL)
	`, newBackupID, environmentID, buildStoragePath(environmentID, newBackupID), retentionUntil, nowStr); err != nil {
		return "", fmt.Errorf("insert pre-restore backup record: %w", err)
	}

	return newBackupID, nil
}

func buildStoragePath(environmentID, backupID string) string {
	return fmt.Sprintf("s3://pressluft/backups/%s/%s.tar.zst", environmentID, backupID)
}

func pendingReleasePath(releaseID string) string {
	return fmt.Sprintf("/var/www/sites/releases/%s", releaseID)
}

func truncateMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "environment mutation failed"
	}
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func assertSiteExists(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, siteID string) error {
	var exists int
	if err := q.QueryRowContext(ctx, `SELECT 1 FROM sites WHERE id = ? LIMIT 1`, siteID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrSiteNotFound
		}
		return fmt.Errorf("query site: %w", err)
	}

	return nil
}

func assertSourceEnvironment(ctx context.Context, tx *sql.Tx, siteID string, sourceEnvironmentID *string) error {
	if sourceEnvironmentID == nil {
		return ErrInvalidInput
	}

	var exists int
	if err := tx.QueryRowContext(ctx, `
		SELECT 1
		FROM environments
		WHERE id = ? AND site_id = ?
		LIMIT 1
	`, strings.TrimSpace(*sourceEnvironmentID), siteID).Scan(&exists); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrEnvironmentNotFound
		}
		return fmt.Errorf("query source environment: %w", err)
	}

	return nil
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
		return "", "", fmt.Errorf("select node for environment create: %w", err)
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

func valueOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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
