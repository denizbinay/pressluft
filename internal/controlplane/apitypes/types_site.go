package apitypes

import (
	"fmt"
	"net/mail"
	"strings"

	"pressluft/internal/agent/agentcommand"
)

type CreateSiteRequest struct {
	ServerID              string                     `json:"server_id"`
	Name                  string                     `json:"name"`
	WordPressAdminEmail   string                     `json:"wordpress_admin_email"`
	PrimaryDomain         string                     `json:"primary_domain,omitempty"`
	PrimaryHostnameConfig *SitePrimaryHostnameConfig `json:"primary_hostname_config,omitempty"`
	Status                string                     `json:"status,omitempty"`
	WordPressPath         string                     `json:"wordpress_path,omitempty"`
	PHPVersion            string                     `json:"php_version,omitempty"`
	WordPressVersion      string                     `json:"wordpress_version,omitempty"`
}

type SitePrimaryHostnameConfig struct {
	Source   string `json:"source"`
	Hostname string `json:"hostname,omitempty"`
	Label    string `json:"label,omitempty"`
	DomainID string `json:"domain_id,omitempty"`
}

func (c *SitePrimaryHostnameConfig) Validate() error {
	if c == nil {
		return nil
	}
	c.Source = strings.TrimSpace(c.Source)
	c.Hostname = strings.TrimSpace(c.Hostname)
	c.Label = strings.TrimSpace(c.Label)
	c.DomainID = strings.TrimSpace(c.DomainID)
	switch c.Source {
	case "fallback_resolver":
		if c.Label == "" {
			return fmt.Errorf("primary_hostname_config.label is required for fallback resolver hostnames")
		}
	case "user":
		hasHostname := c.Hostname != ""
		hasDomain := c.DomainID != ""
		hasLabel := c.Label != ""
		switch {
		case hasHostname && (hasDomain || hasLabel):
			return fmt.Errorf("primary_hostname_config.user requires either hostname or domain_id plus label")
		case hasHostname:
			return nil
		case hasDomain && hasLabel:
			return nil
		case hasDomain && !hasLabel:
			return fmt.Errorf("primary_hostname_config.label is required when domain_id is set")
		default:
			return fmt.Errorf("primary_hostname_config.user requires hostname or domain_id")
		}
	default:
		return fmt.Errorf("primary_hostname_config.source must be fallback_resolver or user")
	}
	return nil
}

func (r *CreateSiteRequest) Validate() error {
	r.ServerID = strings.TrimSpace(r.ServerID)
	r.Name = strings.TrimSpace(r.Name)
	r.WordPressAdminEmail = strings.TrimSpace(r.WordPressAdminEmail)
	r.PrimaryDomain = strings.TrimSpace(r.PrimaryDomain)
	r.Status = strings.TrimSpace(r.Status)
	r.WordPressPath = strings.TrimSpace(r.WordPressPath)
	r.PHPVersion = strings.TrimSpace(r.PHPVersion)
	r.WordPressVersion = strings.TrimSpace(r.WordPressVersion)
	if r.ServerID == "" {
		return fmt.Errorf("server_id is required")
	}
	if r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.WordPressAdminEmail == "" {
		return fmt.Errorf("wordpress_admin_email is required")
	}
	if _, err := mail.ParseAddress(r.WordPressAdminEmail); err != nil {
		return fmt.Errorf("wordpress_admin_email must be a valid email address")
	}
	if r.PrimaryDomain != "" && r.PrimaryHostnameConfig != nil {
		return fmt.Errorf("use either primary_domain or primary_hostname_config, not both")
	}
	if err := r.PrimaryHostnameConfig.Validate(); err != nil {
		return err
	}
	return nil
}

type UpdateSiteRequest struct {
	ServerID            *string `json:"server_id,omitempty"`
	Name                *string `json:"name,omitempty"`
	WordPressAdminEmail *string `json:"wordpress_admin_email,omitempty"`
	PrimaryDomain       *string `json:"primary_domain,omitempty"`
	Status              *string `json:"status,omitempty"`
	WordPressPath       *string `json:"wordpress_path,omitempty"`
	PHPVersion          *string `json:"php_version,omitempty"`
	WordPressVersion    *string `json:"wordpress_version,omitempty"`
}

func (r *UpdateSiteRequest) Validate() error {
	trim := func(value **string) {
		if *value == nil {
			return
		}
		trimmed := strings.TrimSpace(**value)
		*value = &trimmed
	}
	trim(&r.ServerID)
	trim(&r.Name)
	trim(&r.WordPressAdminEmail)
	trim(&r.PrimaryDomain)
	trim(&r.Status)
	trim(&r.WordPressPath)
	trim(&r.PHPVersion)
	trim(&r.WordPressVersion)
	if r.Name != nil && *r.Name == "" {
		return fmt.Errorf("name is required")
	}
	if r.ServerID != nil && *r.ServerID == "" {
		return fmt.Errorf("server_id is required")
	}
	if r.WordPressAdminEmail != nil {
		if *r.WordPressAdminEmail == "" {
			return fmt.Errorf("wordpress_admin_email is required")
		}
		if _, err := mail.ParseAddress(*r.WordPressAdminEmail); err != nil {
			return fmt.Errorf("wordpress_admin_email must be a valid email address")
		}
	}
	return nil
}

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

type SiteHealthResponse struct {
	SiteID         string                           `json:"site_id"`
	AgentConnected bool                             `json:"agent_connected"`
	Snapshot       *agentcommand.SiteHealthSnapshot `json:"snapshot,omitempty"`
	RuntimeState   string                           `json:"runtime_health_state"`
	RuntimeMessage string                           `json:"runtime_health_status_message,omitempty"`
	LastCheckedAt  string                           `json:"last_health_check_at,omitempty"`
}

type DeleteSiteResponse struct {
	SiteID      string `json:"site_id"`
	Deleted     bool   `json:"deleted"`
	Description string `json:"description"`
}
