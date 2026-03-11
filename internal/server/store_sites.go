package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/idutil"
)

const (
	SiteStatusDraft     = "draft"
	SiteStatusActive    = "active"
	SiteStatusAttention = "attention"
	SiteStatusArchived  = "archived"
)

type StoredSite struct {
	ID               string `json:"id"`
	ServerID         string `json:"server_id"`
	ServerName       string `json:"server_name"`
	Name             string `json:"name"`
	PrimaryDomain    string `json:"primary_domain,omitempty"`
	Status           string `json:"status"`
	WordPressPath    string `json:"wordpress_path,omitempty"`
	PHPVersion       string `json:"php_version,omitempty"`
	WordPressVersion string `json:"wordpress_version,omitempty"`
	CreatedAt        string `json:"created_at"`
	UpdatedAt        string `json:"updated_at"`
}

type CreateSiteInput struct {
	ServerID         string
	Name             string
	PrimaryDomain    string
	Status           string
	WordPressPath    string
	PHPVersion       string
	WordPressVersion string
}

type UpdateSiteInput struct {
	Name             *string
	PrimaryDomain    *string
	Status           *string
	WordPressPath    *string
	PHPVersion       *string
	WordPressVersion *string
	ServerID         *string
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

func NormalizeSiteStatus(raw string) (string, error) {
	status := strings.TrimSpace(raw)
	switch status {
	case SiteStatusDraft, SiteStatusActive, SiteStatusAttention, SiteStatusArchived:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported site status %q", raw)
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
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO sites (id, server_id, name, primary_domain, status, wordpress_path, php_version, wordpress_version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		publicID,
		serverID,
		strings.TrimSpace(in.Name),
		nullableSiteString(in.PrimaryDomain),
		strings.TrimSpace(in.Status),
		nullableSiteString(in.WordPressPath),
		nullableSiteString(in.PHPVersion),
		nullableSiteString(in.WordPressVersion),
		now,
		now,
	)
	if err != nil {
		return "", fmt.Errorf("insert site: %w", err)
	}
	return publicID, nil
}

func (s *SiteStore) List(ctx context.Context) ([]StoredSite, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, si.primary_domain, si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
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
		`SELECT si.id, si.server_id, srv.name, si.name, si.primary_domain, si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
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
		site             StoredSite
		primaryDomain    sql.NullString
		wordpressPath    sql.NullString
		phpVersion       sql.NullString
		wordpressVersion sql.NullString
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, si.primary_domain, si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
		 FROM sites si
		 JOIN servers srv ON srv.id = si.server_id
		 WHERE si.id = ?`,
		publicID,
	).Scan(
		&site.ID,
		&site.ServerID,
		&site.ServerName,
		&site.Name,
		&primaryDomain,
		&site.Status,
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
	site.WordPressPath = nullStringValue(wordpressPath)
	site.PHPVersion = nullStringValue(phpVersion)
	site.WordPressVersion = nullStringValue(wordpressVersion)
	if _, err := NormalizeSiteStatus(site.Status); err != nil {
		return nil, fmt.Errorf("get site status: %w", err)
	}
	return &site, nil
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
	primaryDomain := current.PrimaryDomain
	if in.PrimaryDomain != nil {
		primaryDomain = strings.TrimSpace(*in.PrimaryDomain)
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
	res, err := s.db.ExecContext(ctx,
		`UPDATE sites
		 SET server_id = ?, name = ?, primary_domain = ?, status = ?, wordpress_path = ?, php_version = ?, wordpress_version = ?, updated_at = ?
		 WHERE id = ?`,
		serverID,
		name,
		nullableSiteString(primaryDomain),
		status,
		nullableSiteString(wordpressPath),
		nullableSiteString(phpVersion),
		nullableSiteString(wordpressVersion),
		now,
		publicID,
	)
	if err != nil {
		return nil, fmt.Errorf("update site: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, fmt.Errorf("site %s not found", publicID)
	}
	return s.GetByID(ctx, publicID)
}

func (s *SiteStore) Delete(ctx context.Context, id string) error {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return err
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM sites WHERE id = ?`, publicID)
	if err != nil {
		return fmt.Errorf("delete site: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("site %s not found", publicID)
	}
	return nil
}

func validateCreateSiteInput(in CreateSiteInput) error {
	if _, err := idutil.Normalize(in.ServerID); err != nil {
		return fmt.Errorf("server_id: %w", err)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if _, err := NormalizeSiteStatus(in.Status); err != nil {
		return err
	}
	return nil
}

func validateUpdateSiteInput(in UpdateSiteInput) error {
	if in.Name != nil && strings.TrimSpace(*in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if in.Status != nil {
		if _, err := NormalizeSiteStatus(*in.Status); err != nil {
			return err
		}
	}
	if in.ServerID != nil {
		if _, err := idutil.Normalize(strings.TrimSpace(*in.ServerID)); err != nil {
			return fmt.Errorf("server_id: %w", err)
		}
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

func nullableSiteString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func scanSites(rows *sql.Rows) ([]StoredSite, error) {
	var out []StoredSite
	for rows.Next() {
		var (
			site             StoredSite
			primaryDomain    sql.NullString
			wordpressPath    sql.NullString
			phpVersion       sql.NullString
			wordpressVersion sql.NullString
		)
		if err := rows.Scan(
			&site.ID,
			&site.ServerID,
			&site.ServerName,
			&site.Name,
			&primaryDomain,
			&site.Status,
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
		site.PrimaryDomain = nullStringValue(primaryDomain)
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
