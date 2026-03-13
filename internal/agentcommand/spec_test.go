package agentcommand

import (
	"encoding/json"
	"testing"
)

func TestValidateRestartServiceAcceptsAllowedService(t *testing.T) {
	payload, err := json.Marshal(RestartServiceParams{ServiceName: " nginx "})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	normalized, err := Validate(TypeRestartService, payload)
	if err != nil {
		t.Fatalf("validate payload: %v", err)
	}

	var decoded RestartServiceParams
	if err := json.Unmarshal(normalized, &decoded); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}
	if decoded.ServiceName != "nginx" {
		t.Fatalf("service_name = %q, want nginx", decoded.ServiceName)
	}
}

func TestValidateRestartServiceRejectsDisallowedService(t *testing.T) {
	payload, err := json.Marshal(RestartServiceParams{ServiceName: "sshd"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	_, err = Validate(TypeRestartService, payload)
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if validationErr.Code != ErrorCodeServiceNotAllowed {
		t.Fatalf("code = %q, want %q", validationErr.Code, ErrorCodeServiceNotAllowed)
	}
}

func TestValidateListServicesRejectsUnexpectedPayload(t *testing.T) {
	_, err := Validate(TypeListServices, json.RawMessage(`{"unexpected":true}`))
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if validationErr.Code != ErrorCodeInvalidPayload {
		t.Fatalf("code = %q, want %q", validationErr.Code, ErrorCodeInvalidPayload)
	}
}

func TestValidateSiteHealthAcceptsRequiredFields(t *testing.T) {
	payload, err := json.Marshal(SiteHealthSnapshotParams{SiteID: "site-1", Hostname: "example.testable.io", SitePath: "/srv/www/site"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	normalized, err := Validate(TypeSiteHealth, payload)
	if err != nil {
		t.Fatalf("validate payload: %v", err)
	}

	var decoded SiteHealthSnapshotParams
	if err := json.Unmarshal(normalized, &decoded); err != nil {
		t.Fatalf("unmarshal normalized payload: %v", err)
	}
	if decoded.Hostname != "example.testable.io" {
		t.Fatalf("hostname = %q, want %q", decoded.Hostname, "example.testable.io")
	}
}

func TestValidateSiteHealthRejectsMissingHostname(t *testing.T) {
	_, err := Validate(TypeSiteHealth, json.RawMessage(`{"site_id":"site-1","site_path":"/srv/www/site"}`))
	validationErr, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("err = %v, want ValidationError", err)
	}
	if validationErr.Code != ErrorCodeInvalidPayload {
		t.Fatalf("code = %q, want %q", validationErr.Code, ErrorCodeInvalidPayload)
	}
}
