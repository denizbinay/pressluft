package auth

import "testing"

func TestRoleCapabilitiesAdminIncludesAllCapabilities(t *testing.T) {
	granted := RoleCapabilities(RoleAdmin)
	all := AllCapabilities()
	if len(granted) != len(all) {
		t.Fatalf("len(RoleCapabilities(admin)) = %d, want %d", len(granted), len(all))
	}
	for _, capability := range all {
		if !HasCapability(Actor{
			ID:            "1",
			Role:          RoleAdmin,
			Authenticated: true,
			Capabilities:  granted,
		}, capability) {
			t.Fatalf("expected admin to have capability %q", capability)
		}
	}
}

func TestRequireCapabilityRejectsAnonymousActor(t *testing.T) {
	allow := RequireCapability(CapabilityManageServers)
	if allow(AnonymousActor()) {
		t.Fatal("expected anonymous actor to be rejected")
	}
}

func TestAllCapabilitiesIncludesManageSites(t *testing.T) {
	for _, capability := range AllCapabilities() {
		if capability == CapabilityManageSites {
			return
		}
	}
	t.Fatal("expected manage_sites capability to be published")
}
