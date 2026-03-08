package contract

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGeneratedTypeScriptContractIsFresh(t *testing.T) {
	rendered, err := RenderTypeScriptModule()
	if err != nil {
		t.Fatalf("RenderTypeScriptModule() error = %v", err)
	}

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve caller path")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(filename), "../.."))
	contractPath := filepath.Join(repoRoot, "web", "app", "lib", "platform-contract.generated.ts")
	current, err := os.ReadFile(contractPath)
	if err != nil {
		t.Fatalf("read generated contract: %v", err)
	}

	if string(current) != rendered {
		t.Fatalf("generated contract is stale; run `make generate-contract` or `go run ./cmd/pressluft-contractgen -format ts > web/app/lib/platform-contract.generated.ts`")
	}
}
