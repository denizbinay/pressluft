package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"pressluft/internal/cli/cliui"
	"pressluft/internal/cli/cliutil"
)

var buildCmd = &cobra.Command{
	Use:   "build [server|agent]",
	Short: "Build binaries (default: full pipeline)",
	Long: `Build project binaries.

Without arguments, runs the full pipeline: generate contracts, build frontend,
embed assets, and compile both server and agent binaries.

  pressluft build              Full pipeline
  pressluft build server       Build only the control-plane server binary
  pressluft build agent        Build the production agent binary
  pressluft build agent --dev  Build the dev agent binary`,
	Args:      cobra.MaximumNArgs(1),
	ValidArgs: []string{"server", "agent"},
	RunE:      runBuild,
}

var buildDev bool

func init() {
	buildCmd.Flags().BoolVar(&buildDev, "dev", false, "Build with dev tags (agent only)")
}

func runBuild(cmd *cobra.Command, args []string) error {
	rootDir, err := cliutil.FindRepoRoot()
	if err != nil {
		return err
	}

	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	switch target {
	case "":
		return buildFull(rootDir)
	case "server":
		return buildServer(rootDir)
	case "agent":
		return buildAgent(rootDir, buildDev)
	default:
		return fmt.Errorf("unknown build target %q (use server or agent)", target)
	}
}

func buildFull(rootDir string) error {
	cliui.Step("Generating contracts")
	if err := runGenerate(nil); err != nil {
		return fmt.Errorf("generate: %w", err)
	}
	cliui.StepDone("Generating contracts")

	cliui.Step("Building frontend")
	if err := buildFrontend(rootDir); err != nil {
		return fmt.Errorf("frontend: %w", err)
	}
	cliui.StepDone("Building frontend")

	cliui.Step("Embedding frontend assets")
	if err := embedFrontend(rootDir); err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	cliui.StepDone("Embedding frontend assets")

	cliui.Step("Building server")
	if err := buildServer(rootDir); err != nil {
		return fmt.Errorf("server: %w", err)
	}
	cliui.StepDone("Building server")

	cliui.Step("Building agent")
	if err := buildAgent(rootDir, false); err != nil {
		return fmt.Errorf("agent: %w", err)
	}
	cliui.StepDone("Building agent")

	lipgloss.Println()
	lipgloss.Println(cliui.Success.Render("Build complete."))
	return nil
}

func buildServer(rootDir string) error {
	binPath := filepath.Join(rootDir, "bin", "pressluft-server")
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command(cliutil.GoCmd(), "build", "-o", binPath, "./cmd/pressluft-server")
	cmd.Dir = rootDir
	cmd.Env = appendBuildEnv(os.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildAgent(rootDir string, dev bool) error {
	binPath := filepath.Join(rootDir, "bin", "pressluft-agent")
	if err := os.MkdirAll(filepath.Dir(binPath), 0o755); err != nil {
		return err
	}
	buildArgs := []string{"build", "-o", binPath}
	if dev {
		buildArgs = append(buildArgs, "-tags", "dev")
	}
	buildArgs = append(buildArgs, "./cmd/pressluft-agent")
	cmd := exec.Command(cliutil.GoCmd(), buildArgs...)
	cmd.Dir = rootDir
	cmd.Env = appendBuildEnv(os.Environ())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func buildFrontend(rootDir string) error {
	webDir := filepath.Join(rootDir, "web")

	// Install deps if needed.
	if _, err := os.Stat(filepath.Join(webDir, "node_modules")); os.IsNotExist(err) {
		install := exec.Command(cliutil.NpmCmd(), "--prefix", webDir, "install")
		install.Dir = rootDir
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("install frontend deps: %w", err)
		}
	}

	gen := exec.Command(cliutil.NpmCmd(), "--prefix", webDir, "run", "generate")
	gen.Dir = rootDir
	gen.Env = append(os.Environ(), "NODE_OPTIONS=--max-old-space-size=8192")
	gen.Stdout = os.Stdout
	gen.Stderr = os.Stderr
	if err := gen.Run(); err != nil {
		return fmt.Errorf("generate frontend: %w", err)
	}

	// Verify output exists.
	indexPath := filepath.Join(webDir, ".output", "public", "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return fmt.Errorf("frontend build did not produce %s", indexPath)
	}
	return nil
}

func embedFrontend(rootDir string) error {
	embedDir := filepath.Join(rootDir, "internal", "controlplane", "server", "dist")
	if err := os.RemoveAll(embedDir); err != nil {
		return err
	}
	if err := os.MkdirAll(embedDir, 0o755); err != nil {
		return err
	}

	// Create .gitkeep.
	if err := os.WriteFile(filepath.Join(embedDir, ".gitkeep"), nil, 0o644); err != nil {
		return err
	}

	// Copy built frontend assets.
	srcDir := filepath.Join(rootDir, "web", ".output", "public")
	cmd := exec.Command("cp", "-R", srcDir+"/.", embedDir+"/")
	cmd.Dir = rootDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
