package stores

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/shared/idutil"
)

const (
	SiteStatusDraft     = "draft"
	SiteStatusActive    = "active"
	SiteStatusAttention = "attention"
	SiteStatusArchived  = "archived"

	SiteDeploymentStatePending   = "pending"
	SiteDeploymentStateDeploying = "deploying"
	SiteDeploymentStateReady     = "ready"
	SiteDeploymentStateFailed    = "failed"

	SiteRuntimeHealthStatePending = "pending"
	SiteRuntimeHealthStateHealthy = "healthy"
	SiteRuntimeHealthStateIssue   = "issue"
	SiteRuntimeHealthStateUnknown = "unknown"
)

type StoredSite struct {
	ID                  string `json:"id"`
	ServerID            string `json:"server_id"`
	ServerName          string `json:"server_name"`
	Name                string `json:"name"`
	WordPressAdminEmail string `json:"wordpress_admin_email,omitempty"`
	PrimaryDomain       string `json:"primary_domain,omitempty"`
	Status              string `json:"status"`
	DeploymentState     string `json:"deployment_state"`
	DeploymentStatus    string `json:"deployment_status_message,omitempty"`
	LastDeployJobID     string `json:"last_deploy_job_id,omitempty"`
	LastDeployedAt      string `json:"last_deployed_at,omitempty"`
	RuntimeHealthState  string `json:"runtime_health_state"`
	RuntimeHealthStatus string `json:"runtime_health_status_message,omitempty"`
	LastHealthCheckAt   string `json:"last_health_check_at,omitempty"`
	WordPressPath       string `json:"wordpress_path,omitempty"`
	PHPVersion          string `json:"php_version,omitempty"`
	WordPressVersion    string `json:"wordpress_version,omitempty"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type CreateSiteInput struct {
	ServerID              string
	Name                  string
	WordPressAdminEmail   string
	PrimaryDomain         string
	PrimaryHostnameConfig *CreateSitePrimaryHostnameInput
	Status                string
	WordPressPath         string
	PHPVersion            string
	WordPressVersion      string
}

type CreateSitePrimaryHostnameInput struct {
	Source   string
	Hostname string
	Label    string
	DomainID string
}

type UpdateSiteInput struct {
	Name                *string
	WordPressAdminEmail *string
	PrimaryDomain       *string
	Status              *string
	WordPressPath       *string
	PHPVersion          *string
	WordPressVersion    *string
	ServerID            *string
}

type SiteStore struct {
	db *sql.DB
}

func NewSiteStore(db *sql.DB) *SiteStore {
	return &SiteStore{db: db}
}

func AllSiteStatuses() []string {
	return []string{SiteStatusDraft, SiteStatusActive, SiteStatusAttention, SiteStatusArchived}
}

func AllSiteDeploymentStates() []string {
	return []string{SiteDeploymentStatePending, SiteDeploymentStateDeploying, SiteDeploymentStateReady, SiteDeploymentStateFailed}
}

func AllSiteRuntimeHealthStates() []string {
	return []string{SiteRuntimeHealthStatePending, SiteRuntimeHealthStateHealthy, SiteRuntimeHealthStateIssue, SiteRuntimeHealthStateUnknown}
}

func NormalizeSiteStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	switch status {
	case SiteStatusDraft, SiteStatusActive, SiteStatusAttention, SiteStatusArchived:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported site status %q", raw)
	}
}

func NormalizeSiteDeploymentState(raw string) (string, error) {
	state := strings.TrimSpace(raw)
	switch state {
	case SiteDeploymentStatePending, SiteDeploymentStateDeploying, SiteDeploymentStateReady, SiteDeploymentStateFailed:
		return state, nil
	default:
		return "", fmt.Errorf("unsupported site deployment_state %q", raw)
	}
}

func NormalizeSiteRuntimeHealthState(raw string) (string, error) {
	state := strings.TrimSpace(raw)
	switch state {
	case SiteRuntimeHealthStatePending, SiteRuntimeHealthStateHealthy, SiteRuntimeHealthStateIssue, SiteRuntimeHealthStateUnknown:
		return state, nil
	default:
		return "", fmt.Errorf("unsupported site runtime_health_state %q", raw)
	}
}

func (s *SiteStore) Create(ctx context.Context, in CreateSiteInput) (string, error) {
	if err := validateCreateSiteInput(in); err != nil {
		return "", err
	}
	serverID, err := idutil.Normalize(in.ServerID)
	if err != nil {
		return "", fmt.Errorf("server_id: %w", err)
	}
	if err := s.ensureServerExists(ctx, serverID); err != nil {
		return "", err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	publicID, err := idutil.New()
	if err != nil {
		return "", err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin create site tx: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sites (id, server_id, name, wordpress_admin_email, primary_domain, status, deployment_state, deployment_status_message, last_deploy_job_id, last_deployed_at, runtime_health_state, runtime_health_status_message, last_health_check_at, wordpress_path, php_version, wordpress_version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		publicID,
		serverID,
		strings.TrimSpace(in.Name),
		strings.TrimSpace(in.WordPressAdminEmail),
		nil,
		strings.TrimSpace(in.Status),
		SiteDeploymentStatePending,
		nil,
		nil,
		nil,
		SiteRuntimeHealthStatePending,
		"Waiting for the first runtime health check.",
		nil,
		nullableString(in.WordPressPath),
		nullableString(in.PHPVersion),
		nullableString(in.WordPressVersion),
		now,
		now,
	)
	if err != nil {
		return "", fmt.Errorf("insert site: %w", err)
	}
	if input, ok := resolveCreateSitePrimaryHostnameInput(in); ok {
		if _, err := NewDomainStore(s.db).createWithTx(ctx, tx, publicID, serverID, input); err != nil {
			return "", err
		}
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit create site tx: %w", err)
	}
	return publicID, nil
}

func (s *SiteStore) List(ctx context.Context) ([]StoredSite, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(si.wordpress_admin_email, ''), COALESCE(dom.hostname, si.primary_domain), si.status, si.deployment_state, COALESCE(si.deployment_status_message, ''), COALESCE(si.last_deploy_job_id, ''), COALESCE(si.last_deployed_at, ''), COALESCE(si.runtime_health_state, 'pending'), COALESCE(si.runtime_health_status_message, ''), COALESCE(si.last_health_check_at, ''), si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
		 LEFT JOIN domains dom ON dom.site_id = si.id AND dom.is_primary = 1
		 ORDER BY si.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()
	return scanSites(rows)
}

func (s *SiteStore) ListByServer(ctx context.Context, serverID string) ([]StoredSite, error) {
	normalized, err := idutil.Normalize(serverID)
	if err != nil {
		return nil, fmt.Errorf("server_id: %w", err)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(si.wordpress_admin_email, ''), COALESCE(dom.hostname, si.primary_domain), si.status, si.deployment_state, COALESCE(si.deployment_status_message, ''), COALESCE(si.last_deploy_job_id, ''), COALESCE(si.last_deployed_at, ''), COALESCE(si.runtime_health_state, 'pending'), COALESCE(si.runtime_health_status_message, ''), COALESCE(si.last_health_check_at, ''), si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
		 LEFT JOIN domains dom ON dom.site_id = si.id AND dom.is_primary = 1
		 WHERE si.server_id = ?
		 ORDER BY si.created_at DESC`,
		normalized,
	)
	if err != nil {
		return nil, fmt.Errorf("list sites by server: %w", err)
	}
	defer rows.Close()
	return scanSites(rows)
}

func (s *SiteStore) GetByID(ctx context.Context, id string) (*StoredSite, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	var (
		site                StoredSite
		primaryDomain       sql.NullString
		deploymentStatus    sql.NullString
		lastDeployJobID     sql.NullString
		lastDeployedAt      sql.NullString
		runtimeHealthState  sql.NullString
		runtimeHealthStatus sql.NullString
		lastHealthCheckAt   sql.NullString
		wordpressPath       sql.NullString
		phpVersion          sql.NullString
		wordpressVersion    sql.NullString
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(si.wordpress_admin_email, ''), COALESCE(dom.hostname, si.primary_domain), si.status, si.deployment_state, COALESCE(si.deployment_status_message, ''), COALESCE(si.last_deploy_job_id, ''), COALESCE(si.last_deployed_at, ''), COALESCE(si.runtime_health_state, 'pending'), COALESCE(si.runtime_health_status_message, ''), COALESCE(si.last_health_check_at, ''), si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
		 LEFT JOIN domains dom ON dom.site_id = si.id AND dom.is_primary = 1
		 WHERE si.id = ?`,
		publicID,
	).Scan(
		&site.ID,
		&site.ServerID,
		&site.ServerName,
		&site.Name,
		&site.WordPressAdminEmail,
		&primaryDomain,
		&site.Status,
		&site.DeploymentState,
		&deploymentStatus,
		&lastDeployJobID,
		&lastDeployedAt,
		&runtimeHealthState,
		&runtimeHealthStatus,
		&lastHealthCheckAt,
		&wordpressPath,
		&phpVersion,
		&wordpressVersion,
		&site.CreatedAt,
		&site.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("site %s not found", publicID)
		}
		return nil, fmt.Errorf("get site: %w", err)
	}
	site.PrimaryDomain = nullStringValue(primaryDomain)
	site.DeploymentStatus = nullStringValue(deploymentStatus)
	site.LastDeployJobID = nullStringValue(lastDeployJobID)
	site.LastDeployedAt = nullStringValue(lastDeployedAt)
	site.RuntimeHealthState = nullStringValue(runtimeHealthState)
	site.RuntimeHealthStatus = nullStringValue(runtimeHealthStatus)
	site.LastHealthCheckAt = nullStringValue(lastHealthCheckAt)
	site.WordPressPath = nullStringValue(wordpressPath)
	site.PHPVersion = nullStringValue(phpVersion)
	site.WordPressVersion = nullStringValue(wordpressVersion)
	if _, err := NormalizeSiteStatus(site.Status); err != nil {
		return nil, fmt.Errorf("get site status: %w", err)
	}
	if _, err := NormalizeSiteDeploymentState(site.DeploymentState); err != nil {
		return nil, fmt.Errorf("get site deployment state: %w", err)
	}
	if _, err := NormalizeSiteRuntimeHealthState(site.RuntimeHealthState); err != nil {
		return nil, fmt.Errorf("get site runtime health state: %w", err)
	}
	return &site, nil
}

func (s *SiteStore) UpdateDeployment(ctx context.Context, siteID, deploymentState, deploymentStatus, lastDeployJobID, lastDeployedAt string) error {
	publicID, err := idutil.Normalize(siteID)
	if err != nil {
		return err
	}
	deploymentState, err = NormalizeSiteDeploymentState(deploymentState)
	if err != nil {
		return err
	}
	lastDeployJobID = strings.TrimSpace(lastDeployJobID)
	lastDeployedAt = strings.TrimSpace(lastDeployedAt)
	if lastDeployedAt != "" {
		if _, err := time.Parse(time.RFC3339, lastDeployedAt); err != nil {
			return fmt.Errorf("last_deployed_at: %w", err)
		}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE sites
		 SET deployment_state = ?, deployment_status_message = ?, last_deploy_job_id = ?, last_deployed_at = ?, updated_at = ?
		 WHERE id = ?`,
		deploymentState,
		nullableString(deploymentStatus),
		nullableString(lastDeployJobID),
		nullableString(lastDeployedAt),
		time.Now().UTC().Format(time.RFC3339),
		publicID,
	)
	if err != nil {
		return fmt.Errorf("update site deployment: %w", err)
	}
	return nil
}

func (s *SiteStore) UpdateRuntimeHealth(ctx context.Context, siteID, runtimeHealthState, runtimeHealthStatus, lastHealthCheckAt string) error {
	publicID, err := idutil.Normalize(siteID)
	if err != nil {
		return err
	}
	runtimeHealthState, err = NormalizeSiteRuntimeHealthState(runtimeHealthState)
	if err != nil {
		return err
	}
	lastHealthCheckAt = strings.TrimSpace(lastHealthCheckAt)
	if lastHealthCheckAt != "" {
		if _, err := time.Parse(time.RFC3339, lastHealthCheckAt); err != nil {
			return fmt.Errorf("last_health_check_at: %w", err)
		}
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE sites
		 SET runtime_health_state = ?, runtime_health_status_message = ?, last_health_check_at = ?, updated_at = ?
		 WHERE id = ?`,
		runtimeHealthState,
		nullableString(runtimeHealthStatus),
		nullableString(lastHealthCheckAt),
		time.Now().UTC().Format(time.RFC3339),
		publicID,
	)
	if err != nil {
		return fmt.Errorf("update site runtime health: %w", err)
	}
	return nil
}

func (s *SiteStore) Update(ctx context.Context, id string, in UpdateSiteInput) (*StoredSite, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	if err := validateUpdateSiteInput(in); err != nil {
		return nil, err
	}
	current, err := s.GetByID(ctx, publicID)
	if err != nil {
		return nil, err
	}
	serverID := current.ServerID
	if in.ServerID != nil {
		serverID, err = idutil.Normalize(strings.TrimSpace(*in.ServerID))
		if err != nil {
			return nil, fmt.Errorf("server_id: %w", err)
		}
		if err := s.ensureServerExists(ctx, serverID); err != nil {
			return nil, err
		}
	}
	name := current.Name
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
	}
	wordpressAdminEmail := current.WordPressAdminEmail
	if in.WordPressAdminEmail != nil {
		wordpressAdminEmail = strings.TrimSpace(*in.WordPressAdminEmail)
	}
	primaryDomain := current.PrimaryDomain
	if in.PrimaryDomain != nil {
		primaryDomain = current.PrimaryDomain
	}
	status := current.Status
	if in.Status != nil {
		status = strings.TrimSpace(*in.Status)
	}
	wordpressPath := current.WordPressPath
	if in.WordPressPath != nil {
		wordpressPath = strings.TrimSpace(*in.WordPressPath)
	}
	phpVersion := current.PHPVersion
	if in.PHPVersion != nil {
		phpVersion = strings.TrimSpace(*in.PHPVersion)
	}
	wordpressVersion := current.WordPressVersion
	if in.WordPressVersion != nil {
		wordpressVersion = strings.TrimSpace(*in.WordPressVersion)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update site tx: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx,
		`UPDATE sites
		 SET server_id = ?, name = ?, wordpress_admin_email = ?, primary_domain = ?, status = ?, wordpress_path = ?, php_version = ?, wordpress_version = ?, updated_at = ?
		 WHERE id = ?`,
		serverID,
		name,
		wordpressAdminEmail,
		nullableString(primaryDomain),
		status,
		nullableString(wordpressPath),
		nullableString(phpVersion),
		nullableString(wordpressVersion),
		now,
		publicID,
	)
	if err != nil {
		return nil, fmt.Errorf("update site: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, fmt.Errorf("site %s not found", publicID)
	}
	if in.PrimaryDomain != nil {
		domainStore := NewDomainStore(s.db)
		primaryDomain = strings.TrimSpace(*in.PrimaryDomain)
		if primaryDomain == "" {
			if err := domainStore.clearPrimaryHostnameForSiteTx(ctx, tx, publicID); err != nil {
				return nil, err
			}
		} else {
			if err := domainStore.setPrimaryHostnameForSiteTx(ctx, tx, publicID, primaryDomain, DomainSourceUser); err != nil {
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit update site tx: %w", err)
	}
	return s.GetByID(ctx, publicID)
}

func (s *SiteStore) Delete(ctx context.Context, id string) error {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete site tx: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `DELETE FROM domains WHERE site_id = ?`, publicID); err != nil {
		return fmt.Errorf("delete site domains: %w", err)
	}
	res, err := tx.ExecContext(ctx, `DELETE FROM sites WHERE id = ?`, publicID)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("site %s not found", publicID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete site tx: %w", err)
	}
	return nil
}

func (s *SiteStore) ensureServerExists(ctx context.Context, serverID string) error {
	var exists string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM servers WHERE id = ?`, serverID).Scan(&exists); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("server %s not found", serverID)
		}
		return fmt.Errorf("lookup server id: %w", err)
	}
	return nil
}

func scanSites(rows *sql.Rows) ([]StoredSite, error) {
	var out []StoredSite
	for rows.Next() {
		var (
			site                StoredSite
			wordpressAdminEmail sql.NullString
			primaryDomain       sql.NullString
			deploymentStatus    sql.NullString
			lastDeployJobID     sql.NullString
			lastDeployedAt      sql.NullString
			runtimeHealthState  sql.NullString
			runtimeHealthStatus sql.NullString
			lastHealthCheckAt   sql.NullString
			wordpressPath       sql.NullString
			phpVersion          sql.NullString
			wordpressVersion    sql.NullString
		)
		if err := rows.Scan(
			&site.ID,
			&site.ServerID,
			&site.ServerName,
			&site.Name,
			&wordpressAdminEmail,
			&primaryDomain,
			&site.Status,
			&site.DeploymentState,
			&deploymentStatus,
			&lastDeployJobID,
			&lastDeployedAt,
			&runtimeHealthState,
			&runtimeHealthStatus,
			&lastHealthCheckAt,
			&wordpressPath,
			&phpVersion,
			&wordpressVersion,
			&site.CreatedAt,
			&site.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan site: %w", err)
		}
		if _, err := NormalizeSiteStatus(site.Status); err != nil {
			return nil, fmt.Errorf("scan site status: %w", err)
		}
		if _, err := NormalizeSiteDeploymentState(site.DeploymentState); err != nil {
			return nil, fmt.Errorf("scan site deployment state: %w", err)
		}
		site.RuntimeHealthState = nullStringValue(runtimeHealthState)
		if _, err := NormalizeSiteRuntimeHealthState(site.RuntimeHealthState); err != nil {
			return nil, fmt.Errorf("scan site runtime health state: %w", err)
		}
		site.PrimaryDomain = nullStringValue(primaryDomain)
		site.WordPressAdminEmail = nullStringValue(wordpressAdminEmail)
		site.DeploymentStatus = nullStringValue(deploymentStatus)
		site.LastDeployJobID = nullStringValue(lastDeployJobID)
		site.LastDeployedAt = nullStringValue(lastDeployedAt)
		site.RuntimeHealthStatus = nullStringValue(runtimeHealthStatus)
		site.LastHealthCheckAt = nullStringValue(lastHealthCheckAt)
		site.WordPressPath = nullStringValue(wordpressPath)
		site.PHPVersion = nullStringValue(phpVersion)
		site.WordPressVersion = nullStringValue(wordpressVersion)
		out = append(out, site)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sites: %w", err)
	}
	return out, nil
}
