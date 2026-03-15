package agentcommand

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	TypeRestartService = "restart_service"
	TypeListServices   = "list_services"
	TypeSiteHealth     = "site_health_snapshot"

	ErrorCodeUnknownCommand      = "unknown_command"
	ErrorCodeInvalidPayload      = "invalid_payload"
	ErrorCodeInvalidServiceName  = "invalid_service_name"
	ErrorCodeServiceNotAllowed   = "service_not_allowed"
	ErrorCodeExecutionFailed     = "execution_failed"
	ErrorCodeCommandTimedOut     = "command_timed_out"
	ErrorCodeSerializationFailed = "serialization_failed"
)

type Spec struct {
	Type     string
	Timeout  time.Duration
	Validate func(json.RawMessage) (json.RawMessage, error)
}

type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type RestartServiceParams struct {
	ServiceName string `json:"service_name"`
}

type RestartServiceResult struct {
	ServiceName string `json:"service_name"`
	Action      string `json:"action"`
}

type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ActiveState string `json:"active_state"`
	LoadState   string `json:"load_state"`
}

type ListServicesResult struct {
	Services []Service `json:"services"`
}

type SiteHealthSnapshotParams struct {
	SiteID   string `json:"site_id"`
	Hostname string `json:"hostname"`
	SitePath string `json:"site_path"`
}

type SiteHealthCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

type SiteHealthSnapshot struct {
	SiteID       string            `json:"site_id"`
	Hostname     string            `json:"hostname"`
	GeneratedAt  string            `json:"generated_at"`
	Healthy      bool              `json:"healthy"`
	Summary      string            `json:"summary"`
	Services     []Service         `json:"services,omitempty"`
	Checks       []SiteHealthCheck `json:"checks,omitempty"`
	RecentErrors []string          `json:"recent_errors,omitempty"`
}

var serviceNamePattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._@-]{0,127}$`)

var allowedServiceNames = map[string]struct{}{
	"nginx":           {},
	"php8.3-fpm":      {},
	"pressluft-agent": {},
	"redis-server":    {},
}

var specs = map[string]Spec{
	TypeRestartService: {Type: TypeRestartService, Timeout: 2 * time.Minute, Validate: validateRestartServicePayload},
	TypeListServices:   {Type: TypeListServices, Timeout: 10 * time.Second, Validate: validateEmptyPayload},
	TypeSiteHealth:     {Type: TypeSiteHealth, Timeout: 20 * time.Second, Validate: validateSiteHealthPayload},
}

func Lookup(commandType string) (Spec, bool) {
	spec, ok := specs[strings.TrimSpace(commandType)]
	return spec, ok
}

func Timeout(commandType string) time.Duration {
	spec, ok := Lookup(commandType)
	if !ok {
		return 0
	}
	return spec.Timeout
}

func Validate(commandType string, payload json.RawMessage) (json.RawMessage, error) {
	spec, ok := Lookup(commandType)
	if !ok {
		return nil, &ValidationError{Code: ErrorCodeUnknownCommand, Message: fmt.Sprintf("unknown command: %s", strings.TrimSpace(commandType))}
	}
	return spec.Validate(payload)
}

func DecodeRestartServicePayload(payload json.RawMessage) (RestartServiceParams, error) {
	normalized, err := validateRestartServicePayload(payload)
	if err != nil {
		return RestartServiceParams{}, err
	}
	var params RestartServiceParams
	if err := json.Unmarshal(normalized, &params); err != nil {
		return RestartServiceParams{}, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "invalid restart_service payload"}
	}
	return params, nil
}

func AllowedServiceNames() []string {
	out := make([]string, 0, len(allowedServiceNames))
	for name := range allowedServiceNames {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func DecodeSiteHealthPayload(payload json.RawMessage) (SiteHealthSnapshotParams, error) {
	normalized, err := validateSiteHealthPayload(payload)
	if err != nil {
		return SiteHealthSnapshotParams{}, err
	}
	var params SiteHealthSnapshotParams
	if err := json.Unmarshal(normalized, &params); err != nil {
		return SiteHealthSnapshotParams{}, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "invalid site_health_snapshot payload"}
	}
	return params, nil
}

func validateEmptyPayload(payload json.RawMessage) (json.RawMessage, error) {
	trimmed := strings.TrimSpace(string(payload))
	if trimmed == "" || trimmed == "null" {
		return nil, nil
	}
	return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "command does not accept a payload"}
}

func validateRestartServicePayload(payload json.RawMessage) (json.RawMessage, error) {
	if strings.TrimSpace(string(payload)) == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "restart_service payload is required"}
	}
	var params RestartServiceParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "invalid restart_service payload"}
	}
	params.ServiceName = strings.TrimSpace(params.ServiceName)
	if params.ServiceName == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "service_name is required"}
	}
	if !serviceNamePattern.MatchString(params.ServiceName) {
		return nil, &ValidationError{Code: ErrorCodeInvalidServiceName, Message: "service_name format is invalid"}
	}
	if _, ok := allowedServiceNames[params.ServiceName]; !ok {
		return nil, &ValidationError{Code: ErrorCodeServiceNotAllowed, Message: fmt.Sprintf("service_name %q is not allowed", params.ServiceName)}
	}
	normalized, err := json.Marshal(params)
	if err != nil {
		return nil, &ValidationError{Code: ErrorCodeSerializationFailed, Message: "failed to normalize restart_service payload"}
	}
	return normalized, nil
}

func validateSiteHealthPayload(payload json.RawMessage) (json.RawMessage, error) {
	if strings.TrimSpace(string(payload)) == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "site_health_snapshot payload is required"}
	}
	var params SiteHealthSnapshotParams
	if err := json.Unmarshal(payload, &params); err != nil {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "invalid site_health_snapshot payload"}
	}
	params.SiteID = strings.TrimSpace(params.SiteID)
	params.Hostname = strings.TrimSpace(params.Hostname)
	params.SitePath = strings.TrimSpace(params.SitePath)
	if params.SiteID == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "site_id is required"}
	}
	if params.Hostname == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "hostname is required"}
	}
	if params.SitePath == "" {
		return nil, &ValidationError{Code: ErrorCodeInvalidPayload, Message: "site_path is required"}
	}
	normalized, err := json.Marshal(params)
	if err != nil {
		return nil, &ValidationError{Code: ErrorCodeSerializationFailed, Message: "failed to normalize site_health_snapshot payload"}
	}
	return normalized, nil
}
