package orchestrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"pressluft/internal/agent/agentcommand"
)

type ConfigureServerPayload struct {
	IPv4 string `json:"ipv4,omitempty"`
}

type DeleteServerPayload struct{}

type RebuildServerPayload struct {
	ServerImage string `json:"server_image,omitempty"`
}

type ResizeServerPayload struct {
	ServerType  string `json:"server_type"`
	UpgradeDisk bool   `json:"upgrade_disk"`
}

type UpdateFirewallsPayload struct {
	Firewalls []string `json:"firewalls"`
}

type ManageVolumePayload struct {
	VolumeName string `json:"volume_name"`
	SizeGB     int    `json:"size_gb,omitempty"`
	Location   string `json:"location,omitempty"`
	State      string `json:"state"`
	Automount  *bool  `json:"automount,omitempty"`
}

type DeploySitePayload struct {
	SiteID          string `json:"site_id"`
	TLSContactEmail string `json:"tls_contact_email,omitempty"`
}

func MarshalConfigureServerPayload(in ConfigureServerPayload) (string, error) {
	return marshalNormalizedPayload(in)
}

func UnmarshalConfigureServerPayload(raw string) (ConfigureServerPayload, error) {
	var out ConfigureServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ConfigureServerPayload{}, err
	}
	out.IPv4 = strings.TrimSpace(out.IPv4)
	return out, nil
}

func MarshalDeleteServerPayload() (string, error) {
	return marshalNormalizedPayload(DeleteServerPayload{})
}

func UnmarshalDeleteServerPayload(raw string) (DeleteServerPayload, error) {
	var out DeleteServerPayload
	return out, unmarshalNormalizedPayload(raw, &out)
}

func MarshalRebuildServerPayload(in RebuildServerPayload) (string, error) {
	in.ServerImage = strings.TrimSpace(in.ServerImage)
	if in.ServerImage == "" {
		return marshalNormalizedPayload(struct{}{})
	}
	return marshalNormalizedPayload(in)
}

func UnmarshalRebuildServerPayload(raw string) (RebuildServerPayload, error) {
	var out RebuildServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return RebuildServerPayload{}, err
	}
	out.ServerImage = strings.TrimSpace(out.ServerImage)
	return out, nil
}

func MarshalResizeServerPayload(in ResizeServerPayload) (string, error) {
	in.ServerType = strings.TrimSpace(in.ServerType)
	return marshalNormalizedPayload(in)
}

func UnmarshalResizeServerPayload(raw string) (ResizeServerPayload, error) {
	var out ResizeServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ResizeServerPayload{}, err
	}
	out.ServerType = strings.TrimSpace(out.ServerType)
	return out, nil
}

func MarshalUpdateFirewallsPayload(in UpdateFirewallsPayload) (string, error) {
	firewalls := make([]string, 0, len(in.Firewalls))
	for _, firewall := range in.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	in.Firewalls = firewalls
	return marshalNormalizedPayload(in)
}

func UnmarshalUpdateFirewallsPayload(raw string) (UpdateFirewallsPayload, error) {
	var out UpdateFirewallsPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return UpdateFirewallsPayload{}, err
	}
	firewalls := make([]string, 0, len(out.Firewalls))
	for _, firewall := range out.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	out.Firewalls = firewalls
	return out, nil
}

func MarshalManageVolumePayload(in ManageVolumePayload) (string, error) {
	in.VolumeName = strings.TrimSpace(in.VolumeName)
	in.Location = strings.TrimSpace(in.Location)
	in.State = strings.TrimSpace(in.State)
	return marshalNormalizedPayload(in)
}

func UnmarshalManageVolumePayload(raw string) (ManageVolumePayload, error) {
	var out ManageVolumePayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ManageVolumePayload{}, err
	}
	out.VolumeName = strings.TrimSpace(out.VolumeName)
	out.Location = strings.TrimSpace(out.Location)
	out.State = strings.TrimSpace(out.State)
	return out, nil
}

func MarshalDeploySitePayload(in DeploySitePayload) (string, error) {
	in.SiteID = strings.TrimSpace(in.SiteID)
	in.TLSContactEmail = strings.TrimSpace(in.TLSContactEmail)
	return marshalNormalizedPayload(in)
}

func UnmarshalDeploySitePayload(raw string) (DeploySitePayload, error) {
	var out DeploySitePayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return DeploySitePayload{}, err
	}
	out.SiteID = strings.TrimSpace(out.SiteID)
	out.TLSContactEmail = strings.TrimSpace(out.TLSContactEmail)
	return out, nil
}

func marshalNormalizedPayload(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal normalized payload: %w", err)
	}
	if string(data) == "null" {
		return "", nil
	}
	return string(data), nil
}

func unmarshalNormalizedPayload(raw string, target any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("invalid normalized job payload: %w", err)
	}
	return nil
}

func normalizeArbitraryPayload(payload json.RawMessage) string {
	trimmed := strings.TrimSpace(string(bytes.TrimSpace(payload)))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	return trimmed
}

func requireServerID(serverID string, kind JobKind) error {
	if strings.TrimSpace(serverID) == "" {
		return fmt.Errorf("server_id is required for %s job", kind)
	}
	return nil
}

func validateProvisionServerPayload(payload json.RawMessage, _ string) (string, error) {
	return normalizeArbitraryPayload(payload), nil
}

func validateConfigureServerPayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindConfigureServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return MarshalConfigureServerPayload(ConfigureServerPayload{})
	}
	var parsed struct {
		IPv4 string `json:"ipv4"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid configure_server payload: %w", err)
	}
	return MarshalConfigureServerPayload(ConfigureServerPayload{IPv4: parsed.IPv4})
}

func validateDeleteServerPayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindDeleteServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return "", nil
	}
	var parsed DeleteServerPayload
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid delete_server payload: %w", err)
	}
	return MarshalDeleteServerPayload()
}

func validateRebuildServerPayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindRebuildServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return MarshalRebuildServerPayload(RebuildServerPayload{})
	}
	var parsed RebuildServerPayload
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid rebuild_server payload: %w", err)
	}
	return MarshalRebuildServerPayload(parsed)
}

func validateResizeServerPayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindResizeServer); err != nil {
		return "", err
	}
	var parsed struct {
		ServerType  string `json:"server_type"`
		UpgradeDisk *bool  `json:"upgrade_disk"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid resize_server payload: %w", err)
	}
	if strings.TrimSpace(parsed.ServerType) == "" {
		return "", fmt.Errorf("server_type is required for resize_server job")
	}
	if parsed.UpgradeDisk == nil {
		return "", fmt.Errorf("upgrade_disk is required for resize_server job")
	}
	return MarshalResizeServerPayload(ResizeServerPayload{
		ServerType:  parsed.ServerType,
		UpgradeDisk: *parsed.UpgradeDisk,
	})
}

func validateUpdateFirewallsPayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindUpdateFirewalls); err != nil {
		return "", err
	}
	var parsed UpdateFirewallsPayload
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid update_firewalls payload: %w", err)
	}
	firewalls := make([]string, 0, len(parsed.Firewalls))
	for _, firewall := range parsed.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	if len(firewalls) == 0 {
		return "", fmt.Errorf("firewalls payload must contain at least one firewall")
	}
	return MarshalUpdateFirewallsPayload(UpdateFirewallsPayload{Firewalls: firewalls})
}

func validateManageVolumePayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindManageVolume); err != nil {
		return "", err
	}
	var parsed ManageVolumePayload
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid manage_volume payload: %w", err)
	}
	volumeName := strings.TrimSpace(parsed.VolumeName)
	state := strings.TrimSpace(parsed.State)
	if volumeName == "" {
		return "", fmt.Errorf("volume_name is required for manage_volume job")
	}
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent for manage_volume job")
	}
	if state == "present" {
		if parsed.Automount == nil {
			return "", fmt.Errorf("automount is required for manage_volume job when state=present")
		}
		if parsed.SizeGB <= 0 {
			return "", fmt.Errorf("size_gb is required for manage_volume job when state=present")
		}
	}
	return MarshalManageVolumePayload(parsed)
}

func validateRestartServicePayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindRestartService); err != nil {
		return "", err
	}
	normalizedPayload, err := agentcommand.Validate(string(JobKindRestartService), bytes.TrimSpace(defaultPayloadObject(payload)))
	if err != nil {
		return "", fmt.Errorf("invalid restart_service payload: %w", err)
	}
	return string(normalizedPayload), nil
}

func validateDeploySitePayload(payload json.RawMessage, serverID string) (string, error) {
	if err := requireServerID(serverID, JobKindDeploySite); err != nil {
		return "", err
	}
	var parsed DeploySitePayload
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid deploy_site payload: %w", err)
	}
	if strings.TrimSpace(parsed.SiteID) == "" {
		return "", fmt.Errorf("site_id is required for deploy_site job")
	}
	return MarshalDeploySitePayload(parsed)
}

func defaultPayloadObject(payload json.RawMessage) []byte {
	if normalizeArbitraryPayload(payload) == "" {
		return []byte("{}")
	}
	return payload
}
