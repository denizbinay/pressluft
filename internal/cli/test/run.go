package test

import (
	"fmt"
	"os"
	"os/exec"

	"pressluft/internal/cli/cliutil"
)

func Run() error {
	rootDir, err := cliutil.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	cmd := exec.Command(cliutil.GoCmd(), "test", "./...")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go test failed: %w", err)
	}
	return nil
}
