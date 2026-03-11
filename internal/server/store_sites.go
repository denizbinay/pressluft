package server

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
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
	ServerID            string
	Name                string
	PrimaryDomain       string
	PrimaryDomainConfig *CreateSitePrimaryDomainInput
	Status              string
	WordPressPath       string
	PHPVersion          string
	WordPressVersion    string
}

type CreateSitePrimaryDomainInput struct {
	Mode           string
	Hostname       string
	Label          string
	ParentDomainID string
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
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin create site tx: %w", err)
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx,
		`INSERT INTO sites (id, server_id, name, primary_domain, status, wordpress_path, php_version, wordpress_version, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		publicID,
		serverID,
		strings.TrimSpace(in.Name),
		nil,
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
	if input, ok := resolveCreateSitePrimaryDomainInput(in); ok {
		if _, err := NewDomainStore(s.db).createWithTx(ctx, tx, publicID, input); err != nil {
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
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(dom.hostname, si.primary_domain), si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
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
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(dom.hostname, si.primary_domain), si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
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
		site             StoredSite
		primaryDomain    sql.NullString
		wordpressPath    sql.NullString
		phpVersion       sql.NullString
		wordpressVersion sql.NullString
	)
	err = s.db.QueryRowContext(ctx,
		`SELECT si.id, si.server_id, srv.name, si.name, COALESCE(dom.hostname, si.primary_domain), si.status, si.wordpress_path, si.php_version, si.wordpress_version, si.created_at, si.updated_at
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
	if in.PrimaryDomain != nil {
		domainStore := NewDomainStore(s.db)
		primaryDomain = strings.TrimSpace(*in.PrimaryDomain)
		if primaryDomain == "" {
			if err := domainStore.clearPrimaryHostnameForSiteTx(ctx, tx, publicID); err != nil {
				return nil, err
			}
		} else {
			if err := domainStore.setPrimaryHostnameForSiteTx(ctx, tx, publicID, primaryDomain, DomainOwnershipCustomer); err != nil {
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

func validateCreateSiteInput(in CreateSiteInput) error {
	if _, err := idutil.Normalize(in.ServerID); err != nil {
		return fmt.Errorf("server_id: %w", err)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.PrimaryDomain) != "" && in.PrimaryDomainConfig != nil {
		return fmt.Errorf("use either primary_domain or primary_domain_config, not both")
	}
	if strings.TrimSpace(in.PrimaryDomain) != "" {
		if _, err := normalizeHostname(in.PrimaryDomain); err != nil {
			return err
		}
	}
	if in.PrimaryDomainConfig != nil {
		if err := validateCreateSitePrimaryDomainInput(*in.PrimaryDomainConfig); err != nil {
			return err
		}
	}
	if _, err := NormalizeSiteStatus(in.Status); err != nil {
		return err
	}
	return nil
}

func validateCreateSitePrimaryDomainInput(in CreateSitePrimaryDomainInput) error {
	mode := strings.TrimSpace(in.Mode)
	switch mode {
	case "wildcard":
		if strings.TrimSpace(in.Label) == "" {
			return fmt.Errorf("primary_domain_config.label is required for wildcard domains")
		}
		if strings.TrimSpace(in.ParentDomainID) == "" {
			return fmt.Errorf("primary_domain_config.parent_domain_id is required for wildcard domains")
		}
		if _, err := normalizeDomainLabel(strings.TrimSpace(in.Label)); err != nil {
			return err
		}
		return nil
	case "direct":
		if _, err := normalizeHostname(strings.TrimSpace(in.Hostname)); err != nil {
			return err
		}
	default:
		return fmt.Errorf("primary_domain_config.mode must be direct or wildcard")
	}
	return nil
}

func resolveCreateSitePrimaryDomainInput(in CreateSiteInput) (CreateSitePrimaryDomainInput, bool) {
	if in.PrimaryDomainConfig != nil {
		return *in.PrimaryDomainConfig, true
	}
	if strings.TrimSpace(in.PrimaryDomain) == "" {
		return CreateSitePrimaryDomainInput{}, false
	}
	return CreateSitePrimaryDomainInput{
		Mode:     "direct",
		Hostname: strings.TrimSpace(in.PrimaryDomain),
	}, true
}

func (s *DomainStore) createWithTx(ctx context.Context, tx *sql.Tx, siteID string, input CreateSitePrimaryDomainInput) (string, error) {
	mode := strings.TrimSpace(input.Mode)
	switch mode {
	case "wildcard":
		parent, err := s.getByIDTx(ctx, tx, strings.TrimSpace(input.ParentDomainID))
		if err != nil {
			return "", fmt.Errorf("primary_domain_config.parent_domain_id: %w", err)
		}
		hostname, err := buildWildcardChildHostname(strings.TrimSpace(input.Label), *parent)
		if err != nil {
			return "", err
		}
		return s.createTx(ctx, tx, CreateDomainInput{
			Hostname:       hostname,
			Kind:           DomainKindDirect,
			Ownership:      parent.Ownership,
			Status:         DomainStatusActive,
			SiteID:         siteID,
			ParentDomainID: strings.TrimSpace(input.ParentDomainID),
			IsPrimary:      true,
		})
	case "direct":
		return s.createTx(ctx, tx, CreateDomainInput{
			Hostname:  strings.TrimSpace(input.Hostname),
			Kind:      DomainKindDirect,
			Ownership: DomainOwnershipCustomer,
			Status:    DomainStatusActive,
			SiteID:    siteID,
			IsPrimary: true,
		})
	default:
		return "", fmt.Errorf("primary_domain_config.mode must be direct or wildcard")
	}
}

func buildWildcardChildHostname(label string, parent StoredDomain) (string, error) {
	normalizedLabel, err := normalizeDomainLabel(label)
	if err != nil {
		return "", err
	}
	if parent.Kind != DomainKindWildcard {
		return "", fmt.Errorf("primary_domain_config.parent_domain_id must reference a wildcard domain")
	}
	if parent.Status != DomainStatusActive {
		return "", fmt.Errorf("primary_domain_config.parent_domain_id must reference an active wildcard domain")
	}
	return normalizeHostname(normalizedLabel + "." + parent.Hostname)
}

func normalizeDomainLabel(label string) (string, error) {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.ReplaceAll(label, "_", "-")
	label = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(label, "-")
	label = strings.Trim(label, "-")
	if label == "" {
		return "", fmt.Errorf("primary_domain_config.label is required for wildcard domains")
	}
	if strings.Contains(label, ".") {
		return "", fmt.Errorf("primary_domain_config.label must be a single subdomain label")
	}
	if len(label) > 63 {
		return "", fmt.Errorf("primary_domain_config.label must be 63 characters or fewer")
	}
	return label, nil
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
	if in.PrimaryDomain != nil && strings.TrimSpace(*in.PrimaryDomain) != "" {
		if _, err := normalizeHostname(*in.PrimaryDomain); err != nil {
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
