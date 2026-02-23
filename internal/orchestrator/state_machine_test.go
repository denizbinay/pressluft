package orchestrator

import "testing"

func TestCanTransition(t *testing.T) {
	if !CanTransition(JobStatusQueued, JobStatusPreparing) {
		t.Fatal("expected queued -> preparing to be valid")
	}
	if CanTransition(JobStatusSucceeded, JobStatusRunning) {
		t.Fatal("expected succeeded -> running to be invalid")
	}
}

func TestValidateTransition(t *testing.T) {
	if err := ValidateTransition(JobStatusRunning, JobStatusVerifying); err != nil {
		t.Fatalf("expected valid transition, got %v", err)
	}
	if err := ValidateTransition(JobStatusRunning, JobStatusQueued); err == nil {
		t.Fatal("expected invalid transition error")
	}
}

func TestIsTerminalStatus(t *testing.T) {
	if !IsTerminalStatus(JobStatusSucceeded) {
		t.Fatal("expected succeeded to be terminal")
	}
	if IsTerminalStatus(JobStatusRunning) {
		t.Fatal("expected running to be non-terminal")
	}
}
