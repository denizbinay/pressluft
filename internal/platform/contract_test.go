package platform

import "testing"

func TestNormalizeControlPlaneExecutionMode(t *testing.T) {
	t.Run("dev build forces dev mode", func(t *testing.T) {
		mode, err := NormalizeControlPlaneExecutionMode("", true)
		if err != nil {
			t.Fatalf("NormalizeControlPlaneExecutionMode() error = %v", err)
		}
		if mode != ExecutionModeDev {
			t.Fatalf("mode = %q, want %q", mode, ExecutionModeDev)
		}
	})

	t.Run("non-dev defaults to single-node local", func(t *testing.T) {
		mode, err := NormalizeControlPlaneExecutionMode("", false)
		if err != nil {
			t.Fatalf("NormalizeControlPlaneExecutionMode() error = %v", err)
		}
		if mode != ExecutionModeSingleNodeLocal {
			t.Fatalf("mode = %q, want %q", mode, ExecutionModeSingleNodeLocal)
		}
	})

	t.Run("rejects unsupported mode", func(t *testing.T) {
		if _, err := NormalizeControlPlaneExecutionMode("dev", false); err == nil {
			t.Fatal("expected error for unsupported non-dev control-plane mode")
		}
	})
}

func TestNormalizeAgentExecutionMode(t *testing.T) {
	t.Run("dev build forces dev mode", func(t *testing.T) {
		mode, err := NormalizeAgentExecutionMode("", true)
		if err != nil {
			t.Fatalf("NormalizeAgentExecutionMode() error = %v", err)
		}
		if mode != ExecutionModeDev {
			t.Fatalf("mode = %q, want %q", mode, ExecutionModeDev)
		}
	})

	t.Run("non-dev defaults to production bootstrap", func(t *testing.T) {
		mode, err := NormalizeAgentExecutionMode("", false)
		if err != nil {
			t.Fatalf("NormalizeAgentExecutionMode() error = %v", err)
		}
		if mode != ExecutionModeProductionBootstrap {
			t.Fatalf("mode = %q, want %q", mode, ExecutionModeProductionBootstrap)
		}
	})

	t.Run("rejects unsupported non-dev mode", func(t *testing.T) {
		if _, err := NormalizeAgentExecutionMode("single-node-local-control-plane", false); err == nil {
			t.Fatal("expected error for unsupported non-dev agent mode")
		}
	})
}
