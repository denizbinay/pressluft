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
	"pressluft/internal/envconfig"
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

	_ "pressluft/internal/provider/hetzner"
)

const defaultAddr = ":8080"

type playbookPaths struct {
	basePath  string
	configure string
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cwd, _ := os.Getwd()
	runtimeConfig, err := envconfig.ResolveControlPlaneRuntime(isDevBuild(), cwd)
	if err != nil {
		log.Fatalf("resolve control-plane config: %v", err)
	}
	logger.Info(
		"runtime paths resolved",
		"mode", envconfig.Mode,
		"data_dir", runtimeConfig.DataDir,
		"db_path", runtimeConfig.DBPath,
		"age_key_path", runtimeConfig.AgeKeyPath,
		"ca_key_path", runtimeConfig.CAKeyPath,
		"session_secret_path", runtimeConfig.SessionSecretPath,
	)
	executionMode := runtimeConfig.ExecutionMode
	logExecutionMode(logger, executionMode)

	allowGenerate := strings.TrimSpace(os.Getenv("PRESSLUFT_AGE_KEY_PATH")) == ""
	generated, err := security.EnsureAgeKey(runtimeConfig.AgeKeyPath, allowGenerate)
	if err != nil {
		log.Fatalf("ensure age key: %v", err)
	}
	if generated {
		logger.Info("age key generated", "path", runtimeConfig.AgeKeyPath)
	}

	db, err := database.Open(runtimeConfig.DBPath, logger)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer db.Close()

	jobStore := orchestrator.NewStore(db.DB)
	serverStore := server.NewServerStore(db.DB)
	providerStore := provider.NewStore(db.DB)
	siteStore := server.NewSiteStore(db.DB)
	domainStore := server.NewDomainStore(db.DB)
	activityStore := activity.NewStore(db.DB)
	agentTokenStore := agentauth.NewStore(db.DB)
	pkiStore := pki.NewStore(db.DB)
	registrationStore := registration.NewStore(db.DB)
	authStore := auth.NewStore(db.DB)
	ca, err := pki.LoadOrCreateCA(db.DB, runtimeConfig.AgeKeyPath, runtimeConfig.CAKeyPath)
	if err != nil {
		log.Fatalf("load or create CA: %v", err)
	}
	sessionSecret, resolvedSessionSecretPath, err := auth.LoadSessionSecret(runtimeConfig.SessionSecretPath)
	if err != nil {
		log.Fatalf("load session secret: %v", err)
	}
	logger.Info("session secret ready", "path", resolvedSessionSecretPath)

	authService := auth.NewService(authStore, sessionSecret, runtimeConfig.SessionIdleTimeout, runtimeConfig.SessionAbsoluteTimeout, runtimeConfig.SessionCookieSecure)

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
			ActorID:      bootstrapUser.ID,
			Title:        fmt.Sprintf("Bootstrap admin %s created", bootstrapUser.Email),
		})
	}

	ansibleBinary, err := resolveAnsibleBinary(runtimeConfig.AnsibleBinary, runtimeConfig.AnsibleDir)
	if err != nil {
		log.Fatalf("resolve ansible binary: %v", err)
	}
	if err := logAnsibleVersion(ansibleBinary, runtimeConfig.AnsibleDir, logger); err != nil {
		log.Fatalf("ansible preflight failed: %v", err)
	}
	playbooks := defaultPlaybookPaths()

	controlPlaneURL := runtimeConfig.ControlPlaneURL
	if executionMode == platform.ExecutionModeDev && platform.DetectCallbackURLMode(controlPlaneURL) == platform.CallbackURLModeEphemeral {
		logger.Warn("ephemeral dev callback URL detected",
			"control_plane_url", controlPlaneURL,
			"reconnect_durability", "remote agents configured against Cloudflare quick tunnels will not reconnect after control-plane restart",
		)
	}

	ansibleRunner := ansible.NewAdapter(ansibleBinary, runtimeConfig.AnsibleDir, []string{
		playbooks.configure,
		playbooks.basePath + "/",
	})

	hub := ws.NewHub()
	agentRunner := dispatch.NewAgentRunner(hub, jobStore, logger)
	executor := worker.NewExecutor(
		jobStore,
		worker.NewServerStoreAdapter(serverStore),
		worker.NewProviderStoreAdapter(providerStore),
		worker.NewSiteStoreAdapter(siteStore),
		worker.NewDomainStoreAdapter(domainStore),
		activityStore,
		ansibleRunner,
		worker.ExecutorConfig{
			PlaybookBasePath:      playbooks.basePath,
			ConfigurePlaybookPath: playbooks.configure,
			ControlPlaneURL:       controlPlaneURL,
			ExecutionMode:         executionMode,
			DevTokenStore:         agentTokenStore,
			RegistrationStore:     registrationStore,
			AgentRunner:           agentRunner,
		},
		logger,
	)
	w := worker.New(jobStore, executor, logger, worker.DefaultConfig())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go w.Run(ctx)

	resultWaiter := ws.NewResultWaiter()
	hub.SetResultWaiter(resultWaiter)
	completer := dispatch.NewCompleter(jobStore, activityStore, logger)
	wsHandler := ws.NewHandler(hub, completer, resultWaiter, serverStore, logger)
	wsHTTPHandler := server.NewWSHandler(hub, wsHandler, pkiStore, agentTokenStore, logger)
	nodeHandler := server.NewNodeHandler(db.DB, pkiStore, registrationStore, ca, logger)

	monitor := ws.NewMonitor(hub, serverStore, logger)
	go monitor.Start(ctx)
	siteHealthMonitor := server.NewSiteHealthMonitor(siteStore, domainStore, activityStore, hub, logger)
	go siteHealthMonitor.Start(ctx)

	operatorAuthenticator := operatorAuthenticatorForMode(executionMode, authService)
	httpServer := &http.Server{
		Addr:              resolveAddr(),
		Handler:           server.WithRequestLogging(server.NewHandlerWithOptions(db.DB, hub, wsHTTPHandler, nodeHandler, server.HandlerOptions{Authenticator: operatorAuthenticator, AuthService: authService, IsDev: executionMode == platform.ExecutionModeDev, ControlPlaneURL: controlPlaneURL}), logger),
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
		listenAndServe = configureProductionTLSServer(httpServer, ca, controlPlaneURL, runtimeConfig.TLSCertFile, runtimeConfig.TLSKeyFile)
	}

	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		logger.Info("shutdown signal received")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			logger.Error("http server shutdown error", "error", err)
		}
	}()

	logger.Info("pressluft listening", "addr", httpServer.Addr, "mode", envconfig.Mode)
	if err := listenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}

	logger.Info("pressluft stopped")
}

func defaultPlaybookPaths() playbookPaths {
	return playbookPaths{
		basePath:  "ops/ansible/playbooks",
		configure: "ops/ansible/playbooks/configure.yml",
	}
}

func operatorAuthenticatorForMode(mode platform.ExecutionMode, authService *auth.Service) auth.Authenticator {
	if mode == platform.ExecutionModeDev {
		return auth.NewDevAuthenticator()
	}
	return auth.NewSessionAuthenticator(authService)
}

func logExecutionMode(logger *slog.Logger, mode platform.ExecutionMode) {
	logger.Info("platform contract loaded",
		"execution_mode", mode,
		"contract_package", "pressluft/internal/contract",
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

func resolveAnsibleBinary(ansibleBinary, ansibleDir string) (string, error) {
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

func configureProductionTLSServer(httpServer *http.Server, ca *pki.CA, controlPlaneURL, tlsCertFile, tlsKeyFile string) func() error {
	if err := resolveProductionTLSConfig(controlPlaneURL, tlsCertFile, tlsKeyFile); err != nil {
		log.Fatalf("resolve production TLS config: %v", err)
	}
	httpServer.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS12,
		ClientAuth: tls.VerifyClientCertIfGiven,
		ClientCAs:  ca.CertPool(),
	}
	return func() error {
		return httpServer.ListenAndServeTLS(tlsCertFile, tlsKeyFile)
	}
}

func resolveProductionTLSConfig(controlPlaneURL, tlsCertFile, tlsKeyFile string) error {
	if strings.TrimSpace(tlsCertFile) == "" || strings.TrimSpace(tlsKeyFile) == "" {
		return fmt.Errorf("PRESSLUFT_TLS_CERT_FILE and PRESSLUFT_TLS_KEY_FILE are required in production-bootstrap mode")
	}
	parsedURL, err := url.Parse(strings.TrimSpace(controlPlaneURL))
	if err != nil {
		return fmt.Errorf("parse PRESSLUFT_CONTROL_PLANE_URL: %w", err)
	}
	if parsedURL.Scheme != "https" || parsedURL.Host == "" {
		return fmt.Errorf("PRESSLUFT_CONTROL_PLANE_URL must be an https URL in production-bootstrap mode")
	}
	return nil
}
