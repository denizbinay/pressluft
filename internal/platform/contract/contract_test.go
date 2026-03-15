package contract

import (
	"strings"
	"testing"
)

func TestRenderTypeScriptModuleExportsContractSurface(t *testing.T) {
	rendered, err := RenderTypeScriptModule()
	if err != nil {
		t.Fatalf("RenderTypeScriptModule() error = %v", err)
	}
	for _, needle := range []string{
		"export const platformContract = ",
		"export type ServerStatus = ",
		"export const jobKindLabels: Record<JobKind, string>",
		"export const jobKindSteps: Record<JobKind, readonly WorkflowStep[]>",
		"export const jobTerminalStatuses: readonly JobTerminalStatus[] = ",
		"dispatch_policy",
	} {
		if !strings.Contains(rendered, needle) {
			t.Fatalf("RenderTypeScriptModule() missing %q", needle)
		}
	}
}

func TestJobKindsRemainSorted(t *testing.T) {
	spec := SpecData()
	for i := 1; i < len(spec.JobKinds); i++ {
		if spec.JobKinds[i-1].Kind > spec.JobKinds[i].Kind {
			t.Fatalf("job kinds are not sorted: %q appears before %q", spec.JobKinds[i-1].Kind, spec.JobKinds[i].Kind)
		}
	}
}

func TestJobKindsIncludeWorkflowSteps(t *testing.T) {
	spec := SpecData()
	for _, jobKind := range spec.JobKinds {
		if len(jobKind.Steps) == 0 {
			t.Fatalf("job kind %q is missing workflow steps", jobKind.Kind)
		}
		if jobKind.Steps[0].Key != "validate" {
			t.Fatalf("job kind %q should begin with validate step, got %#v", jobKind.Kind, jobKind.Steps)
		}
	}
}

func TestJobKindsExposeDispatchPolicy(t *testing.T) {
	spec := SpecData()
	for _, jobKind := range spec.JobKinds {
		if jobKind.ExecutionPath == "" {
			t.Fatalf("job kind %q missing execution path", jobKind.Kind)
		}
		if jobKind.Destructive && !jobKind.DispatchPolicy.QueueServer && jobKind.Kind != "configure_server" && jobKind.Kind != "provision_server" {
			t.Fatalf("job kind %q should declare queue-backed dispatch policy", jobKind.Kind)
		}
	}
}
