package apitypes

import (
	"encoding/json"
	"fmt"
	"net/mail"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/agentcommand"
	"pressluft/internal/auth"
	"pressluft/internal/idutil"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/server/profiles"
	"pressluft/internal/ws"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" || strings.TrimSpace(r.Password) == "" {
		return fmt.Errorf("email and password are required")
	}
	return nil
}

type StatusResponse struct {
	Status string `json:"status"`
}

type HealthResponse struct {
	Status             string                   `json:"status"`
	CallbackURLMode    platform.CallbackURLMode `json:"callback_url_mode,omitempty"`
	CallbackURLWarning string                   `json:"callback_url_warning,omitempty"`
}

type CreateProviderRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	APIToken string `json:"api_token"`
}

func (r *CreateProviderRequest) Validate() error {
	r.Type = strings.TrimSpace(r.Type)
	r.Name = strings.TrimSpace(r.Name)
	if r.Type == "" || r.Name == "" || strings.TrimSpace(r.APIToken) == "" {
		return fmt.Errorf("type, name, and api_token are required")
	}
	return nil
}

type ValidateProviderRequest struct {
	Type     string `json:"type"`
	APIToken string `json:"api_token"`
}

func (r *ValidateProviderRequest) Validate() error {
	r.Type = strings.TrimSpace(r.Type)
	if r.Type == "" || strings.TrimSpace(r.APIToken) == "" {
		return fmt.Errorf("type and api_token are required")
	}
	return nil
}

type CreateProviderResponse struct {
	ID         string                    `json:"id"`
	Validation provider.ValidationResult `json:"validation"`
}

type CreateServerRequest struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Location   string `json:"location"`
	ServerType string `json:"server_type"`
	ProfileKey string `json:"profile_key"`
}

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

type ServerCatalogResponse struct {
	Catalog  provider.ServerCatalog `json:"catalog"`
	Profiles []profiles.Profile     `json:"profiles"`
}

type CreateServerResponse struct {
	ServerID string                `json:"server_id"`
	JobID    string                `json:"job_id"`
	Status   platform.ServerStatus `json:"status"`
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

type DeleteSiteResponse struct {
	SiteID      string `json:"site_id"`
	Deleted     bool   `json:"deleted"`
	Description string `json:"description"`
}

type StoredServer struct {
	ID               string                `json:"id"`
	ProviderID       string                `json:"provider_id"`
	ProviderType     string                `json:"provider_type"`
	ProviderServerID string                `json:"provider_server_id,omitempty"`
	IPv4             string                `json:"ipv4,omitempty"`
	IPv6             string                `json:"ipv6,omitempty"`
	Name             string                `json:"name"`
	Location         string                `json:"location"`
	ServerType       string                `json:"server_type"`
	Image            string                `json:"image"`
	ProfileKey       string                `json:"profile_key"`
	Status           platform.ServerStatus `json:"status"`
	SetupState       platform.SetupState   `json:"setup_state"`
	SetupLastError   string                `json:"setup_last_error,omitempty"`
	ActionID         string                `json:"action_id,omitempty"`
	ActionStatus     string                `json:"action_status,omitempty"`
	HasKey           bool                  `json:"has_key"`
	NodeStatus       platform.NodeStatus   `json:"node_status,omitempty"`
	NodeLastSeen     string                `json:"node_last_seen,omitempty"`
	NodeVersion      string                `json:"node_version,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

type DeleteServerResponse struct {
	ServerID    string                 `json:"server_id"`
	JobID       string                 `json:"job_id"`
	Status      platform.ServerStatus  `json:"status"`
	JobStatus   orchestrator.JobStatus `json:"job_status"`
	Async       bool                   `json:"async"`
	Description string                 `json:"description"`
}

type RebuildOptionsResponse struct {
	ServerID     string                       `json:"server_id"`
	ServerType   string                       `json:"server_type"`
	Architecture string                       `json:"architecture"`
	Images       []provider.ServerImageOption `json:"images"`
}

type ResizeOptionsResponse struct {
	ServerID     string                      `json:"server_id"`
	Location     string                      `json:"location"`
	ServerType   string                      `json:"server_type"`
	Architecture string                      `json:"architecture"`
	ServerTypes  []provider.ServerTypeOption `json:"server_types"`
}

type FirewallsResponse struct {
	ServerID  string                    `json:"server_id"`
	Firewalls []provider.FirewallOption `json:"firewalls"`
}

type VolumesResponse struct {
	ServerID string                  `json:"server_id"`
	Volumes  []provider.VolumeOption `json:"volumes"`
}

type AgentStatusMapResponse map[string]ws.AgentInfo

type ServicesResponse struct {
	ServerID       string                 `json:"server_id"`
	AgentConnected bool                   `json:"agent_connected"`
	Services       []agentcommand.Service `json:"services"`
}

type CreateJobRequest struct {
	Kind     string          `json:"kind"`
	ServerID string          `json:"server_id,omitempty"`
	Payload  json.RawMessage `json:"payload"`
}

func (r *CreateJobRequest) Validate() error {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	return nil
}

type ActivityListResponse struct {
	Data       []Activity `json:"data"`
	NextCursor string     `json:"next_cursor,omitempty"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

type Job struct {
	ID          string                 `json:"id"`
	ServerID    string                 `json:"server_id,omitempty"`
	Kind        string                 `json:"kind"`
	Status      orchestrator.JobStatus `json:"status"`
	CurrentStep string                 `json:"current_step"`
	RetryCount  int                    `json:"retry_count"`
	LastError   string                 `json:"last_error,omitempty"`
	Payload     string                 `json:"payload,omitempty"`
	StartedAt   string                 `json:"started_at,omitempty"`
	FinishedAt  string                 `json:"finished_at,omitempty"`
	TimeoutAt   string                 `json:"timeout_at,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	CommandID   *string                `json:"command_id,omitempty"`
}

type Activity struct {
	ID                 string                `json:"id"`
	EventType          activity.EventType    `json:"event_type"`
	Category           activity.Category     `json:"category"`
	Level              activity.Level        `json:"level"`
	ResourceType       activity.ResourceType `json:"resource_type,omitempty"`
	ResourceID         string                `json:"resource_id,omitempty"`
	ParentResourceType activity.ResourceType `json:"parent_resource_type,omitempty"`
	ParentResourceID   string                `json:"parent_resource_id,omitempty"`
	ActorType          activity.ActorType    `json:"actor_type"`
	ActorID            string                `json:"actor_id,omitempty"`
	Title              string                `json:"title"`
	Message            string                `json:"message,omitempty"`
	Payload            string                `json:"payload,omitempty"`
	RequiresAttention  bool                  `json:"requires_attention"`
	ReadAt             string                `json:"read_at,omitempty"`
	CreatedAt          string                `json:"created_at"`
}

func FormatAppID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	normalized, err := idutil.Normalize(id)
	if err != nil {
		return id
	}
	return normalized
}

func ParseAppID(raw string) (string, error) {
	id, err := idutil.Normalize(raw)
	if err != nil {
		return "", fmt.Errorf("app id: %w", err)
	}
	return id, nil
}

func APIJob(in orchestrator.Job) Job {
	return Job{
		ID:          in.ID,
		ServerID:    FormatAppID(in.ServerID),
		Kind:        in.Kind,
		Status:      in.Status,
		CurrentStep: in.CurrentStep,
		RetryCount:  in.RetryCount,
		LastError:   in.LastError,
		Payload:     in.Payload,
		StartedAt:   in.StartedAt,
		FinishedAt:  in.FinishedAt,
		TimeoutAt:   in.TimeoutAt,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		CommandID:   in.CommandID,
	}
}

func APIJobs(in []orchestrator.Job) []Job {
	out := make([]Job, 0, len(in))
	for _, item := range in {
		out = append(out, APIJob(item))
	}
	return out
}

func APIActivity(in activity.Activity) Activity {
	return Activity{
		ID:                 in.ID,
		EventType:          in.EventType,
		Category:           in.Category,
		Level:              in.Level,
		ResourceType:       in.ResourceType,
		ResourceID:         FormatAppID(in.ResourceID),
		ParentResourceType: in.ParentResourceType,
		ParentResourceID:   FormatAppID(in.ParentResourceID),
		ActorType:          in.ActorType,
		ActorID:            in.ActorID,
		Title:              in.Title,
		Message:            in.Message,
		Payload:            in.Payload,
		RequiresAttention:  in.RequiresAttention,
		ReadAt:             in.ReadAt,
		CreatedAt:          in.CreatedAt,
	}
}

func APIActivities(in []activity.Activity) []Activity {
	out := make([]Activity, 0, len(in))
	for _, item := range in {
		out = append(out, APIActivity(item))
	}
	return out
}

var PublishedTypes = map[string]any{
	"LoginRequest":            LoginRequest{},
	"StatusResponse":          StatusResponse{},
	"HealthResponse":          HealthResponse{},
	"CreateProviderRequest":   CreateProviderRequest{},
	"ValidateProviderRequest": ValidateProviderRequest{},
	"CreateProviderResponse":  CreateProviderResponse{},
	"ProviderType":            provider.Info{},
	"StoredProvider":          provider.StoredProvider{},
	"ValidationResult":        provider.ValidationResult{},
	"CreateServerRequest":     CreateServerRequest{},
	"CreateSiteRequest":       CreateSiteRequest{},
	"CreateDomainRequest":     CreateDomainRequest{},
	"ServerCatalogResponse":   ServerCatalogResponse{},
	"CreateServerResponse":    CreateServerResponse{},
	"StoredSite":              StoredSite{},
	"SiteHealthCheck":         agentcommand.SiteHealthCheck{},
	"SiteHealthSnapshot":      agentcommand.SiteHealthSnapshot{},
	"SiteHealthResponse":      SiteHealthResponse{},
	"StoredDomain":            StoredDomain{},
	"DeleteSiteResponse":      DeleteSiteResponse{},
	"DeleteDomainResponse":    DeleteDomainResponse{},
	"DeleteServerResponse":    DeleteServerResponse{},
	"UpdateSiteRequest":       UpdateSiteRequest{},
	"UpdateDomainRequest":     UpdateDomainRequest{},
	"RebuildOptionsResponse":  RebuildOptionsResponse{},
	"ResizeOptionsResponse":   ResizeOptionsResponse{},
	"FirewallsResponse":       FirewallsResponse{},
	"VolumesResponse":         VolumesResponse{},
	"ServerProfile":           profiles.Profile{},
	"ServerCatalog":           provider.ServerCatalog{},
	"ServerLocation":          provider.ServerLocation{},
	"ServerTypePrice":         provider.ServerTypePrice{},
	"ServerTypeOption":        provider.ServerTypeOption{},
	"StoredServer":            StoredServer{},
	"AgentInfo":               ws.AgentInfo{},
	"AgentStatusMapResponse":  AgentStatusMapResponse{},
	"Service":                 agentcommand.Service{},
	"ServicesResponse":        ServicesResponse{},
	"AuthActor":               auth.Actor{},
	"CreateJobRequest":        CreateJobRequest{},
	"Job":                     Job{},
	"JobEvent":                orchestrator.JobEvent{},
	"Activity":                Activity{},
	"ActivityListResponse":    ActivityListResponse{},
	"UnreadCountResponse":     UnreadCountResponse{},
}
