package orchestrator

import "testing"

func TestMarshalManageVolumePayloadNormalizesFields(t *testing.T) {
	automount := true
	raw, err := MarshalManageVolumePayload(ManageVolumePayload{
		VolumeName: "  data  ",
		Location:   "  fsn1  ",
		State:      " present ",
		SizeGB:     20,
		Automount:  &automount,
	})
	if err != nil {
		t.Fatalf("MarshalManageVolumePayload() error = %v", err)
	}

	decoded, err := UnmarshalManageVolumePayload(raw)
	if err != nil {
		t.Fatalf("UnmarshalManageVolumePayload() error = %v", err)
	}

	if decoded.VolumeName != "data" {
		t.Fatalf("volume_name = %q, want %q", decoded.VolumeName, "data")
	}
	if decoded.Location != "fsn1" {
		t.Fatalf("location = %q, want %q", decoded.Location, "fsn1")
	}
	if decoded.State != "present" {
		t.Fatalf("state = %q, want %q", decoded.State, "present")
	}
}

func TestMarshalUpdateFirewallsPayloadDropsEmptyEntries(t *testing.T) {
	raw, err := MarshalUpdateFirewallsPayload(UpdateFirewallsPayload{
		Firewalls: []string{" web ", "", "db"},
	})
	if err != nil {
		t.Fatalf("MarshalUpdateFirewallsPayload() error = %v", err)
	}

	decoded, err := UnmarshalUpdateFirewallsPayload(raw)
	if err != nil {
		t.Fatalf("UnmarshalUpdateFirewallsPayload() error = %v", err)
	}

	if len(decoded.Firewalls) != 2 {
		t.Fatalf("len(firewalls) = %d, want 2", len(decoded.Firewalls))
	}
	if decoded.Firewalls[0] != "web" || decoded.Firewalls[1] != "db" {
		t.Fatalf("firewalls = %#v, want [web db]", decoded.Firewalls)
	}
}

func TestUnmarshalResizeServerPayloadRejectsInvalidJSON(t *testing.T) {
	if _, err := UnmarshalResizeServerPayload("{"); err == nil {
		t.Fatal("expected invalid normalized payload error")
	}
}
