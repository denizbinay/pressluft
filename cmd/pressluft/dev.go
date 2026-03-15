package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"

	"pressluft/internal/cli/cliui"
	"pressluft/internal/cli/cliutil"
	"pressluft/internal/cli/devdiag"
	"pressluft/internal/shared/envconfig"
)

// DevConfig defines the environment variables that control the dev workflow.
// This struct is the contract: if a field exists here, it must be wired.
type DevConfig struct {
	ControlPlaneURL string // PRESSLUFT_CONTROL_PLANE_URL
	APIPort         int    // DEV_API_PORT (default 8081)
	UIPort          int    // DEV_UI_PORT (default 8080)
	UIHost          string // DEV_UI_HOST (default 0.0.0.0)
	WebDir          string // resolved to <root>/web
	GoCmd           string // GO or "go"
	NpmCmd          string // NPM or "pnpm"
}

func resolveDevConfig(rootDir string) DevConfig {
	apiPort := 8081
	if raw := os.Getenv("DEV_API_PORT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			apiPort = v
		}
	}
	uiPort := 8080
	if raw := os.Getenv("DEV_UI_PORT"); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil && v > 0 {
			uiPort = v
		}
	}
	uiHost := "0.0.0.0"
	if raw := os.Getenv("DEV_UI_HOST"); raw != "" {
		uiHost = raw
	}
	goCmd := "go"
	if raw := os.Getenv("GO"); raw != "" {
		goCmd = raw
	}
	npmCmd := "pnpm"
	if raw := os.Getenv("NPM"); raw != "" {
		npmCmd = raw
	}

	return DevConfig{
		ControlPlaneURL: strings.TrimSpace(os.Getenv("PRESSLUFT_CONTROL_PLANE_URL")),
		APIPort:         apiPort,
		UIPort:          uiPort,
		UIHost:          uiHost,
		WebDir:          filepath.Join(rootDir, "web"),
		GoCmd:           goCmd,
		NpmCmd:          npmCmd,
	}
}

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start the local dev environment",
	Long: `Start the local dev environment with backend, frontend, and tunnel.

Environment variables:
  PRESSLUFT_CONTROL_PLANE_URL  Stable public URL; unset = ephemeral Cloudflare tunnel
  DEV_API_PORT                 Backend port (default: 8081)
  DEV_UI_PORT                  Nuxt port (default: 8080)
  DEV_UI_HOST                  Nuxt host (default: 0.0.0.0)

All variables can be set in a .env file at the repository root.`,
	RunE: runDev,
}

func runDev(cmd *cobra.Command, args []string) error {
	rootDir, err := cliutil.FindRepoRoot()
	if err != nil {
		return fmt.Errorf("find repo root: %w", err)
	}

	// Load .env before resolving config so file values are available.
	if err := loadDotenv(filepath.Join(rootDir, ".env")); err != nil {
		return fmt.Errorf("load .env: %w", err)
	}

	cfg := resolveDevConfig(rootDir)

	// Preflight checks.
	runtime, err := envconfig.ResolveControlPlaneRuntime(true, rootDir)
	if err != nil {
		return fmt.Errorf("resolve runtime: %w", err)
	}
	// Set PRESSLUFT_CONTROL_PLANE_URL so preflight sees it.
	if cfg.ControlPlaneURL != "" {
		os.Setenv("PRESSLUFT_CONTROL_PLANE_URL", cfg.ControlPlaneURL)
		runtime.ControlPlaneURL = cfg.ControlPlaneURL
	}

	// Start tunnel if no stable URL is set.
	var tunnelCmd *exec.Cmd
	var tunnelLog string
	if cfg.ControlPlaneURL == "" {
		url, cmd, logFile, err := startQuickTunnel(cfg.APIPort)
		if err != nil {
			return err
		}
		tunnelCmd = cmd
		tunnelLog = logFile
		cfg.ControlPlaneURL = url
		os.Setenv("PRESSLUFT_CONTROL_PLANE_URL", url)
		runtime.ControlPlaneURL = url
		printTunnelWarning()
	}

	// Generate contracts before starting.
	cliui.Step("Generating contracts")
	if err := runGenerate(nil); err != nil {
		return fmt.Errorf("generate contracts: %w", err)
	}
	cliui.StepDone("Generating contracts")

	// Run doctor checks.
	report := devdiag.Inspect(runtime)
	printDoctorReport(report)
	if !report.Healthy() {
		lipgloss.Println()
		cliui.Issues(report.Issues())
		lipgloss.Println()
		cliui.Hint("To reset local state: rm -rf .pressluft")
		return fmt.Errorf("doctor checks failed")
	}

	lipgloss.Println(cliui.Dim.Render(fmt.Sprintf("  dev state: %s/.pressluft/", rootDir)))

	// Build agent-dev.
	cliui.Step("Building dev agent")
	agentBuild := exec.Command(cfg.GoCmd, "build", "-tags", "dev", "-o", filepath.Join(rootDir, "bin", "pressluft-agent"), "./cmd/pressluft-agent")
	agentBuild.Dir = rootDir
	agentBuild.Env = appendBuildEnv(os.Environ())
	agentBuild.Stdout = os.Stdout
	agentBuild.Stderr = os.Stderr
	if err := agentBuild.Run(); err != nil {
		return fmt.Errorf("build dev agent: %w", err)
	}

	cliui.StepDone("Building dev agent")

	// Install frontend deps if needed.
	if _, err := os.Stat(filepath.Join(cfg.WebDir, "node_modules")); os.IsNotExist(err) {
		cliui.Step("Installing frontend dependencies")
		npmInstall := exec.Command(cfg.NpmCmd, "--prefix", cfg.WebDir, "install")
		npmInstall.Dir = rootDir
		npmInstall.Stdout = os.Stdout
		npmInstall.Stderr = os.Stderr
		if err := npmInstall.Run(); err != nil {
			return fmt.Errorf("install frontend deps: %w", err)
		}
		cliui.StepDone("Installing frontend dependencies")
	}

	// Signal handling — clean up child processes.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start Go backend.
	backendLogFile, err := os.CreateTemp("", "pressluft-backend-*.log")
	if err != nil {
		return fmt.Errorf("create backend log: %w", err)
	}
	defer os.Remove(backendLogFile.Name())

	backend := exec.Command(cfg.GoCmd, "run", "-tags", "dev", "./cmd/pressluft-server")
	backend.Dir = rootDir
	backend.Env = append(os.Environ(),
		fmt.Sprintf("PORT=%d", cfg.APIPort),
		fmt.Sprintf("PRESSLUFT_CONTROL_PLANE_URL=%s", cfg.ControlPlaneURL),
	)
	backend.Stdout = backendLogFile
	backend.Stderr = backendLogFile
	if err := backend.Start(); err != nil {
		return fmt.Errorf("start backend: %w", err)
	}

	// Wait for backend to be ready.
	var backendReady bool
	for i := 0; i < 20; i++ {
		if backend.ProcessState != nil {
			break
		}
		content, _ := os.ReadFile(backendLogFile.Name())
		if strings.Contains(string(content), "pressluft listening") {
			backendReady = true
			break
		}
		time.Sleep(time.Second)
	}

	if !backendReady {
		// Check if backend process died.
		if backend.ProcessState != nil {
			content, _ := os.ReadFile(backendLogFile.Name())
			fmt.Fprintln(os.Stderr, string(content))
			return fmt.Errorf("backend exited during startup")
		}
		// Still running but no "listening" message — proceed anyway.
	}

	// Tail backend log.
	go tailFile(backendLogFile.Name())

	// Start Nuxt dev server.
	proxyTarget := os.Getenv("NUXT_DEV_PROXY_TARGET")
	if proxyTarget == "" {
		proxyTarget = fmt.Sprintf("http://localhost:%d/api", cfg.APIPort)
	}
	nuxt := exec.Command(cfg.NpmCmd, "--prefix", cfg.WebDir, "run", "dev")
	nuxt.Dir = rootDir
	nuxt.Env = append(os.Environ(),
		fmt.Sprintf("NUXT_DEV_PROXY_TARGET=%s", proxyTarget),
		fmt.Sprintf("NUXT_HOST=%s", cfg.UIHost),
		fmt.Sprintf("NUXT_PORT=%d", cfg.UIPort),
	)
	nuxt.Stdout = os.Stdout
	nuxt.Stderr = os.Stderr
	if err := nuxt.Start(); err != nil {
		_ = backend.Process.Kill()
		return fmt.Errorf("start nuxt: %w", err)
	}

	// Wait for signal or child exit.
	cleanup := func() {
		if nuxt.Process != nil {
			_ = nuxt.Process.Kill()
		}
		if backend.Process != nil {
			_ = backend.Process.Kill()
		}
		if tunnelCmd != nil && tunnelCmd.Process != nil {
			_ = tunnelCmd.Process.Kill()
		}
		if tunnelLog != "" {
			_ = os.Remove(tunnelLog)
		}
	}

	done := make(chan struct{}, 2)
	backendErrCh := make(chan error, 1)
	nuxtErrCh := make(chan error, 1)
	go func() { backendErrCh <- backend.Wait(); done <- struct{}{} }()
	go func() { nuxtErrCh <- nuxt.Wait(); done <- struct{}{} }()

	select {
	case <-sigCh:
		lipgloss.Println("\n" + cliui.Dim.Render("Shutting down..."))
		cleanup()
	case <-done:
		cleanup()
		// Check if either child process exited with an error.
		select {
		case err := <-backendErrCh:
			if err != nil {
				return fmt.Errorf("backend process exited with error: %w", err)
			}
		default:
		}
		select {
		case err := <-nuxtErrCh:
			if err != nil {
				return fmt.Errorf("nuxt process exited with error: %w", err)
			}
		default:
		}
	}

	return nil
}

func startQuickTunnel(apiPort int) (url string, cmd *exec.Cmd, logFile string, err error) {
	if _, lookErr := exec.LookPath("cloudflared"); lookErr != nil {
		return "", nil, "", fmt.Errorf("cloudflared not found; install it or set PRESSLUFT_CONTROL_PLANE_URL to a stable public URL")
	}

	tmpFile, err := os.CreateTemp("", "pressluft-tunnel-*.log")
	if err != nil {
		return "", nil, "", fmt.Errorf("create tunnel log: %w", err)
	}
	logFile = tmpFile.Name()
	tmpFile.Close()

	cmd = exec.Command("cloudflared", "tunnel",
		"--url", fmt.Sprintf("http://localhost:%d", apiPort),
		"--no-autoupdate",
		"--logfile", logFile,
		"--loglevel", "info",
	)
	cmd.Stdout = nil
	cmd.Stderr = nil
	if err := cmd.Start(); err != nil {
		return "", nil, "", fmt.Errorf("start cloudflared: %w", err)
	}

	tunnelRe := regexp.MustCompile(`https://[a-z0-9-]+\.trycloudflare\.com`)
	for i := 0; i < 30; i++ {
		content, _ := os.ReadFile(logFile)
		if match := tunnelRe.FindString(string(content)); match != "" {
			lipgloss.Println(cliui.Accent.Render("  tunnel: ") + match)
			return match, cmd, logFile, nil
		}
		time.Sleep(time.Second)
	}

	_ = cmd.Process.Kill()
	return "", nil, "", fmt.Errorf("failed to obtain Cloudflare tunnel URL (check %s)", logFile)
}

func printTunnelWarning() {
	lipgloss.Println()
	cliui.WarnBox([]string{
		"Running with a Cloudflare quick tunnel (ephemeral URL).",
		"Remote agents provisioned in this session cannot reconnect",
		"after a restart.",
		"",
		"For durable testing, set PRESSLUFT_CONTROL_PLANE_URL to a",
		"stable URL before running pressluft dev.",
	})
	lipgloss.Println()
}

func tailFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}

	// Continue tailing.
	for {
		for scanner.Scan() {
			fmt.Println(scanner.Text())
		}
		time.Sleep(500 * time.Millisecond)
		// Reset scanner error state for continued reading.
		scanner = bufio.NewScanner(f)
	}
}

func appendBuildEnv(env []string) []string {
	// Ensure CGO_ENABLED=0 for static agent builds.
	found := false
	for _, e := range env {
		if strings.HasPrefix(e, "CGO_ENABLED=") {
			found = true
			break
		}
	}
	if !found {
		env = append(env, "CGO_ENABLED=0")
	}
	return env
}
