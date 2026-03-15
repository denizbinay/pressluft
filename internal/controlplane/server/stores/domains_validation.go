package stores

import (
	"fmt"
	"regexp"
	"strings"

	"pressluft/internal/shared/idutil"
)

var hostnamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)

func prepareCreateDomainInput(in CreateDomainInput) (CreateDomainInput, error) {
	hostname, err := normalizeHostname(in.Hostname)
	if err != nil {
		return CreateDomainInput{}, err
	}
	in.Hostname = hostname
	in.Kind = strings.TrimSpace(in.Kind)
	in.Source = strings.TrimSpace(in.Source)
	in.DNSState = strings.TrimSpace(in.DNSState)
	in.RoutingState = strings.TrimSpace(in.RoutingState)
	in.DNSStatusMessage = strings.TrimSpace(in.DNSStatusMessage)
	in.RoutingStatusMessage = strings.TrimSpace(in.RoutingStatusMessage)
	in.LastCheckedAt = strings.TrimSpace(in.LastCheckedAt)
	in.SiteID = strings.TrimSpace(in.SiteID)
	in.ParentDomainID = strings.TrimSpace(in.ParentDomainID)
	if in.Kind == "" {
		return CreateDomainInput{}, fmt.Errorf("kind is required")
	}
	if in.Source == "" {
		return CreateDomainInput{}, fmt.Errorf("source is required")
	}
	if _, err := normalizeDomainKind(in.Kind); err != nil {
		return CreateDomainInput{}, err
	}
	if _, err := normalizeDomainSource(in.Source); err != nil {
		return CreateDomainInput{}, err
	}
	if in.DNSState == "" {
		if in.Source == DomainSourceFallbackResolver {
			in.DNSState = DomainDNSStateReady
		} else {
			in.DNSState = DomainDNSStatePending
		}
	}
	if _, err := normalizeDomainDNSState(in.DNSState); err != nil {
		return CreateDomainInput{}, err
	}
	if in.RoutingState == "" {
		if in.SiteID != "" {
			in.RoutingState = DomainRoutingStatePending
		} else {
			in.RoutingState = DomainRoutingStateNotConfigured
		}
	}
	if _, err := normalizeDomainRoutingState(in.RoutingState); err != nil {
		return CreateDomainInput{}, err
	}
	if in.SiteID != "" {
		normalized, err := idutil.Normalize(in.SiteID)
		if err != nil {
			return CreateDomainInput{}, fmt.Errorf("site_id: %w", err)
		}
		in.SiteID = normalized
	}
	if in.ParentDomainID != "" {
		normalized, err := idutil.Normalize(in.ParentDomainID)
		if err != nil {
			return CreateDomainInput{}, fmt.Errorf("parent_domain_id: %w", err)
		}
		in.ParentDomainID = normalized
	}
	if in.Kind == DomainKindBaseDomain && (in.SiteID != "" || in.IsPrimary) {
		return CreateDomainInput{}, fmt.Errorf("base domains cannot be assigned to a site or marked primary")
	}
	if in.Source == DomainSourceFallbackResolver {
		if in.Kind != DomainKindHostname {
			return CreateDomainInput{}, fmt.Errorf("fallback resolver entries must be hostnames")
		}
		if in.ParentDomainID != "" {
			return CreateDomainInput{}, fmt.Errorf("fallback resolver hostnames cannot reference a parent domain")
		}
		if in.SiteID == "" {
			return CreateDomainInput{}, fmt.Errorf("fallback resolver hostnames must be attached to a site")
		}
	}
	return in, nil
}

func prepareUpdateDomainInput(current StoredDomain, in UpdateDomainInput) (CreateDomainInput, error) {
	prepared := CreateDomainInput{
		Hostname:             current.Hostname,
		Kind:                 current.Kind,
		Source:               current.Source,
		DNSState:             current.DNSState,
		RoutingState:         current.RoutingState,
		DNSStatusMessage:     current.DNSStatusMessage,
		RoutingStatusMessage: current.RoutingStatusMessage,
		LastCheckedAt:        current.LastCheckedAt,
		SiteID:               current.SiteID,
		ParentDomainID:       current.ParentDomainID,
		IsPrimary:            current.IsPrimary,
	}
	if in.Hostname != nil {
		prepared.Hostname = *in.Hostname
	}
	if in.Kind != nil {
		prepared.Kind = *in.Kind
	}
	if in.Source != nil {
		prepared.Source = *in.Source
	}
	if in.DNSState != nil {
		prepared.DNSState = *in.DNSState
	}
	if in.RoutingState != nil {
		prepared.RoutingState = *in.RoutingState
	}
	if in.DNSStatusMessage != nil {
		prepared.DNSStatusMessage = strings.TrimSpace(*in.DNSStatusMessage)
	}
	if in.RoutingStatusMessage != nil {
		prepared.RoutingStatusMessage = strings.TrimSpace(*in.RoutingStatusMessage)
	}
	if in.LastCheckedAt != nil {
		prepared.LastCheckedAt = strings.TrimSpace(*in.LastCheckedAt)
	}
	if in.SiteID != nil {
		prepared.SiteID = strings.TrimSpace(*in.SiteID)
	}
	if in.ParentDomainID != nil {
		prepared.ParentDomainID = strings.TrimSpace(*in.ParentDomainID)
	}
	if in.IsPrimary != nil {
		prepared.IsPrimary = *in.IsPrimary
	}
	return prepareCreateDomainInput(prepared)
}

func normalizeHostname(raw string) (string, error) {
	hostname := strings.ToLower(strings.TrimSpace(raw))
	hostname = strings.TrimSuffix(hostname, ".")
	if hostname == "" {
		return "", fmt.Errorf("hostname is required")
	}
	if !hostnamePattern.MatchString(hostname) {
		return "", fmt.Errorf("hostname must be a valid domain name")
	}
	return hostname, nil
}

func normalizeDomainKind(raw string) (string, error) {
	kind := strings.TrimSpace(raw)
	switch kind {
	case DomainKindHostname, DomainKindBaseDomain:
		return kind, nil
	default:
		return "", fmt.Errorf("unsupported domain kind %q", raw)
	}
}

func normalizeDomainSource(raw string) (string, error) {
	source := strings.TrimSpace(raw)
	switch source {
	case DomainSourceUser, DomainSourceFallbackResolver:
		return source, nil
	default:
		return "", fmt.Errorf("unsupported domain source %q", raw)
	}
}

func normalizeDomainDNSState(raw string) (string, error) {
	state := strings.TrimSpace(raw)
	switch state {
	case DomainDNSStatePending, DomainDNSStateReady, DomainDNSStateIssue, DomainDNSStateDisabled:
		return state, nil
	default:
		return "", fmt.Errorf("unsupported dns_state %q", raw)
	}
}

func normalizeDomainRoutingState(raw string) (string, error) {
	state := strings.TrimSpace(raw)
	switch state {
	case DomainRoutingStateNotConfigured, DomainRoutingStatePending, DomainRoutingStateReady, DomainRoutingStateIssue:
		return state, nil
	default:
		return "", fmt.Errorf("unsupported routing_state %q", raw)
	}
}
