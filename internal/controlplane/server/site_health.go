package server

import (
	"context"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/controlplane/server/health"
)

// VerifyPublicSiteRouting verifies that a hostname routes to the expected site over HTTPS.
func VerifyPublicSiteRouting(ctx context.Context, siteID, hostname string) error {
	return health.VerifyPublicSiteRouting(ctx, siteID, hostname)
}

// VerifyPublicWordPressRuntime verifies that WordPress is responding correctly at a hostname.
func VerifyPublicWordPressRuntime(ctx context.Context, hostname string) error {
	return health.VerifyPublicWordPressRuntime(ctx, hostname)
}

// RuntimeHealthFromAgentSnapshot extracts runtime health state from an agent health snapshot.
func RuntimeHealthFromAgentSnapshot(snapshot *agentcommand.SiteHealthSnapshot) (string, string) {
	return health.RuntimeHealthFromAgentSnapshot(snapshot)
}
