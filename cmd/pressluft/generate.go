package main

import (
	"fmt"
	"os"
	"path/filepath"

	"pressluft/internal/cli/cliutil"
	"pressluft/internal/contract"
)

// runGenerate regenerates TypeScript contracts from Go types.
// Called internally by dev and build — not a user-facing command.
func runGenerate(args []string) error {

	rootDir, err := cliutil.FindRepoRoot()
	if err != nil {
		return err
	}

	// Platform contract.
	platformTS, err := contract.RenderTypeScriptModule()
	if err != nil {
		return fmt.Errorf("render platform contract: %w", err)
	}
	platformPath := filepath.Join(rootDir, "web", "app", "lib", "platform-contract.generated.ts")
	if err := os.WriteFile(platformPath, []byte(platformTS), 0o644); err != nil {
		return fmt.Errorf("write platform contract: %w", err)
	}

	// API contract.
	apiTS, err := contract.RenderAPITypeScriptModule()
	if err != nil {
		return fmt.Errorf("render api contract: %w", err)
	}
	apiPath := filepath.Join(rootDir, "web", "app", "lib", "api-contract.ts")
	if err := os.WriteFile(apiPath, []byte(apiTS), 0o644); err != nil {
		return fmt.Errorf("write api contract: %w", err)
	}

	return nil
}
