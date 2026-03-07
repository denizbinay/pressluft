package main

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/agentauth"
	"pressluft/internal/auth"
	"pressluft/internal/database"
	"pressluft/internal/dispatch"
	"pressluft/internal/orchestrator"
	"pressluft/internal/pki"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/registration"
	"pressluft/internal/runner/ansible"
	"pressluft/internal/security"
	"pressluft/internal/server"
	"pressluft/internal/worker"
	"pressluft/internal/ws"

	// Register provider implementations.
	_ "pressluft/internal/provider/hetzner"
)

const defaultAddr = ":8080"

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	executionMode, err := platform.NormalizeControlPlaneExecutionMode(os.Getenv("PRESSLUFT_EXECUTION_MODE"), isDevBuild())
	if err != nil {
		log.Fatalf("resolve execution mode: %v", err)
	}
	logExecutionMode(logger, executionMode)

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
	agentTokenStore := agentauth.NewStore(db.DB)
	pkiStore := pki.NewStore(db.DB)
	registrationStore := registration.NewStore(db.DB)
	authStore := auth.NewStore(db.DB)
	ca, err := pki.LoadOrCreateCA(db.DB, ageKeyPath, resolveCAKeyPath())
	if err != nil {
		log.Fatalf("load or create CA: %v", err)
	}
	sessionSecretPath := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_KEY_PATH"))
	sessionSecret, resolvedSessionSecretPath, err := auth.LoadSessionSecret(sessionSecretPath)
	if err != nil {
		log.Fatalf("load session secret: %v", err)
	}
	logger.Info("session secret ready", "path", resolvedSessionSecretPath)

	idleTimeout, absoluteTimeout, err := resolveSessionTimeouts()
	if err != nil {
		log.Fatalf("resolve session timeouts: %v", err)
	}
	secureSessionCookies := resolveSecureSessionCookies(executionMode)
	authService := auth.NewService(authStore, sessionSecret, idleTimeout, absoluteTimeout, secureSessionCookies)

	bootstrapEmail, bootstrapPassword, err := auth.BootstrapCredentials(executionMode)
	if err != nil && executionMode != platform.ExecutionModeDev {
		log.Fatalf("resolve bootstrap admin credentials: %v", err)
	}
	bootstrapUser, err := authService.EnsureBootstrapAdmin(context.Background(), bootstrapEmail, bootstrapPassword)
	if err != nil && executionMode != platform.ExecutionModeDev {
		log.Fatalf("ensure bootstrap admin: %v", err)
	}
	if bootstrapUser != nil {
		logger.Info("bootstrap admin created", "email", bootstrapUser.Email)
		_, _ = activityStore.Emit(context.Background(), activity.EmitInput{
			EventType:    activity.EventSecurityBootstrapAdmin,
			Category:     activity.CategorySecurity,
			Level:        activity.LevelInfo,
			ResourceType: activity.ResourceAccount,
			ActorType:    activity.ActorSystem,
			ActorID:      fmt.Sprintf("%d", bootstrapUser.ID),
			Title:        fmt.Sprintf("Bootstrap admin %s created", bootstrapUser.Email),
		})
	}

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

	controlPlaneURL := strings.TrimSpace(os.Getenv("PRESSLUFT_CONTROL_PLANE_URL"))
	if err := providerStore.BackfillEncryptedTokens(context.Background()); err != nil {
		log.Fatalf("backfill provider tokens: %v", err)
	}

	ansibleRunner := ansible.NewAdapter(ansibleBinary, ansibleDir, []string{
		playbookPath,
		configurePlaybookPath,
		deletePlaybookPath,
		rebuildPlaybookPath,
		resizePlaybookPath,
		firewallsPlaybookPath,
		volumePlaybookPath,
	})

	hub := ws.NewHub()
	agentRunner := dispatch.NewAgentRunner(hub, jobStore, logger)

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
			ControlPlaneURL:       controlPlaneURL,
			ExecutionMode:         executionMode,
			DevTokenStore:         agentTokenStore,
			RegistrationStore:     registrationStore,
			AgentRunner:           agentRunner,
		},
		logger,
	)
	w := worker.New(jobStore, executor, logger, worker.DefaultConfig())

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start worker in background
	go w.Run(ctx)

	resultWaiter := ws.NewResultWaiter()
	hub.SetResultWaiter(resultWaiter)
	completer := dispatch.NewCompleter(jobStore, activityStore, logger)
	wsHandler := ws.NewHandler(hub, completer, resultWaiter, serverStore, logger)
	wsHTTPHandler := server.NewWSHandler(hub, wsHandler, pkiStore, agentTokenStore, logger)
	nodeHandler := server.NewNodeHandler(db.DB, pkiStore, registrationStore, ca, logger)

	monitor := ws.NewMonitor(hub, serverStore, logger)
	go monitor.Start(ctx)

	var operatorAuthenticator auth.Authenticator
	if executionMode == platform.ExecutionModeDev {
		operatorAuthenticator = auth.NewDevAuthenticator()
	} else {
		operatorAuthenticator = auth.NewSessionAuthenticator(authService)
	}

	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandlerWithOptions(db.DB, hub, wsHTTPHandler, nodeHandler, server.HandlerOptions{Authenticator: operatorAuthenticator, AuthService: authService, IsDev: executionMode == platform.ExecutionModeDev}), logger),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       2 * time.Minute,
		MaxHeaderBytes:    1 << 20,
	}
	listenAndServe := func() error {
		return httpServer.ListenAndServe()
	}
	if executionMode == platform.ExecutionModeProductionBootstrap {
		tlsCertFile, tlsKeyFile, err := resolveProductionTLSConfig(controlPlaneURL)
		if err != nil {
			log.Fatalf("resolve production TLS config: %v", err)
		}
		httpServer.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
			ClientAuth: tls.VerifyClientCertIfGiven,
			ClientCAs:  ca.CertPool(),
		}
		listenAndServe = func() error {
			return httpServer.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
		}
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
	if err := listenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}

	logger.Info("pressluft stopped")
}

func logExecutionMode(logger *slog.Logger, mode platform.ExecutionMode) {
	logger.Info("platform contract loaded",
		"execution_mode", mode,
		"contract_ref", "README.md#platform-contract",
		"lifecycle_note", "docs/internal/lifecycle-state-semantics.md",
		"glossary", "docs/glossary.md",
	)

	switch mode {
	case platform.ExecutionModeDev:
		logger.Info("development transport enabled", "agent_trust", "dev websocket token", "server_tls", "not required")
	case platform.ExecutionModeSingleNodeLocal:
		logger.Warn("single-node local control plane mode is for local infrastructure work only", "agent_bootstrap", "disabled", "provider_support", "hetzner only")
	case platform.ExecutionModeProductionBootstrap:
		logger.Info("production bootstrap path enabled", "server_tls", "required in-process", "agent_transport", "wss plus mTLS")
	}
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

	dataDir := resolveDataDir()
	return filepath.Join(dataDir, "pressluft", "pressluft.db")
}

func resolveDataDir() string {
	dataDir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return dataDir
}

func resolveCAKeyPath() string {
	if p := strings.TrimSpace(os.Getenv("PRESSLUFT_CA_KEY_PATH")); p != "" {
		return p
	}
	return filepath.Join(resolveDataDir(), "pressluft", "ca.key")
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

func resolveProductionTLSConfig(controlPlaneURL string) (string, string, error) {
	tlsCertFile := strings.TrimSpace(os.Getenv("PRESSLUFT_TLS_CERT_FILE"))
	tlsKeyFile := strings.TrimSpace(os.Getenv("PRESSLUFT_TLS_KEY_FILE"))
	if tlsCertFile == "" || tlsKeyFile == "" {
		return "", "", fmt.Errorf("PRESSLUFT_TLS_CERT_FILE and PRESSLUFT_TLS_KEY_FILE are required in production-bootstrap mode")
	}
	parsedURL, err := url.Parse(strings.TrimSpace(controlPlaneURL))
	if err != nil {
		return "", "", fmt.Errorf("parse PRESSLUFT_CONTROL_PLANE_URL: %w", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return "", "", fmt.Errorf("PRESSLUFT_CONTROL_PLANE_URL must be an https URL in production-bootstrap mode")
	}
	return tlsCertFile, tlsKeyFile, nil
}

func resolveSessionTimeouts() (time.Duration, time.Duration, error) {
	idle := auth.DefaultSessionIdleTimeout
	absolute := auth.DefaultSessionAbsoluteTimeout
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_IDLE_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("parse PRESSLUFT_SESSION_IDLE_TIMEOUT: %w", err)
		}
		idle = parsed
	}
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT")); raw != "" {
		parsed, err := time.ParseDuration(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("parse PRESSLUFT_SESSION_ABSOLUTE_TIMEOUT: %w", err)
		}
		absolute = parsed
	}
	return idle, absolute, nil
}

func resolveSecureSessionCookies(mode platform.ExecutionMode) bool {
	if raw := strings.TrimSpace(os.Getenv("PRESSLUFT_SESSION_COOKIE_SECURE")); raw != "" {
		return raw == "1" || strings.EqualFold(raw, "true")
	}
	return mode == platform.ExecutionModeProductionBootstrap
}
