package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var unitTestPackages = []string{
	"./cmd/pressluft-server",
	"./cmd/pressluft-agent",
	"./internal/agent",
	"./internal/agentauth",
	"./internal/agentcommand",
	"./internal/auth",
	"./internal/contract",
	"./internal/database",
	"./internal/dispatch",
	"./internal/envconfig",
	"./internal/platform",
	"./internal/registration",
	"./internal/security",
	"./internal/worker",
	"./internal/ws",
}

var integrationTestPackages = []string{
	"./internal/activity",
	"./internal/orchestrator",
	"./internal/provider/...",
	"./internal/runner/ansible",
	"./internal/server",
	"./internal/server/profiles",
}

var testCmd = &cobra.Command{
	Use:   "test [unit|integration]",
	Short: "Run test suites (default: all)",
	Long: `Run test suites.

  pressluft test              Run all tests (unit + integration)
  pressluft test unit         Run unit tests only
  pressluft test integration  Run integration tests only`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"unit", "integration"},
	RunE:      runTest,
}

func runTest(cmd *cobra.Command, args []string) error {
	suite := ""
	if len(args) > 0 {
		suite = args[0]
	}

	rootDir, err := findRepoRoot()
	if err != nil {
		return err
	}

	switch suite {
	case "unit":
		return goTest(rootDir, unitTestPackages, false)
	case "integration":
		return goTest(rootDir, integrationTestPackages, true)
	default:
		if err := goTest(rootDir, unitTestPackages, false); err != nil {
			return err
		}
		return goTest(rootDir, integrationTestPackages, true)
	}
}

func goTest(rootDir string, packages []string, noCache bool) error {
	testArgs := []string{"test"}
	if noCache {
		testArgs = append(testArgs, "-count=1")
	}
	testArgs = append(testArgs, packages...)

	cmd := exec.Command(goCmd(), testArgs...)
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	return nil
}
