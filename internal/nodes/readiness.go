package nodes

import (
	"context"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"
)

const (
	ReasonNodeStatusNotActive = "node_status_not_active"
	ReasonNodeHostMissing     = "node_host_missing"
	ReasonSSHPortInvalid      = "ssh_port_invalid"
	ReasonSSHUserMissing      = "ssh_user_missing"
	ReasonNodeUnreachable     = "node_unreachable"
	ReasonSudoUnavailable     = "sudo_unavailable"
	ReasonRuntimeMissing      = "runtime_missing"
)

type ReadinessProbe interface {
	CheckNodePrerequisites(ctx context.Context, host string, port int, user string, isLocal bool) ([]string, error)
}

type ReadinessReport struct {
	IsReady     bool      `json:"is_ready"`
	ReasonCodes []string  `json:"reason_codes"`
	Guidance    []string  `json:"guidance"`
	CheckedAt   time.Time `json:"checked_at"`
}

type ReadinessChecker struct {
	probe ReadinessProbe
	now   func() time.Time
}

func NewReadinessChecker(probe ReadinessProbe) *ReadinessChecker {
	return &ReadinessChecker{
		probe: probe,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (c *ReadinessChecker) Evaluate(ctx context.Context, node Node) (ReadinessReport, error) {
	reasons := make([]string, 0, 6)

	if node.Status != StatusActive {
		reasons = append(reasons, ReasonNodeStatusNotActive)
	}
	if strings.TrimSpace(node.Hostname) == "" {
		reasons = append(reasons, ReasonNodeHostMissing)
	}
	if node.SSHPort <= 0 || node.SSHPort > 65535 {
		reasons = append(reasons, ReasonSSHPortInvalid)
	}
	if strings.TrimSpace(node.SSHUser) == "" {
		reasons = append(reasons, ReasonSSHUserMissing)
	}

	if len(reasons) == 0 && c.probe != nil && os.Getenv("PRESSLUFT_DISABLE_RUNTIME_PROBES") != "1" {
		probeReasons, err := c.probe.CheckNodePrerequisites(ctx, node.Hostname, node.SSHPort, node.SSHUser, node.IsLocal)
		if err != nil {
			return ReadinessReport{}, fmt.Errorf("probe node prerequisites: %w", err)
		}
		reasons = append(reasons, probeReasons...)
	}

	ordered := normalizeReasonCodes(reasons)
	guidance := make([]string, 0, len(ordered))
	for _, code := range ordered {
		if message, ok := reasonGuidance(code); ok {
			guidance = append(guidance, message)
		}
	}

	return ReadinessReport{
		IsReady:     len(ordered) == 0,
		ReasonCodes: ordered,
		Guidance:    guidance,
		CheckedAt:   c.now(),
	}, nil
}

func normalizeReasonCodes(codes []string) []string {
	if len(codes) == 0 {
		return nil
	}

	priority := map[string]int{
		ReasonNodeStatusNotActive: 0,
		ReasonNodeHostMissing:     1,
		ReasonSSHPortInvalid:      2,
		ReasonSSHUserMissing:      3,
		ReasonNodeUnreachable:     4,
		ReasonSudoUnavailable:     5,
		ReasonRuntimeMissing:      6,
	}

	unique := make(map[string]struct{}, len(codes))
	filtered := make([]string, 0, len(codes))
	for _, code := range codes {
		normalized := strings.TrimSpace(code)
		if normalized == "" {
			continue
		}
		if _, ok := priority[normalized]; !ok {
			continue
		}
		if _, exists := unique[normalized]; exists {
			continue
		}
		unique[normalized] = struct{}{}
		filtered = append(filtered, normalized)
	}

	slices.SortFunc(filtered, func(a, b string) int {
		return priority[a] - priority[b]
	})

	return filtered
}

func reasonGuidance(code string) (string, bool) {
	switch code {
	case ReasonNodeStatusNotActive:
		return "Node status must be active before create flows can run.", true
	case ReasonNodeHostMissing:
		return "Set node hostname or public_ip so Pressluft can target the runtime host.", true
	case ReasonSSHPortInvalid:
		return "Set a valid SSH port (1-65535) on the node record.", true
	case ReasonSSHUserMissing:
		return "Set the SSH user for the node so runtime checks can authenticate.", true
	case ReasonNodeUnreachable:
		return "Verify SSH reachability and credentials from Pressluft to the node.", true
	case ReasonSudoUnavailable:
		return "Configure passwordless sudo for the node SSH user (sudo -n true must succeed).", true
	case ReasonRuntimeMissing:
		return "Install required runtime tools on the node (wp CLI must be available).", true
	default:
		return "", false
	}
}
