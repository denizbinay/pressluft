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
	DomainKindHostname   = "hostname"
	DomainKindBaseDomain = "base_domain"

	DomainSourceUser             = "user"
	DomainSourceFallbackResolver = "fallback_resolver"

	DomainDNSStatePending  = "pending"
	DomainDNSStateReady    = "ready"
	DomainDNSStateIssue    = "issue"
	DomainDNSStateDisabled = "disabled"

	DomainRoutingStateNotConfigured = "not_configured"
	DomainRoutingStatePending       = "pending"
	DomainRoutingStateReady         = "ready"
	DomainRoutingStateIssue         = "issue"
)

type StoredDomain struct {
	ID                   string `json:"id"`
	Hostname             string `json:"hostname"`
	Kind                 string `json:"kind"`
	Source               string `json:"source"`
	DNSState             string `json:"dns_state"`
	RoutingState         string `json:"routing_state"`
	DNSStatusMessage     string `json:"dns_status_message,omitempty"`
	RoutingStatusMessage string `json:"routing_status_message,omitempty"`
	LastCheckedAt        string `json:"last_checked_at,omitempty"`
	SiteID               string `json:"site_id,omitempty"`
	SiteName             string `json:"site_name,omitempty"`
	ParentDomainID       string `json:"parent_domain_id,omitempty"`
	ParentHostname       string `json:"parent_hostname,omitempty"`
	IsPrimary            bool   `json:"is_primary"`
	CreatedAt            string `json:"created_at"`
	UpdatedAt            string `json:"updated_at"`
}

type CreateDomainInput struct {
	Hostname             string
	Kind                 string
	Source               string
	DNSState             string
	RoutingState         string
	DNSStatusMessage     string
	RoutingStatusMessage string
	LastCheckedAt        string
	SiteID               string
	ParentDomainID       string
	IsPrimary            bool
}

type UpdateDomainInput struct {
	Hostname             *string
	Kind                 *string
	Source               *string
	DNSState             *string
	RoutingState         *string
	DNSStatusMessage     *string
	RoutingStatusMessage *string
	LastCheckedAt        *string
	SiteID               *string
	ParentDomainID       *string
	IsPrimary            *bool
}

type DomainStore struct {
	db *sql.DB
}

func NewDomainStore(db *sql.DB) *DomainStore {
	return &DomainStore{db: db}
}

func (s *DomainStore) BackfillLegacyPrimaryDomains(ctx context.Context) error {
	type legacyDomain struct {
		siteID   string
		hostname string
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT si.id, si.primary_domain
		FROM sites si
		LEFT JOIN domains d ON d.site_id = si.id AND d.is_primary = 1
		WHERE si.primary_domain IS NOT NULL AND TRIM(si.primary_domain) != '' AND d.id IS NULL
	`)
	if err != nil {
		return fmt.Errorf("query legacy site domains: %w", err)
	}
	defer rows.Close()

	var pending []legacyDomain
	for rows.Next() {
		var item legacyDomain
		if err := rows.Scan(&item.siteID, &item.hostname); err != nil {
			return fmt.Errorf("scan legacy site domain: %w", err)
		}
		pending = append(pending, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate legacy site domains: %w", err)
	}
	for _, item := range pending {
		if _, err := s.Create(ctx, CreateDomainInput{
			Hostname:     item.hostname,
			Kind:         DomainKindHostname,
			Source:       DomainSourceUser,
			DNSState:     DomainDNSStatePending,
			RoutingState: DomainRoutingStatePending,
			SiteID:       item.siteID,
			IsPrimary:    true,
		}); err != nil && !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("backfill legacy domain for site %s: %w", item.siteID, err)
		}
	}
	return nil
}

func (s *DomainStore) Create(ctx context.Context, in CreateDomainInput) (string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("begin create domain tx: %w", err)
	}
	defer tx.Rollback()
	id, err := s.createTx(ctx, tx, in)
	if err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("commit create domain tx: %w", err)
	}
	return id, nil
}

func (s *DomainStore) createTx(ctx context.Context, tx *sql.Tx, in CreateDomainInput) (string, error) {
	prepared, err := prepareCreateDomainInput(in)
	if err != nil {
		return "", err
	}
	if err := s.ensureRelationsTx(ctx, tx, prepared.Hostname, prepared.Kind, prepared.Source, prepared.SiteID, prepared.ParentDomainID); err != nil {
		return "", err
	}
	publicID, err := idutil.New()
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Format(time.RFC3339)

	isPrimary := prepared.IsPrimary
	if prepared.SiteID != "" && !isPrimary {
		promote, err := shouldPromotePrimaryTx(ctx, tx, prepared.SiteID)
		if err != nil {
			return "", err
		}
		isPrimary = promote
	}
	if prepared.SiteID != "" && isPrimary {
		if err := clearPrimaryForSiteTx(ctx, tx, prepared.SiteID, ""); err != nil {
			return "", err
		}
	}
	_, err = tx.ExecContext(ctx, `
		INSERT INTO domains (
			id, hostname, kind, source, dns_state, routing_state, dns_status_message, routing_status_message,
			last_checked_at, site_id, parent_domain_id, is_primary, created_at, updated_at
		)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		publicID,
		prepared.Hostname,
		prepared.Kind,
		prepared.Source,
		prepared.DNSState,
		prepared.RoutingState,
		nullableString(prepared.DNSStatusMessage),
		nullableString(prepared.RoutingStatusMessage),
		nullableString(prepared.LastCheckedAt),
		nullableString(prepared.SiteID),
		nullableString(prepared.ParentDomainID),
		boolToInt(isPrimary),
		now,
		now,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return "", fmt.Errorf("hostname %q already exists", prepared.Hostname)
		}
		return "", fmt.Errorf("insert domain: %w", err)
	}
	if err := syncLegacySitePrimaryDomainTx(ctx, tx, prepared.SiteID); err != nil {
		return "", err
	}
	return publicID, nil
}

func (s *DomainStore) List(ctx context.Context) ([]StoredDomain, error) {
	rows, err := s.db.QueryContext(ctx, domainSelectQuery+` ORDER BY d.created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	defer rows.Close()
	return scanDomains(rows)
}

func (s *DomainStore) ListBySite(ctx context.Context, siteID string) ([]StoredDomain, error) {
	normalized, err := idutil.Normalize(siteID)
	if err != nil {
		return nil, fmt.Errorf("site_id: %w", err)
	}
	rows, err := s.db.QueryContext(ctx, domainSelectQuery+` WHERE d.site_id = ? ORDER BY d.is_primary DESC, d.created_at ASC`, normalized)
	if err != nil {
		return nil, fmt.Errorf("list domains by site: %w", err)
	}
	defer rows.Close()
	return scanDomains(rows)
}

func (s *DomainStore) GetByID(ctx context.Context, id string) (*StoredDomain, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, domainSelectQuery+` WHERE d.id = ?`, publicID)
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}
	defer rows.Close()
	domains, err := scanDomains(rows)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("domain %s not found", publicID)
	}
	return &domains[0], nil
}

func (s *DomainStore) UpdateRoutingStatus(ctx context.Context, domainID, routingState, routingStatusMessage string, checkedAt time.Time) error {
	publicID, err := idutil.Normalize(domainID)
	if err != nil {
		return err
	}
	routingState, err = normalizeDomainRoutingState(routingState)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE domains
		 SET routing_state = ?, routing_status_message = ?, last_checked_at = ?, updated_at = ?
		 WHERE id = ?`,
		routingState,
		nullableString(strings.TrimSpace(routingStatusMessage)),
		checkedAt.UTC().Format(time.RFC3339),
		time.Now().UTC().Format(time.RFC3339),
		publicID,
	)
	if err != nil {
		return fmt.Errorf("update domain routing status: %w", err)
	}
	return nil
}

func (s *DomainStore) Update(ctx context.Context, id string, in UpdateDomainInput) (*StoredDomain, error) {
	return s.updateTx(ctx, nil, id, in)
}

func (s *DomainStore) updateTx(ctx context.Context, tx *sql.Tx, id string, in UpdateDomainInput) (*StoredDomain, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	current, err := s.getByIDTx(ctx, tx, publicID)
	if err != nil {
		return nil, err
	}
	prepared, err := prepareUpdateDomainInput(*current, in)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRelationsTx(ctx, tx, prepared.Hostname, prepared.Kind, prepared.Source, prepared.SiteID, prepared.ParentDomainID); err != nil {
		return nil, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	ownedTx := false
	if tx == nil {
		tx, err = s.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin update domain tx: %w", err)
		}
		ownedTx = true
	}
	defer func() {
		if ownedTx {
			_ = tx.Rollback()
		}
	}()

	if prepared.SiteID != "" && prepared.IsPrimary {
		if err := clearPrimaryForSiteTx(ctx, tx, prepared.SiteID, publicID); err != nil {
			return nil, err
		}
	}
	res, err := tx.ExecContext(ctx, `
		UPDATE domains
		SET hostname = ?, kind = ?, source = ?, dns_state = ?, routing_state = ?, dns_status_message = ?,
			routing_status_message = ?, last_checked_at = ?, site_id = ?, parent_domain_id = ?, is_primary = ?, updated_at = ?
		WHERE id = ?
	`,
		prepared.Hostname,
		prepared.Kind,
		prepared.Source,
		prepared.DNSState,
		prepared.RoutingState,
		nullableString(prepared.DNSStatusMessage),
		nullableString(prepared.RoutingStatusMessage),
		nullableString(prepared.LastCheckedAt),
		nullableString(prepared.SiteID),
		nullableString(prepared.ParentDomainID),
		boolToInt(prepared.IsPrimary),
		now,
		publicID,
	)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			return nil, fmt.Errorf("hostname %q already exists", prepared.Hostname)
		}
		return nil, fmt.Errorf("update domain: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, fmt.Errorf("domain %s not found", publicID)
	}
	if prepared.SiteID != "" && !prepared.IsPrimary {
		promote, err := shouldPromotePrimaryTx(ctx, tx, prepared.SiteID)
		if err != nil {
			return nil, err
		}
		if promote {
			if _, err := tx.ExecContext(ctx, `UPDATE domains SET is_primary = 1, updated_at = ? WHERE id = ?`, now, publicID); err != nil {
				return nil, fmt.Errorf("promote domain to primary: %w", err)
			}
		}
	}
	if err := ensurePrimaryDomainTx(ctx, tx, current.SiteID); err != nil {
		return nil, err
	}
	if err := syncLegacySitePrimaryDomainTx(ctx, tx, current.SiteID); err != nil {
		return nil, err
	}
	if current.SiteID != prepared.SiteID {
		if err := ensurePrimaryDomainTx(ctx, tx, prepared.SiteID); err != nil {
			return nil, err
		}
		if err := syncLegacySitePrimaryDomainTx(ctx, tx, prepared.SiteID); err != nil {
			return nil, err
		}
	}
	if ownedTx {
		if err := tx.Commit(); err != nil {
			return nil, fmt.Errorf("commit update domain tx: %w", err)
		}
		return s.GetByID(ctx, publicID)
	}
	return s.getByIDTx(ctx, tx, publicID)
}

func (s *DomainStore) Delete(ctx context.Context, id string) error {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return err
	}
	current, err := s.GetByID(ctx, publicID)
	if err != nil {
		return err
	}
	if current.Kind == DomainKindBaseDomain {
		var childCount int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM domains WHERE parent_domain_id = ?`, publicID).Scan(&childCount); err != nil {
			return fmt.Errorf("count child hostnames: %w", err)
		}
		if childCount > 0 {
			return fmt.Errorf("base domains with attached child hostnames cannot be deleted")
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete domain tx: %w", err)
	}
	defer tx.Rollback()
	res, err := tx.ExecContext(ctx, `DELETE FROM domains WHERE id = ?`, publicID)
	if err != nil {
		return fmt.Errorf("delete domain: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return fmt.Errorf("domain %s not found", publicID)
	}
	if err := ensurePrimaryDomainTx(ctx, tx, current.SiteID); err != nil {
		return err
	}
	if err := syncLegacySitePrimaryDomainTx(ctx, tx, current.SiteID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete domain tx: %w", err)
	}
	return nil
}

const domainSelectQuery = `SELECT d.id, d.hostname, d.kind, d.source, d.dns_state, d.routing_state,
	COALESCE(d.dns_status_message, ''), COALESCE(d.routing_status_message, ''), COALESCE(d.last_checked_at, ''),
	COALESCE(d.site_id, ''), COALESCE(si.name, ''), COALESCE(d.parent_domain_id, ''), COALESCE(parent.hostname, ''),
	d.is_primary, d.created_at, d.updated_at
	FROM domains d
	LEFT JOIN sites si ON si.id = d.site_id
	LEFT JOIN domains parent ON parent.id = d.parent_domain_id`

func scanDomains(rows *sql.Rows) ([]StoredDomain, error) {
	var out []StoredDomain
	for rows.Next() {
		var domain StoredDomain
		var isPrimary int
		if err := rows.Scan(
			&domain.ID,
			&domain.Hostname,
			&domain.Kind,
			&domain.Source,
			&domain.DNSState,
			&domain.RoutingState,
			&domain.DNSStatusMessage,
			&domain.RoutingStatusMessage,
			&domain.LastCheckedAt,
			&domain.SiteID,
			&domain.SiteName,
			&domain.ParentDomainID,
			&domain.ParentHostname,
			&isPrimary,
			&domain.CreatedAt,
			&domain.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan domain: %w", err)
		}
		domain.IsPrimary = isPrimary == 1
		out = append(out, domain)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate domains: %w", err)
	}
	return out, nil
}

func (s *DomainStore) getByIDTx(ctx context.Context, tx *sql.Tx, id string) (*StoredDomain, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return nil, err
	}
	var rows *sql.Rows
	if tx != nil {
		rows, err = tx.QueryContext(ctx, domainSelectQuery+` WHERE d.id = ?`, publicID)
	} else {
		rows, err = s.db.QueryContext(ctx, domainSelectQuery+` WHERE d.id = ?`, publicID)
	}
	if err != nil {
		return nil, fmt.Errorf("get domain: %w", err)
	}
	defer rows.Close()
	domains, err := scanDomains(rows)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("domain %s not found", publicID)
	}
	return &domains[0], nil
}

func (s *DomainStore) getByHostnameTx(ctx context.Context, tx *sql.Tx, hostname string) (*StoredDomain, error) {
	var (
		rows *sql.Rows
		err  error
	)
	if tx != nil {
		rows, err = tx.QueryContext(ctx, domainSelectQuery+` WHERE d.hostname = ? LIMIT 1`, hostname)
	} else {
		rows, err = s.db.QueryContext(ctx, domainSelectQuery+` WHERE d.hostname = ? LIMIT 1`, hostname)
	}
	if err != nil {
		return nil, fmt.Errorf("get domain by hostname: %w", err)
	}
	defer rows.Close()
	domains, err := scanDomains(rows)
	if err != nil {
		return nil, err
	}
	if len(domains) == 0 {
		return nil, sql.ErrNoRows
	}
	return &domains[0], nil
}

func (s *DomainStore) ensureRelations(ctx context.Context, hostname, kind, source, siteID, parentDomainID string) error {
	return s.ensureRelationsTx(ctx, nil, hostname, kind, source, siteID, parentDomainID)
}

func (s *DomainStore) ensureRelationsTx(ctx context.Context, tx *sql.Tx, hostname, kind, source, siteID, parentDomainID string) error {
	if siteID != "" {
		if err := ensureSiteExists(ctx, s.db, tx, siteID); err != nil {
			return err
		}
	}
	if parentDomainID != "" {
		parent, err := s.getByIDTx(ctx, tx, parentDomainID)
		if err != nil {
			return fmt.Errorf("parent_domain_id: %w", err)
		}
		if parent.Kind != DomainKindBaseDomain {
			return fmt.Errorf("parent_domain_id must reference a base domain")
		}
		if parent.Source != DomainSourceUser {
			return fmt.Errorf("parent_domain_id must reference a user-managed base domain")
		}
		if kind != DomainKindHostname {
			return fmt.Errorf("only hostnames can reference a parent_domain_id")
		}
		if source != DomainSourceUser {
			return fmt.Errorf("child hostnames under a base domain must use source %q", DomainSourceUser)
		}
		if !strings.HasSuffix(hostname, "."+parent.Hostname) {
			return fmt.Errorf("hostname must be within the selected base domain")
		}
	}
	if kind == DomainKindBaseDomain && parentDomainID != "" {
		return fmt.Errorf("base domains cannot have a parent_domain_id")
	}
	return nil
}
