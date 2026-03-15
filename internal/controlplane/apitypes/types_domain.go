package apitypes

import (
	"fmt"
	"strings"
)

type CreateDomainRequest struct {
	Hostname             string `json:"hostname"`
	Kind                 string `json:"kind,omitempty"`
	Source               string `json:"source,omitempty"`
	DNSState             string `json:"dns_state,omitempty"`
	RoutingState         string `json:"routing_state,omitempty"`
	DNSStatusMessage     string `json:"dns_status_message,omitempty"`
	RoutingStatusMessage string `json:"routing_status_message,omitempty"`
	LastCheckedAt        string `json:"last_checked_at,omitempty"`
	SiteID               string `json:"site_id,omitempty"`
	ParentDomainID       string `json:"parent_domain_id,omitempty"`
	IsPrimary            bool   `json:"is_primary,omitempty"`
}

func (r *CreateDomainRequest) Validate() error {
	r.Hostname = strings.TrimSpace(r.Hostname)
	r.Kind = strings.TrimSpace(r.Kind)
	r.Source = strings.TrimSpace(r.Source)
	r.DNSState = strings.TrimSpace(r.DNSState)
	r.RoutingState = strings.TrimSpace(r.RoutingState)
	r.DNSStatusMessage = strings.TrimSpace(r.DNSStatusMessage)
	r.RoutingStatusMessage = strings.TrimSpace(r.RoutingStatusMessage)
	r.LastCheckedAt = strings.TrimSpace(r.LastCheckedAt)
	r.SiteID = strings.TrimSpace(r.SiteID)
	r.ParentDomainID = strings.TrimSpace(r.ParentDomainID)
	if r.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	return nil
}

type UpdateDomainRequest struct {
	Hostname             *string `json:"hostname,omitempty"`
	Kind                 *string `json:"kind,omitempty"`
	Source               *string `json:"source,omitempty"`
	DNSState             *string `json:"dns_state,omitempty"`
	RoutingState         *string `json:"routing_state,omitempty"`
	DNSStatusMessage     *string `json:"dns_status_message,omitempty"`
	RoutingStatusMessage *string `json:"routing_status_message,omitempty"`
	LastCheckedAt        *string `json:"last_checked_at,omitempty"`
	SiteID               *string `json:"site_id,omitempty"`
	ParentDomainID       *string `json:"parent_domain_id,omitempty"`
	IsPrimary            *bool   `json:"is_primary,omitempty"`
}

func (r *UpdateDomainRequest) Validate() error {
	trim := func(value **string) {
		if *value == nil {
			return
		}
		trimmed := strings.TrimSpace(**value)
		*value = &trimmed
	}
	trim(&r.Hostname)
	trim(&r.Kind)
	trim(&r.Source)
	trim(&r.DNSState)
	trim(&r.RoutingState)
	trim(&r.DNSStatusMessage)
	trim(&r.RoutingStatusMessage)
	trim(&r.LastCheckedAt)
	trim(&r.SiteID)
	trim(&r.ParentDomainID)
	if r.Hostname != nil && *r.Hostname == "" {
		return fmt.Errorf("hostname is required")
	}
	return nil
}

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

type DeleteDomainResponse struct {
	DomainID    string `json:"domain_id"`
	Deleted     bool   `json:"deleted"`
	Description string `json:"description"`
}
