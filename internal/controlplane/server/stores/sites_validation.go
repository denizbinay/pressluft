package stores

import (
	"context"
	"database/sql"
	"fmt"
	"net/mail"
	"regexp"
	"strings"

	"pressluft/internal/shared/idutil"
)

func validateCreateSiteInput(in CreateSiteInput) error {
	if _, err := idutil.Normalize(in.ServerID); err != nil {
		return fmt.Errorf("server_id: %w", err)
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(in.WordPressAdminEmail) == "" {
		return fmt.Errorf("wordpress_admin_email is required")
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(in.WordPressAdminEmail)); err != nil {
		return fmt.Errorf("wordpress_admin_email must be a valid email address")
	}
	if strings.TrimSpace(in.PrimaryDomain) != "" && in.PrimaryHostnameConfig != nil {
		return fmt.Errorf("use either primary_domain or primary_hostname_config, not both")
	}
	if strings.TrimSpace(in.PrimaryDomain) != "" {
		if _, err := normalizeHostname(in.PrimaryDomain); err != nil {
			return err
		}
	}
	if in.PrimaryHostnameConfig != nil {
		if err := validateCreateSitePrimaryHostnameInput(*in.PrimaryHostnameConfig); err != nil {
			return err
		}
	}
	if _, err := NormalizeSiteStatus(in.Status); err != nil {
		return err
	}
	return nil
}

func validateCreateSitePrimaryHostnameInput(in CreateSitePrimaryHostnameInput) error {
	source := strings.TrimSpace(in.Source)
	switch source {
	case DomainSourceFallbackResolver:
		if strings.TrimSpace(in.Label) == "" {
			return fmt.Errorf("primary_hostname_config.label is required for fallback resolver hostnames")
		}
		if _, err := normalizeDomainLabel(strings.TrimSpace(in.Label)); err != nil {
			return err
		}
		if strings.TrimSpace(in.Hostname) != "" || strings.TrimSpace(in.DomainID) != "" {
			return fmt.Errorf("primary_hostname_config for fallback resolver hostnames only accepts label")
		}
		return nil
	case DomainSourceUser:
		hasHostname := strings.TrimSpace(in.Hostname) != ""
		hasBaseDomain := strings.TrimSpace(in.DomainID) != ""
		hasLabel := strings.TrimSpace(in.Label) != ""
		switch {
		case hasHostname && (hasBaseDomain || hasLabel):
			return fmt.Errorf("primary_hostname_config.user_domain accepts either hostname or domain_id plus label")
		case hasHostname:
			if _, err := normalizeHostname(strings.TrimSpace(in.Hostname)); err != nil {
				return err
			}
			return nil
		case hasBaseDomain:
			if !hasLabel {
				return fmt.Errorf("primary_hostname_config.label is required when using a reusable base domain")
			}
			if _, err := normalizeDomainLabel(strings.TrimSpace(in.Label)); err != nil {
				return err
			}
			return nil
		default:
			return fmt.Errorf("primary_hostname_config.user_domain requires hostname or domain_id")
		}
	default:
		return fmt.Errorf("primary_hostname_config.source must be fallback_resolver or user")
	}
}

func resolveCreateSitePrimaryHostnameInput(in CreateSiteInput) (CreateSitePrimaryHostnameInput, bool) {
	if in.PrimaryHostnameConfig != nil {
		return *in.PrimaryHostnameConfig, true
	}
	if strings.TrimSpace(in.PrimaryDomain) == "" {
		return CreateSitePrimaryHostnameInput{}, false
	}
	return CreateSitePrimaryHostnameInput{
		Source:   DomainSourceUser,
		Hostname: strings.TrimSpace(in.PrimaryDomain),
	}, true
}

func (s *DomainStore) createWithTx(ctx context.Context, tx *sql.Tx, siteID, serverID string, input CreateSitePrimaryHostnameInput) (string, error) {
	source := strings.TrimSpace(input.Source)
	switch source {
	case DomainSourceFallbackResolver:
		serverIPv4, err := lookupServerIPv4Tx(ctx, tx, s.db, serverID)
		if err != nil {
			return "", err
		}
		hostname, err := buildFallbackResolverHostname(strings.TrimSpace(input.Label), serverIPv4)
		if err != nil {
			return "", err
		}
		return s.createTx(ctx, tx, CreateDomainInput{
			Hostname:     hostname,
			Kind:         DomainKindHostname,
			Source:       DomainSourceFallbackResolver,
			DNSState:     DomainDNSStateReady,
			RoutingState: DomainRoutingStatePending,
			SiteID:       siteID,
			IsPrimary:    true,
		})
	case DomainSourceUser:
		if strings.TrimSpace(input.DomainID) != "" {
			parent, err := s.getByIDTx(ctx, tx, strings.TrimSpace(input.DomainID))
			if err != nil {
				return "", fmt.Errorf("primary_hostname_config.domain_id: %w", err)
			}
			hostname, err := buildChildHostname(strings.TrimSpace(input.Label), *parent)
			if err != nil {
				return "", err
			}
			dnsState := DomainDNSStatePending
			if parent.DNSState == DomainDNSStateReady {
				dnsState = DomainDNSStateReady
			}
			return s.createTx(ctx, tx, CreateDomainInput{
				Hostname:       hostname,
				Kind:           DomainKindHostname,
				Source:         DomainSourceUser,
				DNSState:       dnsState,
				RoutingState:   DomainRoutingStatePending,
				SiteID:         siteID,
				ParentDomainID: strings.TrimSpace(input.DomainID),
				IsPrimary:      true,
			})
		}
		return s.createTx(ctx, tx, CreateDomainInput{
			Hostname:     strings.TrimSpace(input.Hostname),
			Kind:         DomainKindHostname,
			Source:       DomainSourceUser,
			DNSState:     DomainDNSStatePending,
			RoutingState: DomainRoutingStatePending,
			SiteID:       siteID,
			IsPrimary:    true,
		})
	default:
		return "", fmt.Errorf("primary_hostname_config.source must be fallback_resolver or user")
	}
}

func buildChildHostname(label string, parent StoredDomain) (string, error) {
	normalizedLabel, err := normalizeDomainLabel(label)
	if err != nil {
		return "", err
	}
	if parent.Kind != DomainKindBaseDomain {
		return "", fmt.Errorf("primary_hostname_config.domain_id must reference a base domain")
	}
	if parent.Source != DomainSourceUser {
		return "", fmt.Errorf("primary_hostname_config.domain_id must reference a user-managed base domain")
	}
	return normalizeHostname(normalizedLabel + "." + parent.Hostname)
}

func buildFallbackResolverHostname(label, ipv4 string) (string, error) {
	normalizedLabel, err := normalizeDomainLabel(label)
	if err != nil {
		return "", err
	}
	ipv4 = strings.TrimSpace(ipv4)
	if ipv4 == "" {
		return "", fmt.Errorf("selected server does not have an IPv4 address for fallback resolver hostnames")
	}
	resolverSuffix := strings.ReplaceAll(ipv4, ".", "-") + ".sslip.io"
	return normalizeHostname(normalizedLabel + "." + resolverSuffix)
}

func normalizeDomainLabel(label string) (string, error) {
	label = strings.ToLower(strings.TrimSpace(label))
	label = strings.ReplaceAll(label, "_", "-")
	label = regexp.MustCompile(`[^a-z0-9-]+`).ReplaceAllString(label, "-")
	label = strings.Trim(label, "-")
	if label == "" {
		return "", fmt.Errorf("primary_hostname_config.label is required")
	}
	if strings.Contains(label, ".") {
		return "", fmt.Errorf("primary_hostname_config.label must be a single subdomain label")
	}
	if len(label) > 63 {
		return "", fmt.Errorf("primary_hostname_config.label must be 63 characters or fewer")
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
	if in.WordPressAdminEmail != nil {
		if strings.TrimSpace(*in.WordPressAdminEmail) == "" {
			return fmt.Errorf("wordpress_admin_email is required")
		}
		if _, err := mail.ParseAddress(strings.TrimSpace(*in.WordPressAdminEmail)); err != nil {
			return fmt.Errorf("wordpress_admin_email must be a valid email address")
		}
	}
	return nil
}

func lookupServerIPv4Tx(ctx context.Context, tx *sql.Tx, db *sql.DB, serverID string) (string, error) {
	var ipv4 sql.NullString
	var err error
	if tx != nil {
		err = tx.QueryRowContext(ctx, `SELECT ipv4 FROM servers WHERE id = ?`, serverID).Scan(&ipv4)
	} else {
		err = db.QueryRowContext(ctx, `SELECT ipv4 FROM servers WHERE id = ?`, serverID).Scan(&ipv4)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("server %s not found", serverID)
		}
		return "", fmt.Errorf("lookup server ipv4: %w", err)
	}
	value := strings.TrimSpace(nullStringValue(ipv4))
	if value == "" {
		return "", fmt.Errorf("selected server does not have an IPv4 address for fallback resolver hostnames")
	}
	return value, nil
}
