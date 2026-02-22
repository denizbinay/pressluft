package nodes

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProbe struct {
	reasons []string
	err     error
}

func (f fakeProbe) CheckNodePrerequisites(context.Context, string, int, string, bool) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.reasons, nil
}

func TestReadinessCheckerEvaluateStaticFailures(t *testing.T) {
	checker := NewReadinessChecker(fakeProbe{})
	checker.now = func() time.Time { return time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC) }

	report, err := checker.Evaluate(context.Background(), Node{Status: StatusProvisioning})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if report.IsReady {
		t.Fatal("expected report to be not ready")
	}
	if len(report.ReasonCodes) < 4 {
		t.Fatalf("reason_codes len = %d, want at least 4", len(report.ReasonCodes))
	}
	if report.ReasonCodes[0] != ReasonNodeStatusNotActive {
		t.Fatalf("first reason = %q, want %q", report.ReasonCodes[0], ReasonNodeStatusNotActive)
	}
	if report.CheckedAt.IsZero() {
		t.Fatal("checked_at is zero")
	}
}

func TestReadinessCheckerEvaluateProbeFailures(t *testing.T) {
	checker := NewReadinessChecker(fakeProbe{reasons: []string{ReasonRuntimeMissing, ReasonSudoUnavailable, ReasonRuntimeMissing}})

	report, err := checker.Evaluate(context.Background(), Node{
		Status:   StatusActive,
		Hostname: "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "pressluft",
		IsLocal:  true,
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if report.IsReady {
		t.Fatal("expected report to be not ready")
	}
	if len(report.ReasonCodes) != 2 {
		t.Fatalf("reason_codes len = %d, want 2", len(report.ReasonCodes))
	}
	if report.ReasonCodes[0] != ReasonSudoUnavailable {
		t.Fatalf("reason[0] = %q, want %q", report.ReasonCodes[0], ReasonSudoUnavailable)
	}
	if report.ReasonCodes[1] != ReasonRuntimeMissing {
		t.Fatalf("reason[1] = %q, want %q", report.ReasonCodes[1], ReasonRuntimeMissing)
	}
	if len(report.Guidance) != 2 {
		t.Fatalf("guidance len = %d, want 2", len(report.Guidance))
	}
}

func TestReadinessCheckerEvaluateProbeError(t *testing.T) {
	checker := NewReadinessChecker(fakeProbe{err: errors.New("probe exploded")})

	_, err := checker.Evaluate(context.Background(), Node{
		Status:   StatusActive,
		Hostname: "127.0.0.1",
		SSHPort:  22,
		SSHUser:  "pressluft",
	})
	if err == nil {
		t.Fatal("expected probe error")
	}
}
