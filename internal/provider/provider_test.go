package provider_test

import (
	"testing"

	"pressluft/internal/provider"
	_ "pressluft/internal/provider/hetzner"
)

func TestWorkflowCapabilities(t *testing.T) {
	if !provider.SupportsProvisioningWorkflow("hetzner") {
		t.Fatal("expected hetzner to support provisioning workflow")
	}
	if !provider.SupportsServerMutationWorkflow("hetzner") {
		t.Fatal("expected hetzner to support server mutation workflows")
	}
	if provider.SupportsProvisioningWorkflow("unknown") {
		t.Fatal("expected unknown provider type to reject provisioning workflow")
	}
}
