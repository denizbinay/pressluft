package stores

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/shared/idutil"
)

func shouldPromotePrimaryTx(ctx context.Context, tx *sql.Tx, siteID string) (bool, error) {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM domains WHERE site_id = ? AND is_primary = 1`, siteID).Scan(&count); err != nil {
		return false, fmt.Errorf("count primary domains: %w", err)
	}
	return count == 0, nil
}

func clearPrimaryForSiteTx(ctx context.Context, tx *sql.Tx, siteID, exceptID string) error {
	query := `UPDATE domains SET is_primary = 0, updated_at = ? WHERE site_id = ?`
	args := []any{time.Now().UTC().Format(time.RFC3339), siteID}
	if strings.TrimSpace(exceptID) != "" {
		query += ` AND id != ?`
		args = append(args, exceptID)
	}
	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		return fmt.Errorf("clear site primary domains: %w", err)
	}
	return nil
}

func ensurePrimaryDomainTx(ctx context.Context, tx *sql.Tx, siteID string) error {
	if strings.TrimSpace(siteID) == "" {
		return nil
	}
	promote, err := shouldPromotePrimaryTx(ctx, tx, siteID)
	if err != nil {
		return err
	}
	if !promote {
		return nil
	}
	var domainID string
	if err := tx.QueryRowContext(ctx, `SELECT id FROM domains WHERE site_id = ? ORDER BY created_at ASC, id ASC LIMIT 1`, siteID).Scan(&domainID); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("select replacement primary domain: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE domains SET is_primary = 1, updated_at = ? WHERE id = ?`, time.Now().UTC().Format(time.RFC3339), domainID); err != nil {
		return fmt.Errorf("promote replacement primary domain: %w", err)
	}
	return nil
}

func syncLegacySitePrimaryDomainTx(ctx context.Context, tx *sql.Tx, siteID string) error {
	if strings.TrimSpace(siteID) == "" {
		return nil
	}
	var hostname sql.NullString
	if err := tx.QueryRowContext(ctx, `SELECT hostname FROM domains WHERE site_id = ? AND is_primary = 1 LIMIT 1`, siteID).Scan(&hostname); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("lookup primary domain for site sync: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE sites SET primary_domain = ?, updated_at = updated_at WHERE id = ?`, nullableString(nullStringValue(hostname)), siteID); err != nil {
		return fmt.Errorf("sync site primary_domain: %w", err)
	}
	return nil
}

func ensureSiteExists(ctx context.Context, db *sql.DB, tx *sql.Tx, siteID string) error {
	var exists string
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, `SELECT id FROM sites WHERE id = ?`, siteID).Scan(&exists)
	} else {
		err = db.QueryRowContext(ctx, `SELECT id FROM sites WHERE id = ?`, siteID).Scan(&exists)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("site %s not found", siteID)
		}
		return fmt.Errorf("lookup site id: %w", err)
	}
	return nil
}

func (s *DomainStore) SetPrimaryHostnameForSite(ctx context.Context, siteID, hostname, source string) error {
	return s.setPrimaryHostnameForSiteTx(ctx, nil, siteID, hostname, source)
}

func (s *DomainStore) setPrimaryHostnameForSiteTx(ctx context.Context, tx *sql.Tx, siteID, hostname, source string) error {
	normalizedSiteID, err := idutil.Normalize(siteID)
	if err != nil {
		return fmt.Errorf("site_id: %w", err)
	}
	normalizedHostname, err := normalizeHostname(hostname)
	if err != nil {
		return err
	}
	normalizedSource, err := normalizeDomainSource(source)
	if err != nil {
		return err
	}
	existing, err := s.getByHostnameTx(ctx, tx, normalizedHostname)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("lookup site hostname: %w", err)
	}
	if err == nil {
		if existing.Kind != DomainKindHostname {
			return fmt.Errorf("hostname %q cannot be used as a site primary hostname", normalizedHostname)
		}
		if existing.SiteID != "" && existing.SiteID != normalizedSiteID {
			return fmt.Errorf("hostname %q already exists", normalizedHostname)
		}
		isPrimary := true
		routingState := DomainRoutingStatePending
		_, err := s.updateTx(ctx, tx, existing.ID, UpdateDomainInput{IsPrimary: &isPrimary, SiteID: &normalizedSiteID, RoutingState: &routingState})
		return err
	}
	_, err = s.createTx(ctx, tx, CreateDomainInput{
		Hostname:     normalizedHostname,
		Kind:         DomainKindHostname,
		Source:       normalizedSource,
		SiteID:       normalizedSiteID,
		IsPrimary:    true,
		RoutingState: DomainRoutingStatePending,
	})
	return err
}

func (s *DomainStore) ClearPrimaryHostnameForSite(ctx context.Context, siteID string) error {
	return s.clearPrimaryHostnameForSiteTx(ctx, nil, siteID)
}

func (s *DomainStore) clearPrimaryHostnameForSiteTx(ctx context.Context, tx *sql.Tx, siteID string) error {
	normalizedSiteID, err := idutil.Normalize(siteID)
	if err != nil {
		return fmt.Errorf("site_id: %w", err)
	}
	ownedTx := false
	if tx == nil {
		tx, err = s.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin clear primary tx: %w", err)
		}
		ownedTx = true
	}
	defer func() {
		if ownedTx {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `UPDATE domains SET is_primary = 0, updated_at = ? WHERE site_id = ?`, time.Now().UTC().Format(time.RFC3339), normalizedSiteID); err != nil {
		return fmt.Errorf("clear primary hostname: %w", err)
	}
	if err := syncLegacySitePrimaryDomainTx(ctx, tx, normalizedSiteID); err != nil {
		return err
	}
	if ownedTx {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit clear primary tx: %w", err)
		}
	}
	return nil
}
