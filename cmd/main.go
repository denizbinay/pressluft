package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/database"
	"pressluft/internal/orchestrator"
	"pressluft/internal/provider"
	"pressluft/internal/runner/ansible"
	"pressluft/internal/security"
	"pressluft/internal/server"
	"pressluft/internal/worker"

	// Register provider implementations.
	_ "pressluft/internal/provider/hetzner"
)

const defaultAddr = ":8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	ageKeyPath := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH"))
	allowGenerate := false
	if ageKeyPath == "" {
		ageKeyPath = security.DefaultAgeKeyPath()
		allowGenerate = true
	}
	generated, err := security.EnsureAgeKey(ageKeyPath, allowGenerate)
	if err != nil {
		log.Fatalf("ensure age key: %v", err)
	}
	if generated {
		logger.Info("age key generated", "path", ageKeyPath)
	}

	db, err := database.Open(resolveDBPath(), logger)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	// Create stores for worker
	jobStore := orchestrator.NewStore(db.DB)
	serverStore := server.NewServerStore(db.DB)
	providerStore := provider.NewStore(db.DB)
	activityStore := activity.NewStore(db.DB)

	ansibleDir := strings.TrimSpace(os.Getenv("PRESSLUFT_ANSIBLE_DIR"))
	if ansibleDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			ansibleDir = cwd
		} else {
			ansibleDir = "."
		}
	}
	ansibleBinary, err := resolveAnsibleBinary(ansibleDir)
	if err != nil {
		log.Fatalf("resolve ansible binary: %v", err)
	}
	if err := logAnsibleVersion(ansibleBinary, ansibleDir, logger); err != nil {
		log.Fatalf("ansible preflight failed: %v", err)
	}
	playbookPath := "ops/ansible/playbooks/provision.yml"
	configurePlaybookPath := "ops/ansible/playbooks/configure.yml"
	deletePlaybookPath := "ops/ansible/playbooks/delete_server.yml"
	rebuildPlaybookPath := "ops/ansible/playbooks/rebuild_server.yml"
	resizePlaybookPath := "ops/ansible/playbooks/resize_server.yml"
	firewallsPlaybookPath := "ops/ansible/playbooks/update_firewalls.yml"
	volumePlaybookPath := "ops/ansible/playbooks/manage_volume.yml"

	ansibleRunner := ansible.NewAdapter(ansibleBinary, ansibleDir, []string{
		playbookPath,
		configurePlaybookPath,
		deletePlaybookPath,
		rebuildPlaybookPath,
		resizePlaybookPath,
		firewallsPlaybookPath,
		volumePlaybookPath,
	})

	// Create worker with executor
	executor := worker.NewExecutor(
		jobStore,
		worker.NewServerStoreAdapter(serverStore),
		worker.NewProviderStoreAdapter(providerStore),
		activityStore,
		ansibleRunner,
		worker.ExecutorConfig{
			ProvisionPlaybookPath: playbookPath,
			ConfigurePlaybookPath: configurePlaybookPath,
			DeletePlaybookPath:    deletePlaybookPath,
			RebuildPlaybookPath:   rebuildPlaybookPath,
			ResizePlaybookPath:    resizePlaybookPath,
			FirewallsPlaybookPath: firewallsPlaybookPath,
			VolumePlaybookPath:    volumePlaybookPath,
		},
		logger,
	)
	w := worker.New(jobStore, executor, logger, worker.DefaultConfig())

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in background
	go w.Run(ctx)

	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandler(db.DB), logger),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Handle shutdown signals
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info("shutdown signal received")
		cancel() // Stop worker

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", "error", err)
		}
	}()

	logger.Info("pressluft listening", "addr", httpServer.Addr)
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}

	logger.Info("pressluft stopped")
}

func resolveAddr() string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return defaultAddr
	}
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

func resolveDBPath() string {
	if p := os.Getenv("PRESSLUFT_DB"); p != "" {
		return p
	}

	dataDir := os.Getenv("XDG_DATA_HOME")
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataDir, "pressluft", "pressluft.db")
}

func resolveAnsibleBinary(ansibleDir string) (string, error) {
	ansibleBinary := strings.TrimSpace(os.Getenv("PRESSLUFT_ANSIBLE_BIN"))
	if ansibleBinary == "" {
		ansibleBinary = filepath.Join(ansibleDir, ".venv", "bin", "ansible-playbook")
	}
	if !filepath.IsAbs(ansibleBinary) {
		ansibleBinary = filepath.Join(ansibleDir, ansibleBinary)
	}
	ansibleBinary = filepath.Clean(ansibleBinary)
	absBinary, err := filepath.Abs(ansibleBinary)
	if err != nil {
		return "", fmt.Errorf("resolve ansible-playbook path: %w", err)
	}
	ansibleBinary = absBinary
	if err := ensureExecutable(ansibleBinary); err != nil {
		return "", err
	}
	return ansibleBinary, nil
}

func ensureExecutable(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("ansible-playbook not found at %q; set PRESSLUFT_ANSIBLE_BIN to an absolute path", path)
		}
		return fmt.Errorf("stat ansible-playbook: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("ansible-playbook path is a directory: %q", path)
	}
	if info.Mode()&0o111 == 0 {
		return fmt.Errorf("ansible-playbook is not executable: %q", path)
	}
	return nil
}

func logAnsibleVersion(binaryPath, workingDir string, logger *slog.Logger) error {
	cmd := exec.Command(binaryPath, "--version")
	cmd.Dir = workingDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ansible-playbook --version failed: %w (output=%s)", err, strings.TrimSpace(string(output)))
	}
	logger.Info("ansible preflight", "binary", binaryPath, "version", strings.TrimSpace(string(output)))
	return nil
}
