package orchestrator

import "testing"

func TestSupportedJobKindsDeclareDispatchPolicy(t *testing.T) {
	for _, spec := range SupportedJobKinds() {
		if spec.ExecutionPath == "" {
			t.Fatalf("job kind %q missing execution path", spec.Kind)
		}
		if _, ok := DispatchPolicyForKind(string(spec.Kind)); !ok {
			t.Fatalf("job kind %q missing dispatch policy", spec.Kind)
		}
	}
}

func TestDispatchPolicyForKindMatchesSupportedKindContract(t *testing.T) {
	for _, spec := range SupportedJobKinds() {
		policy, ok := DispatchPolicyForKind(string(spec.Kind))
		if !ok {
			t.Fatalf("job kind %q missing dispatch policy lookup", spec.Kind)
		}
		if policy != spec.DispatchPolicy {
			t.Fatalf("dispatch policy = %#v, want %#v", policy, spec.DispatchPolicy)
		}
	}
}
