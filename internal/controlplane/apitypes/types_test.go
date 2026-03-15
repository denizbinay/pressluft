package apitypes

import (
	"testing"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/shared/idutil"
)

// --- LoginRequest ---

func TestLoginRequest_Validate_Valid(t *testing.T) {
	r := &LoginRequest{Email: "user@example.com", Password: "secret"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoginRequest_Validate_TrimsEmail(t *testing.T) {
	r := &LoginRequest{Email: "  user@example.com  ", Password: "secret"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Email != "user@example.com" {
		t.Fatalf("email = %q, want trimmed", r.Email)
	}
}

func TestLoginRequest_Validate_EmptyEmail(t *testing.T) {
	r := &LoginRequest{Email: "", Password: "secret"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestLoginRequest_Validate_WhitespaceEmail(t *testing.T) {
	r := &LoginRequest{Email: "   ", Password: "secret"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only email")
	}
}

func TestLoginRequest_Validate_EmptyPassword(t *testing.T) {
	r := &LoginRequest{Email: "user@example.com", Password: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestLoginRequest_Validate_WhitespacePassword(t *testing.T) {
	r := &LoginRequest{Email: "user@example.com", Password: "   "}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only password")
	}
}

func TestLoginRequest_Validate_BothEmpty(t *testing.T) {
	r := &LoginRequest{}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for both fields empty")
	}
}

// --- CreateProviderRequest ---

func TestCreateProviderRequest_Validate_Valid(t *testing.T) {
	r := &CreateProviderRequest{Type: "hetzner", Name: "my-provider", APIToken: "tok-123"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateProviderRequest_Validate_TrimsFields(t *testing.T) {
	r := &CreateProviderRequest{Type: "  hetzner  ", Name: "  my-provider  ", APIToken: "tok-123"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Type != "hetzner" {
		t.Fatalf("Type = %q, want trimmed", r.Type)
	}
	if r.Name != "my-provider" {
		t.Fatalf("Name = %q, want trimmed", r.Name)
	}
}

func TestCreateProviderRequest_Validate_EmptyType(t *testing.T) {
	r := &CreateProviderRequest{Type: "", Name: "my-provider", APIToken: "tok-123"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestCreateProviderRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateProviderRequest{Type: "hetzner", Name: "", APIToken: "tok-123"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreateProviderRequest_Validate_EmptyAPIToken(t *testing.T) {
	r := &CreateProviderRequest{Type: "hetzner", Name: "my-provider", APIToken: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty api_token")
	}
}

func TestCreateProviderRequest_Validate_WhitespaceAPIToken(t *testing.T) {
	r := &CreateProviderRequest{Type: "hetzner", Name: "my-provider", APIToken: "   "}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only api_token")
	}
}

func TestCreateProviderRequest_Validate_WhitespaceType(t *testing.T) {
	r := &CreateProviderRequest{Type: "   ", Name: "my-provider", APIToken: "tok-123"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only type")
	}
}

func TestCreateProviderRequest_Validate_AllEmpty(t *testing.T) {
	r := &CreateProviderRequest{}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for all fields empty")
	}
}

// --- ValidateProviderRequest ---

func TestValidateProviderRequest_Validate_Valid(t *testing.T) {
	r := &ValidateProviderRequest{Type: "hetzner", APIToken: "tok-123"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateProviderRequest_Validate_TrimsType(t *testing.T) {
	r := &ValidateProviderRequest{Type: "  hetzner  ", APIToken: "tok-123"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Type != "hetzner" {
		t.Fatalf("Type = %q, want trimmed", r.Type)
	}
}

func TestValidateProviderRequest_Validate_EmptyType(t *testing.T) {
	r := &ValidateProviderRequest{Type: "", APIToken: "tok-123"}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty type")
	}
}

func TestValidateProviderRequest_Validate_EmptyAPIToken(t *testing.T) {
	r := &ValidateProviderRequest{Type: "hetzner", APIToken: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty api_token")
	}
}

func TestValidateProviderRequest_Validate_WhitespaceAPIToken(t *testing.T) {
	r := &ValidateProviderRequest{Type: "hetzner", APIToken: "   "}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only api_token")
	}
}

// --- CreateDomainRequest ---

func TestCreateDomainRequest_Validate_Valid(t *testing.T) {
	r := &CreateDomainRequest{Hostname: "example.com"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDomainRequest_Validate_TrimsFields(t *testing.T) {
	r := &CreateDomainRequest{
		Hostname: "  example.com  ",
		Kind:     "  primary  ",
		Source:   "  manual  ",
		SiteID:   "  site-1  ",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Hostname != "example.com" {
		t.Fatalf("Hostname = %q, want trimmed", r.Hostname)
	}
	if r.Kind != "primary" {
		t.Fatalf("Kind = %q, want trimmed", r.Kind)
	}
	if r.Source != "manual" {
		t.Fatalf("Source = %q, want trimmed", r.Source)
	}
	if r.SiteID != "site-1" {
		t.Fatalf("SiteID = %q, want trimmed", r.SiteID)
	}
}

func TestCreateDomainRequest_Validate_EmptyHostname(t *testing.T) {
	r := &CreateDomainRequest{Hostname: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty hostname")
	}
}

func TestCreateDomainRequest_Validate_WhitespaceHostname(t *testing.T) {
	r := &CreateDomainRequest{Hostname: "   "}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only hostname")
	}
}

func TestCreateDomainRequest_Validate_OptionalFieldsEmpty(t *testing.T) {
	r := &CreateDomainRequest{Hostname: "example.com"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Optional fields should remain empty
	if r.Kind != "" {
		t.Fatalf("Kind = %q, want empty", r.Kind)
	}
}

// --- UpdateDomainRequest ---

func TestUpdateDomainRequest_Validate_AllNil(t *testing.T) {
	r := &UpdateDomainRequest{}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDomainRequest_Validate_ValidHostname(t *testing.T) {
	hostname := "example.com"
	r := &UpdateDomainRequest{Hostname: &hostname}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDomainRequest_Validate_TrimsHostname(t *testing.T) {
	hostname := "  example.com  "
	r := &UpdateDomainRequest{Hostname: &hostname}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *r.Hostname != "example.com" {
		t.Fatalf("Hostname = %q, want trimmed", *r.Hostname)
	}
}

func TestUpdateDomainRequest_Validate_EmptyHostname(t *testing.T) {
	hostname := ""
	r := &UpdateDomainRequest{Hostname: &hostname}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty hostname")
	}
}

func TestUpdateDomainRequest_Validate_WhitespaceHostname(t *testing.T) {
	hostname := "   "
	r := &UpdateDomainRequest{Hostname: &hostname}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only hostname")
	}
}

func TestUpdateDomainRequest_Validate_TrimsOptionalFields(t *testing.T) {
	kind := "  primary  "
	source := "  manual  "
	r := &UpdateDomainRequest{Kind: &kind, Source: &source}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *r.Kind != "primary" {
		t.Fatalf("Kind = %q, want trimmed", *r.Kind)
	}
	if *r.Source != "manual" {
		t.Fatalf("Source = %q, want trimmed", *r.Source)
	}
}

func TestUpdateDomainRequest_Validate_NilHostnameIsAllowed(t *testing.T) {
	kind := "alias"
	r := &UpdateDomainRequest{Kind: &kind}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- CreateSiteRequest ---

func TestCreateSiteRequest_Validate_Valid(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSiteRequest_Validate_TrimsFields(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "  srv-1  ",
		Name:                "  my-site  ",
		WordPressAdminEmail: "  admin@example.com  ",
		PrimaryDomain:       "  example.com  ",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.ServerID != "srv-1" {
		t.Fatalf("ServerID = %q, want trimmed", r.ServerID)
	}
	if r.Name != "my-site" {
		t.Fatalf("Name = %q, want trimmed", r.Name)
	}
	if r.WordPressAdminEmail != "admin@example.com" {
		t.Fatalf("WordPressAdminEmail = %q, want trimmed", r.WordPressAdminEmail)
	}
	if r.PrimaryDomain != "example.com" {
		t.Fatalf("PrimaryDomain = %q, want trimmed", r.PrimaryDomain)
	}
}

func TestCreateSiteRequest_Validate_EmptyServerID(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty server_id")
	}
}

func TestCreateSiteRequest_Validate_WhitespaceServerID(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "   ",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only server_id")
	}
}

func TestCreateSiteRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestCreateSiteRequest_Validate_EmptyEmail(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty wordpress_admin_email")
	}
}

func TestCreateSiteRequest_Validate_InvalidEmail(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "not-an-email",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestCreateSiteRequest_Validate_InvalidEmailNoAt(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "user.example.com",
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for email without @")
	}
}

func TestCreateSiteRequest_Validate_BothPrimaryDomainAndConfig(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
		PrimaryDomain:       "example.com",
		PrimaryHostnameConfig: &SitePrimaryHostnameConfig{
			Source:   "user",
			Hostname: "example.com",
		},
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when both primary_domain and primary_hostname_config are set")
	}
}

func TestCreateSiteRequest_Validate_WithPrimaryDomain(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
		PrimaryDomain:       "example.com",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSiteRequest_Validate_WithValidHostnameConfig(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
		PrimaryHostnameConfig: &SitePrimaryHostnameConfig{
			Source:   "user",
			Hostname: "example.com",
		},
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSiteRequest_Validate_NilHostnameConfig(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- SitePrimaryHostnameConfig ---

func TestSitePrimaryHostnameConfig_Validate_Nil(t *testing.T) {
	var c *SitePrimaryHostnameConfig
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error for nil config: %v", err)
	}
}

func TestSitePrimaryHostnameConfig_Validate_FallbackResolver_Valid(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "fallback_resolver",
		Label:  "my-label",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSitePrimaryHostnameConfig_Validate_FallbackResolver_MissingLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "fallback_resolver",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for fallback_resolver without label")
	}
}

func TestSitePrimaryHostnameConfig_Validate_FallbackResolver_WhitespaceLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "fallback_resolver",
		Label:  "   ",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for fallback_resolver with whitespace-only label")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_Hostname(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		Hostname: "example.com",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_DomainIDAndLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		DomainID: "dom-1",
		Label:    "www",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_HostnameAndDomainID(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		Hostname: "example.com",
		DomainID: "dom-1",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when both hostname and domain_id are set")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_HostnameAndLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		Hostname: "example.com",
		Label:    "www",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when both hostname and label are set")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_HostnameDomainIDAndLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		Hostname: "example.com",
		DomainID: "dom-1",
		Label:    "www",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when hostname, domain_id, and label are all set")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_DomainIDWithoutLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "user",
		DomainID: "dom-1",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for domain_id without label")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_NeitherHostnameNorDomainID(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "user",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when neither hostname nor domain_id is set")
	}
}

func TestSitePrimaryHostnameConfig_Validate_User_OnlyLabel(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "user",
		Label:  "www",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error when only label is set without domain_id")
	}
}

func TestSitePrimaryHostnameConfig_Validate_InvalidSource(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "invalid",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for invalid source")
	}
}

func TestSitePrimaryHostnameConfig_Validate_EmptySource(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source: "",
	}
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for empty source")
	}
}

func TestSitePrimaryHostnameConfig_Validate_TrimsFields(t *testing.T) {
	c := &SitePrimaryHostnameConfig{
		Source:   "  user  ",
		Hostname: "  example.com  ",
	}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Source != "user" {
		t.Fatalf("Source = %q, want trimmed", c.Source)
	}
	if c.Hostname != "example.com" {
		t.Fatalf("Hostname = %q, want trimmed", c.Hostname)
	}
}

// --- UpdateSiteRequest ---

func TestUpdateSiteRequest_Validate_AllNil(t *testing.T) {
	r := &UpdateSiteRequest{}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSiteRequest_Validate_ValidName(t *testing.T) {
	name := "my-site"
	r := &UpdateSiteRequest{Name: &name}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSiteRequest_Validate_EmptyName(t *testing.T) {
	name := ""
	r := &UpdateSiteRequest{Name: &name}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestUpdateSiteRequest_Validate_WhitespaceName(t *testing.T) {
	name := "   "
	r := &UpdateSiteRequest{Name: &name}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only name")
	}
}

func TestUpdateSiteRequest_Validate_EmptyServerID(t *testing.T) {
	serverID := ""
	r := &UpdateSiteRequest{ServerID: &serverID}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty server_id")
	}
}

func TestUpdateSiteRequest_Validate_WhitespaceServerID(t *testing.T) {
	serverID := "   "
	r := &UpdateSiteRequest{ServerID: &serverID}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only server_id")
	}
}

func TestUpdateSiteRequest_Validate_ValidServerID(t *testing.T) {
	serverID := "srv-1"
	r := &UpdateSiteRequest{ServerID: &serverID}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSiteRequest_Validate_EmptyEmail(t *testing.T) {
	email := ""
	r := &UpdateSiteRequest{WordPressAdminEmail: &email}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestUpdateSiteRequest_Validate_InvalidEmail(t *testing.T) {
	email := "not-an-email"
	r := &UpdateSiteRequest{WordPressAdminEmail: &email}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for invalid email")
	}
}

func TestUpdateSiteRequest_Validate_ValidEmail(t *testing.T) {
	email := "admin@example.com"
	r := &UpdateSiteRequest{WordPressAdminEmail: &email}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateSiteRequest_Validate_TrimsFields(t *testing.T) {
	name := "  my-site  "
	serverID := "  srv-1  "
	email := "  admin@example.com  "
	phpVersion := "  8.3  "
	r := &UpdateSiteRequest{
		Name:                &name,
		ServerID:            &serverID,
		WordPressAdminEmail: &email,
		PHPVersion:          &phpVersion,
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *r.Name != "my-site" {
		t.Fatalf("Name = %q, want trimmed", *r.Name)
	}
	if *r.ServerID != "srv-1" {
		t.Fatalf("ServerID = %q, want trimmed", *r.ServerID)
	}
	if *r.WordPressAdminEmail != "admin@example.com" {
		t.Fatalf("WordPressAdminEmail = %q, want trimmed", *r.WordPressAdminEmail)
	}
	if *r.PHPVersion != "8.3" {
		t.Fatalf("PHPVersion = %q, want trimmed", *r.PHPVersion)
	}
}

func TestUpdateSiteRequest_Validate_WhitespaceEmail(t *testing.T) {
	email := "   "
	r := &UpdateSiteRequest{WordPressAdminEmail: &email}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only email")
	}
}

// --- CreateJobRequest ---

func TestCreateJobRequest_Validate_Valid(t *testing.T) {
	r := &CreateJobRequest{Kind: "deploy_site", ServerID: "srv-1"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateJobRequest_Validate_TrimsKind(t *testing.T) {
	r := &CreateJobRequest{Kind: "  deploy_site  "}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.Kind != "deploy_site" {
		t.Fatalf("Kind = %q, want trimmed", r.Kind)
	}
}

func TestCreateJobRequest_Validate_EmptyKind(t *testing.T) {
	r := &CreateJobRequest{Kind: ""}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for empty kind")
	}
}

func TestCreateJobRequest_Validate_WhitespaceKind(t *testing.T) {
	r := &CreateJobRequest{Kind: "   "}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for whitespace-only kind")
	}
}

func TestCreateJobRequest_Validate_NoServerID(t *testing.T) {
	r := &CreateJobRequest{Kind: "deploy_site"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- FormatAppID ---

func TestFormatAppID_Valid(t *testing.T) {
	id := idutil.MustNew()
	result := FormatAppID(id)
	if result != id {
		t.Fatalf("FormatAppID(%q) = %q, want %q", id, result, id)
	}
}

func TestFormatAppID_Empty(t *testing.T) {
	result := FormatAppID("")
	if result != "" {
		t.Fatalf("FormatAppID(\"\") = %q, want empty", result)
	}
}

func TestFormatAppID_Whitespace(t *testing.T) {
	result := FormatAppID("   ")
	if result != "" {
		t.Fatalf("FormatAppID(\"   \") = %q, want empty", result)
	}
}

func TestFormatAppID_InvalidUUID(t *testing.T) {
	// Not a valid UUID -- should return the input as-is (trimmed)
	result := FormatAppID("not-a-uuid")
	if result != "not-a-uuid" {
		t.Fatalf("FormatAppID(\"not-a-uuid\") = %q, want \"not-a-uuid\"", result)
	}
}

func TestFormatAppID_TrimsInput(t *testing.T) {
	id := idutil.MustNew()
	result := FormatAppID("  " + id + "  ")
	if result != id {
		t.Fatalf("FormatAppID with spaces = %q, want %q", result, id)
	}
}

func TestFormatAppID_NonV7UUID(t *testing.T) {
	// UUIDv4 should cause Normalize to fail; FormatAppID returns trimmed input
	v4 := "550e8400-e29b-41d4-a716-446655440000"
	result := FormatAppID(v4)
	if result != v4 {
		t.Fatalf("FormatAppID(v4) = %q, want %q", result, v4)
	}
}

// --- ParseAppID ---

func TestParseAppID_Valid(t *testing.T) {
	id := idutil.MustNew()
	result, err := ParseAppID(id)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != id {
		t.Fatalf("ParseAppID(%q) = %q, want %q", id, result, id)
	}
}

func TestParseAppID_InvalidUUID(t *testing.T) {
	_, err := ParseAppID("not-a-uuid")
	if err == nil {
		t.Fatal("expected error for invalid UUID")
	}
}

func TestParseAppID_Empty(t *testing.T) {
	_, err := ParseAppID("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestParseAppID_NonV7UUID(t *testing.T) {
	_, err := ParseAppID("550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Fatal("expected error for non-v7 UUID")
	}
}

func TestParseAppID_TrimsWhitespace(t *testing.T) {
	id := idutil.MustNew()
	result, err := ParseAppID("  " + id + "  ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != id {
		t.Fatalf("ParseAppID with spaces = %q, want %q", result, id)
	}
}

// --- APIJob ---

func TestAPIJob_MapsFields(t *testing.T) {
	serverID := idutil.MustNew()
	cmdID := "cmd-123"
	in := orchestrator.Job{
		ID:          "job-1",
		ServerID:    serverID,
		Kind:        "deploy_site",
		Status:      orchestrator.JobStatusQueued,
		CurrentStep: "step1",
		RetryCount:  2,
		LastError:   "some error",
		Payload:     `{"key":"value"}`,
		StartedAt:   "2026-01-01T00:00:00Z",
		FinishedAt:  "2026-01-01T01:00:00Z",
		TimeoutAt:   "2026-01-01T02:00:00Z",
		CreatedAt:   "2026-01-01T00:00:00Z",
		UpdatedAt:   "2026-01-01T00:00:00Z",
		CommandID:   &cmdID,
	}
	out := APIJob(in)
	if out.ID != "job-1" {
		t.Fatalf("ID = %q, want %q", out.ID, "job-1")
	}
	if out.ServerID != serverID {
		t.Fatalf("ServerID = %q, want %q", out.ServerID, serverID)
	}
	if out.Kind != "deploy_site" {
		t.Fatalf("Kind = %q, want %q", out.Kind, "deploy_site")
	}
	if out.Status != orchestrator.JobStatusQueued {
		t.Fatalf("Status = %q, want %q", out.Status, orchestrator.JobStatusQueued)
	}
	if out.CurrentStep != "step1" {
		t.Fatalf("CurrentStep = %q, want %q", out.CurrentStep, "step1")
	}
	if out.RetryCount != 2 {
		t.Fatalf("RetryCount = %d, want %d", out.RetryCount, 2)
	}
	if out.LastError != "some error" {
		t.Fatalf("LastError = %q, want %q", out.LastError, "some error")
	}
	if out.Payload != `{"key":"value"}` {
		t.Fatalf("Payload = %q, want %q", out.Payload, `{"key":"value"}`)
	}
	if out.StartedAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("StartedAt = %q, want %q", out.StartedAt, "2026-01-01T00:00:00Z")
	}
	if out.FinishedAt != "2026-01-01T01:00:00Z" {
		t.Fatalf("FinishedAt = %q, want %q", out.FinishedAt, "2026-01-01T01:00:00Z")
	}
	if out.CommandID == nil || *out.CommandID != "cmd-123" {
		t.Fatalf("CommandID = %v, want %q", out.CommandID, "cmd-123")
	}
}

func TestAPIJob_EmptyServerID(t *testing.T) {
	in := orchestrator.Job{
		ID:        "job-1",
		ServerID:  "",
		Kind:      "deploy_site",
		Status:    orchestrator.JobStatusQueued,
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	out := APIJob(in)
	if out.ServerID != "" {
		t.Fatalf("ServerID = %q, want empty", out.ServerID)
	}
}

func TestAPIJob_NilCommandID(t *testing.T) {
	in := orchestrator.Job{
		ID:        "job-1",
		Kind:      "deploy_site",
		Status:    orchestrator.JobStatusQueued,
		CreatedAt: "2026-01-01T00:00:00Z",
		UpdatedAt: "2026-01-01T00:00:00Z",
	}
	out := APIJob(in)
	if out.CommandID != nil {
		t.Fatalf("CommandID = %v, want nil", out.CommandID)
	}
}

// --- APIJobs ---

func TestAPIJobs_Empty(t *testing.T) {
	out := APIJobs(nil)
	if len(out) != 0 {
		t.Fatalf("len = %d, want 0", len(out))
	}
}

func TestAPIJobs_Multiple(t *testing.T) {
	in := []orchestrator.Job{
		{ID: "job-1", Kind: "deploy_site", Status: orchestrator.JobStatusQueued, CreatedAt: "t", UpdatedAt: "t"},
		{ID: "job-2", Kind: "delete_server", Status: orchestrator.JobStatusRunning, CreatedAt: "t", UpdatedAt: "t"},
	}
	out := APIJobs(in)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].ID != "job-1" {
		t.Fatalf("out[0].ID = %q, want %q", out[0].ID, "job-1")
	}
	if out[1].ID != "job-2" {
		t.Fatalf("out[1].ID = %q, want %q", out[1].ID, "job-2")
	}
}

func TestAPIJobs_PreservesOrder(t *testing.T) {
	in := []orchestrator.Job{
		{ID: "c", Kind: "k", CreatedAt: "t", UpdatedAt: "t"},
		{ID: "a", Kind: "k", CreatedAt: "t", UpdatedAt: "t"},
		{ID: "b", Kind: "k", CreatedAt: "t", UpdatedAt: "t"},
	}
	out := APIJobs(in)
	if out[0].ID != "c" || out[1].ID != "a" || out[2].ID != "b" {
		t.Fatalf("order not preserved: %v", []string{out[0].ID, out[1].ID, out[2].ID})
	}
}

// --- APIActivity ---

func TestAPIActivity_MapsFields(t *testing.T) {
	resourceID := idutil.MustNew()
	parentResourceID := idutil.MustNew()
	in := activity.Activity{
		ID:                 "act-1",
		EventType:          activity.EventJobCreated,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         resourceID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   parentResourceID,
		ActorType:          activity.ActorSystem,
		ActorID:            "system",
		Title:              "Job Created",
		Message:            "A job was created",
		Payload:            `{"key":"value"}`,
		RequiresAttention:  true,
		ReadAt:             "2026-01-01T00:00:00Z",
		CreatedAt:          "2026-01-01T00:00:00Z",
	}
	out := APIActivity(in)
	if out.ID != "act-1" {
		t.Fatalf("ID = %q, want %q", out.ID, "act-1")
	}
	if out.EventType != activity.EventJobCreated {
		t.Fatalf("EventType = %q, want %q", out.EventType, activity.EventJobCreated)
	}
	if out.Category != activity.CategoryJob {
		t.Fatalf("Category = %q, want %q", out.Category, activity.CategoryJob)
	}
	if out.Level != activity.LevelInfo {
		t.Fatalf("Level = %q, want %q", out.Level, activity.LevelInfo)
	}
	if out.ResourceType != activity.ResourceJob {
		t.Fatalf("ResourceType = %q, want %q", out.ResourceType, activity.ResourceJob)
	}
	if out.ResourceID != resourceID {
		t.Fatalf("ResourceID = %q, want %q", out.ResourceID, resourceID)
	}
	if out.ParentResourceType != activity.ResourceServer {
		t.Fatalf("ParentResourceType = %q, want %q", out.ParentResourceType, activity.ResourceServer)
	}
	if out.ParentResourceID != parentResourceID {
		t.Fatalf("ParentResourceID = %q, want %q", out.ParentResourceID, parentResourceID)
	}
	if out.ActorType != activity.ActorSystem {
		t.Fatalf("ActorType = %q, want %q", out.ActorType, activity.ActorSystem)
	}
	if out.Title != "Job Created" {
		t.Fatalf("Title = %q, want %q", out.Title, "Job Created")
	}
	if out.Message != "A job was created" {
		t.Fatalf("Message = %q, want %q", out.Message, "A job was created")
	}
	if out.RequiresAttention != true {
		t.Fatal("RequiresAttention = false, want true")
	}
	if out.ReadAt != "2026-01-01T00:00:00Z" {
		t.Fatalf("ReadAt = %q, want %q", out.ReadAt, "2026-01-01T00:00:00Z")
	}
}

func TestAPIActivity_FormatsResourceIDs(t *testing.T) {
	resourceID := idutil.MustNew()
	in := activity.Activity{
		ID:         "act-1",
		EventType:  activity.EventJobCreated,
		Category:   activity.CategoryJob,
		Level:      activity.LevelInfo,
		ActorType:  activity.ActorSystem,
		ResourceID: resourceID,
		Title:      "test",
		CreatedAt:  "t",
	}
	out := APIActivity(in)
	if out.ResourceID != resourceID {
		t.Fatalf("ResourceID = %q, want %q", out.ResourceID, resourceID)
	}
}

func TestAPIActivity_EmptyResourceID(t *testing.T) {
	in := activity.Activity{
		ID:         "act-1",
		EventType:  activity.EventJobCreated,
		Category:   activity.CategoryJob,
		Level:      activity.LevelInfo,
		ActorType:  activity.ActorSystem,
		ResourceID: "",
		Title:      "test",
		CreatedAt:  "t",
	}
	out := APIActivity(in)
	if out.ResourceID != "" {
		t.Fatalf("ResourceID = %q, want empty", out.ResourceID)
	}
}

// --- APIActivities ---

func TestAPIActivities_Empty(t *testing.T) {
	out := APIActivities(nil)
	if len(out) != 0 {
		t.Fatalf("len = %d, want 0", len(out))
	}
}

func TestAPIActivities_Multiple(t *testing.T) {
	in := []activity.Activity{
		{ID: "act-1", EventType: activity.EventJobCreated, Category: activity.CategoryJob, Level: activity.LevelInfo, ActorType: activity.ActorSystem, Title: "t1", CreatedAt: "t"},
		{ID: "act-2", EventType: activity.EventJobCompleted, Category: activity.CategoryJob, Level: activity.LevelSuccess, ActorType: activity.ActorSystem, Title: "t2", CreatedAt: "t"},
	}
	out := APIActivities(in)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].ID != "act-1" {
		t.Fatalf("out[0].ID = %q, want %q", out[0].ID, "act-1")
	}
	if out[1].ID != "act-2" {
		t.Fatalf("out[1].ID = %q, want %q", out[1].ID, "act-2")
	}
}

func TestAPIActivities_PreservesOrder(t *testing.T) {
	in := []activity.Activity{
		{ID: "c", EventType: activity.EventJobCreated, Category: activity.CategoryJob, Level: activity.LevelInfo, ActorType: activity.ActorSystem, Title: "t", CreatedAt: "t"},
		{ID: "a", EventType: activity.EventJobCreated, Category: activity.CategoryJob, Level: activity.LevelInfo, ActorType: activity.ActorSystem, Title: "t", CreatedAt: "t"},
		{ID: "b", EventType: activity.EventJobCreated, Category: activity.CategoryJob, Level: activity.LevelInfo, ActorType: activity.ActorSystem, Title: "t", CreatedAt: "t"},
	}
	out := APIActivities(in)
	if out[0].ID != "c" || out[1].ID != "a" || out[2].ID != "b" {
		t.Fatalf("order not preserved: %v", []string{out[0].ID, out[1].ID, out[2].ID})
	}
}

// --- PublishedTypes ---

func TestPublishedTypes_ContainsExpectedEntries(t *testing.T) {
	expectedKeys := []string{
		"LoginRequest",
		"StatusResponse",
		"HealthResponse",
		"CreateProviderRequest",
		"ValidateProviderRequest",
		"CreateProviderResponse",
		"CreateServerRequest",
		"CreateSiteRequest",
		"CreateDomainRequest",
		"StoredSite",
		"StoredDomain",
		"StoredServer",
		"DeleteSiteResponse",
		"DeleteDomainResponse",
		"DeleteServerResponse",
		"UpdateSiteRequest",
		"UpdateDomainRequest",
		"CreateJobRequest",
		"Job",
		"Activity",
		"ActivityListResponse",
		"UnreadCountResponse",
	}
	for _, key := range expectedKeys {
		if _, ok := PublishedTypes[key]; !ok {
			t.Errorf("PublishedTypes missing key %q", key)
		}
	}
}

func TestPublishedTypes_NotEmpty(t *testing.T) {
	if len(PublishedTypes) == 0 {
		t.Fatal("PublishedTypes is empty")
	}
}

// --- Edge cases: special characters ---

func TestCreateSiteRequest_Validate_SpecialCharactersInName(t *testing.T) {
	r := &CreateSiteRequest{
		ServerID:            "srv-1",
		Name:                "my-site_with.special-chars!@#",
		WordPressAdminEmail: "admin@example.com",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error for special chars in name: %v", err)
	}
}

func TestCreateProviderRequest_Validate_SpecialCharactersInName(t *testing.T) {
	r := &CreateProviderRequest{
		Type:     "hetzner",
		Name:     "my provider (production) #1",
		APIToken: "tok-abc-123",
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error for special chars in name: %v", err)
	}
}

func TestCreateDomainRequest_Validate_SubdomainHostname(t *testing.T) {
	r := &CreateDomainRequest{Hostname: "sub.domain.example.com"}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Edge cases: very long strings ---

func TestLoginRequest_Validate_LongEmail(t *testing.T) {
	longEmail := make([]byte, 500)
	for i := range longEmail {
		longEmail[i] = 'a'
	}
	r := &LoginRequest{Email: string(longEmail), Password: "secret"}
	// Should not fail validation; length limits are not enforced at this layer
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error for long email: %v", err)
	}
}

func TestCreateProviderRequest_Validate_LongAPIToken(t *testing.T) {
	longToken := make([]byte, 10000)
	for i := range longToken {
		longToken[i] = 'x'
	}
	r := &CreateProviderRequest{Type: "hetzner", Name: "my-provider", APIToken: string(longToken)}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error for long api_token: %v", err)
	}
}

// --- Table-driven tests for CreateSiteRequest email validation ---

func TestCreateSiteRequest_Validate_EmailVariations(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{"valid simple", "user@example.com", false},
		{"valid with plus", "user+tag@example.com", false},
		{"valid with dots", "first.last@example.com", false},
		{"empty", "", true},
		{"no at sign", "userexample.com", true},
		{"no domain", "user@", true},
		{"whitespace only", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &CreateSiteRequest{
				ServerID:            "srv-1",
				Name:                "my-site",
				WordPressAdminEmail: tt.email,
			}
			err := r.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("email=%q: error=%v, wantErr=%v", tt.email, err, tt.wantErr)
			}
		})
	}
}

// --- Table-driven tests for SitePrimaryHostnameConfig ---

func TestSitePrimaryHostnameConfig_Validate_TableDriven(t *testing.T) {
	tests := []struct {
		name    string
		config  *SitePrimaryHostnameConfig
		wantErr bool
	}{
		{"nil config", nil, false},
		{"fallback_resolver with label", &SitePrimaryHostnameConfig{Source: "fallback_resolver", Label: "my-label"}, false},
		{"fallback_resolver without label", &SitePrimaryHostnameConfig{Source: "fallback_resolver"}, true},
		{"user with hostname only", &SitePrimaryHostnameConfig{Source: "user", Hostname: "example.com"}, false},
		{"user with domain_id and label", &SitePrimaryHostnameConfig{Source: "user", DomainID: "d1", Label: "www"}, false},
		{"user with hostname and domain_id", &SitePrimaryHostnameConfig{Source: "user", Hostname: "h", DomainID: "d"}, true},
		{"user with hostname and label", &SitePrimaryHostnameConfig{Source: "user", Hostname: "h", Label: "l"}, true},
		{"user with domain_id only", &SitePrimaryHostnameConfig{Source: "user", DomainID: "d1"}, true},
		{"user with nothing", &SitePrimaryHostnameConfig{Source: "user"}, true},
		{"unknown source", &SitePrimaryHostnameConfig{Source: "unknown"}, true},
		{"empty source", &SitePrimaryHostnameConfig{Source: ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("config=%+v: error=%v, wantErr=%v", tt.config, err, tt.wantErr)
			}
		})
	}
}
