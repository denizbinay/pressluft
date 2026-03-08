package platform

import (
	"fmt"
	"strings"
)

type ExecutionMode string

const (
	ExecutionModeDev                 ExecutionMode = "dev"
	ExecutionModeSingleNodeLocal     ExecutionMode = "single-node-local-control-plane"
	ExecutionModeProductionBootstrap ExecutionMode = "production-bootstrap"
	defaultControlPlaneExecutionMode ExecutionMode = ExecutionModeSingleNodeLocal
	defaultNonDevAgentExecutionMode  ExecutionMode = ExecutionModeProductionBootstrap
)

type SupportLevel string

const (
	SupportLevelSupported    SupportLevel = "supported"
	SupportLevelExperimental SupportLevel = "experimental"
	SupportLevelUnavailable  SupportLevel = "unavailable"
)

type ServerStatus string

type SetupState string

type NodeStatus string

const (
	ServerStatusPending      ServerStatus = "pending"
	ServerStatusProvisioning ServerStatus = "provisioning"
	ServerStatusConfiguring  ServerStatus = "configuring"
	ServerStatusRebuilding   ServerStatus = "rebuilding"
	ServerStatusResizing     ServerStatus = "resizing"
	ServerStatusDeleting     ServerStatus = "deleting"
	ServerStatusDeleted      ServerStatus = "deleted"
	ServerStatusReady        ServerStatus = "ready"
	ServerStatusFailed       ServerStatus = "failed"
)

const (
	SetupStateNotStarted SetupState = "not_started"
	SetupStateRunning    SetupState = "running"
	SetupStateDegraded   SetupState = "degraded"
	SetupStateReady      SetupState = "ready"
)

const (
	NodeStatusOnline    NodeStatus = "online"
	NodeStatusUnhealthy NodeStatus = "unhealthy"
	NodeStatusOffline   NodeStatus = "offline"
	NodeStatusUnknown   NodeStatus = "unknown"
)

const (
	NodeUnhealthyThresholdSeconds = 45
	NodeOfflineThresholdSeconds   = 150
)

type CallbackURLMode string

const (
	CallbackURLModeUnknown   CallbackURLMode = "unknown"
	CallbackURLModeStable    CallbackURLMode = "stable"
	CallbackURLModeEphemeral CallbackURLMode = "ephemeral"
)

func AllExecutionModes() []ExecutionMode {
	return []ExecutionMode{
		ExecutionModeDev,
		ExecutionModeSingleNodeLocal,
		ExecutionModeProductionBootstrap,
	}
}

func AllSupportLevels() []SupportLevel {
	return []SupportLevel{
		SupportLevelSupported,
		SupportLevelExperimental,
		SupportLevelUnavailable,
	}
}

func AllServerStatuses() []ServerStatus {
	return []ServerStatus{
		ServerStatusPending,
		ServerStatusProvisioning,
		ServerStatusConfiguring,
		ServerStatusRebuilding,
		ServerStatusResizing,
		ServerStatusDeleting,
		ServerStatusDeleted,
		ServerStatusReady,
		ServerStatusFailed,
	}
}

func NormalizeServerStatus(raw string) (ServerStatus, error) {
	status := ServerStatus(strings.TrimSpace(raw))
	switch status {
	case ServerStatusPending,
		ServerStatusProvisioning,
		ServerStatusConfiguring,
		ServerStatusRebuilding,
		ServerStatusResizing,
		ServerStatusDeleting,
		ServerStatusDeleted,
		ServerStatusReady,
		ServerStatusFailed:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported server status %q", raw)
	}
}

func InProgressServerStatuses() []ServerStatus {
	return []ServerStatus{
		ServerStatusPending,
		ServerStatusProvisioning,
		ServerStatusConfiguring,
		ServerStatusRebuilding,
		ServerStatusResizing,
		ServerStatusDeleting,
	}
}

func MutationBlockedServerStatuses() []ServerStatus {
	return []ServerStatus{
		ServerStatusDeleting,
		ServerStatusDeleted,
	}
}

func DestructiveActionServerStatuses() []ServerStatus {
	return []ServerStatus{
		ServerStatusRebuilding,
		ServerStatusResizing,
		ServerStatusDeleting,
	}
}

func AllSetupStates() []SetupState {
	return []SetupState{
		SetupStateNotStarted,
		SetupStateRunning,
		SetupStateDegraded,
		SetupStateReady,
	}
}

func NormalizeSetupState(raw string) (SetupState, error) {
	state := SetupState(strings.TrimSpace(raw))
	switch state {
	case SetupStateNotStarted, SetupStateRunning, SetupStateDegraded, SetupStateReady:
		return state, nil
	default:
		return "", fmt.Errorf("unsupported setup state %q", raw)
	}
}

func AllNodeStatuses() []NodeStatus {
	return []NodeStatus{
		NodeStatusOnline,
		NodeStatusUnhealthy,
		NodeStatusOffline,
		NodeStatusUnknown,
	}
}

func NormalizeNodeStatus(raw string) (NodeStatus, error) {
	status := NodeStatus(strings.TrimSpace(raw))
	switch status {
	case NodeStatusOnline, NodeStatusUnhealthy, NodeStatusOffline, NodeStatusUnknown:
		return status, nil
	default:
		return "", fmt.Errorf("unsupported node status %q", raw)
	}
}

func ReachableNodeStatuses() []NodeStatus {
	return []NodeStatus{
		NodeStatusOnline,
		NodeStatusUnhealthy,
	}
}

func AllCallbackURLModes() []CallbackURLMode {
	return []CallbackURLMode{
		CallbackURLModeUnknown,
		CallbackURLModeStable,
		CallbackURLModeEphemeral,
	}
}

func QueuedServerStatusForJobKind(kind string) (ServerStatus, bool) {
	switch strings.TrimSpace(kind) {
	case "delete_server":
		return ServerStatusDeleting, true
	case "rebuild_server":
		return ServerStatusRebuilding, true
	case "resize_server":
		return ServerStatusResizing, true
	default:
		return "", false
	}
}

func IsDeletingOrDeletedServerStatus(status string) bool {
	switch ServerStatus(strings.TrimSpace(status)) {
	case ServerStatusDeleting, ServerStatusDeleted:
		return true
	default:
		return false
	}
}

func DetectCallbackURLMode(raw string) CallbackURLMode {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return CallbackURLModeUnknown
	}
	if strings.Contains(value, ".trycloudflare.com") {
		return CallbackURLModeEphemeral
	}
	return CallbackURLModeStable
}

func NormalizeControlPlaneExecutionMode(raw string, devBuild bool) (ExecutionMode, error) {
	if devBuild {
		if strings.TrimSpace(raw) != "" && ExecutionMode(strings.TrimSpace(raw)) != ExecutionModeDev {
			return "", fmt.Errorf("dev builds only support execution mode %q", ExecutionModeDev)
		}
		return ExecutionModeDev, nil
	}

	mode := ExecutionMode(strings.TrimSpace(raw))
	if mode == "" {
		mode = defaultControlPlaneExecutionMode
	}

	switch mode {
	case ExecutionModeSingleNodeLocal, ExecutionModeProductionBootstrap:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported control-plane execution mode %q", mode)
	}
}

func NormalizeAgentExecutionMode(raw string, devBuild bool) (ExecutionMode, error) {
	if devBuild {
		if strings.TrimSpace(raw) != "" && ExecutionMode(strings.TrimSpace(raw)) != ExecutionModeDev {
			return "", fmt.Errorf("dev agent builds only support execution mode %q", ExecutionModeDev)
		}
		return ExecutionModeDev, nil
	}

	mode := ExecutionMode(strings.TrimSpace(raw))
	if mode == "" {
		mode = defaultNonDevAgentExecutionMode
	}

	switch mode {
	case ExecutionModeProductionBootstrap:
		return mode, nil
	default:
		return "", fmt.Errorf("unsupported agent execution mode %q", mode)
	}
}
