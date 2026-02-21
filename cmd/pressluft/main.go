package main

import (
	"context"
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pressluft/internal/admin"
	"pressluft/internal/api"
	"pressluft/internal/audit"
	"pressluft/internal/auth"
	"pressluft/internal/backups"
	"pressluft/internal/bootstrap"
	"pressluft/internal/domains"
	"pressluft/internal/environments"
	"pressluft/internal/jobs"
	"pressluft/internal/metrics"
	"pressluft/internal/migration"
	"pressluft/internal/migrations"
	"pressluft/internal/promotion"
	"pressluft/internal/secrets"
	"pressluft/internal/settings"
	"pressluft/internal/sites"
	"pressluft/internal/ssh"
	"pressluft/internal/store"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "pressluft: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var dbPath string
	var nodeProvisionPlaybook string
	var siteImportPlaybook string
	var backupCleanupPlaybook string
	var listenAddr string
	var webDistDir string
	flag.StringVar(&dbPath, "db", defaultDBPath(), "sqlite database path")
	flag.StringVar(&nodeProvisionPlaybook, "node-provision-playbook", "ansible/playbooks/node-provision.yml", "path to node_provision ansible playbook")
	flag.StringVar(&siteImportPlaybook, "site-import-playbook", "ansible/playbooks/site-import.yml", "path to site_import ansible playbook")
	flag.StringVar(&backupCleanupPlaybook, "backup-cleanup-playbook", "ansible/playbooks/backup-cleanup.yml", "path to backup_cleanup ansible playbook")
	flag.StringVar(&listenAddr, "listen", defaultListenAddr(), "http listen address")
	flag.StringVar(&webDistDir, "web-dist-dir", defaultWebDistDir(), "path to built dashboard assets")
	flag.Parse()

	command := "bootstrap"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	db, err := store.OpenSQLite(dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	baseCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	switch command {
	case "version":
		fmt.Printf("pressluft %s (commit=%s built=%s)\n", buildVersion, buildCommit, buildDate)
		return nil
	case "migrate":
		action := "up"
		if flag.NArg() > 1 {
			action = flag.Arg(1)
		}
		if err := migrations.Run(action, filepath.Join(".", "migrations"), dbPath); err != nil {
			return err
		}
		fmt.Printf("migrate %s complete\n", action)
		return nil
	case "bootstrap":
		ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()
		hostname, err := os.Hostname()
		if err != nil {
			return fmt.Errorf("resolve hostname: %w", err)
		}

		result, err := bootstrap.Run(ctx, db, hostname)
		if err != nil {
			return err
		}

		fmt.Printf("bootstrap complete: node=%s created=%t provision_job=%t status=%s\n", result.NodeID, result.NodeCreated, result.NodeProvisionJob, result.NodeCurrentStatus)
		return nil
	case "admin":
		return runAdmin(baseCtx, db, flag.Args()[1:])
	case "worker-once":
		ctx, cancel := context.WithTimeout(baseCtx, 35*time.Minute)
		defer cancel()
		nodeProvisionExecuted, err := jobs.ExecuteQueuedNodeProvision(ctx, db, jobs.ExecRunner{}, nodeProvisionPlaybook)
		if err != nil {
			return err
		}
		siteImportExecuted, err := migration.ExecuteQueuedSiteImport(ctx, db, jobs.ExecRunner{}, siteImportPlaybook)
		if err != nil {
			return err
		}
		backupCleanupExecuted, err := backups.ExecuteQueuedBackupCleanup(ctx, db, jobs.ExecRunner{}, backupCleanupPlaybook)
		if err != nil {
			return err
		}
		fmt.Printf("worker-once complete: node_provision_executed=%t site_import_executed=%t backup_cleanup_executed=%t\n", nodeProvisionExecuted, siteImportExecuted, backupCleanupExecuted)
		return nil
	case "worker":
		return runWorker(baseCtx, db, nodeProvisionPlaybook, siteImportPlaybook, backupCleanupPlaybook)
	case "serve":
		authService := auth.NewService(db)
		siteService := sites.NewService(db)
		environmentService := environments.NewService(db)
		promotionService := promotion.NewService(db)
		magicLoginService := ssh.NewService(db, ssh.ExecRunner{})
		secretStore := secrets.NewStore(defaultSecretsDir())
		settingsService := settings.NewService(db, secretStore)
		jobsService := jobs.NewService(db)
		metricsService := metrics.NewService(db)
		backupService := backups.NewService(db)
		domainService := domains.NewService(db)
		migrationService := migration.NewService(db)
		auditService := audit.NewService(db)
		serverOptions := []api.ServerOption{}
		if hasDashboardDist(webDistDir) {
			serverOptions = append(serverOptions, api.WithDashboardFS(os.DirFS(webDistDir)))
			fmt.Printf("pressluft dashboard assets loaded from %s\n", webDistDir)
		} else {
			fmt.Printf("pressluft dashboard assets not found at %s, serving API routes only\n", webDistDir)
		}

		apiServer := api.NewServer(authService, siteService, environmentService, promotionService, magicLoginService, settingsService, jobsService, metricsService, backupService, domainService, migrationService, auditService, serverOptions...)
		srv := &http.Server{Addr: listenAddr, Handler: apiServer.Handler()}
		serveErr := make(chan error, 1)
		fmt.Printf("pressluft api listening on %s\n", listenAddr)
		go func() {
			serveErr <- srv.ListenAndServe()
		}()

		select {
		case err := <-serveErr:
			if errors.Is(err, http.ErrServerClosed) {
				return nil
			}
			return err
		case <-baseCtx.Done():
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := srv.Shutdown(shutdownCtx); err != nil {
				_ = srv.Close()
				return fmt.Errorf("shutdown server: %w", err)
			}
			err := <-serveErr
			if errors.Is(err, http.ErrServerClosed) || err == nil {
				return nil
			}
			return err
		}
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func runAdmin(ctx context.Context, db *sql.DB, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: pressluft admin <init|set-password>")
	}

	sub := args[0]
	switch sub {
	case "init":
		fs := flag.NewFlagSet("pressluft admin init", flag.ContinueOnError)
		fs.SetOutput(io.Discard)

		email := strings.TrimSpace(os.Getenv("PRESSLUFT_ADMIN_EMAIL"))
		displayName := strings.TrimSpace(os.Getenv("PRESSLUFT_ADMIN_DISPLAY_NAME"))
		password := strings.TrimSpace(os.Getenv("PRESSLUFT_ADMIN_PASSWORD"))
		fs.StringVar(&email, "email", email, "admin email")
		fs.StringVar(&displayName, "display-name", displayName, "admin display name")
		fs.StringVar(&password, "password", password, "admin password (optional; defaults to PRESSLUFT_ADMIN_PASSWORD)")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("parse flags: %w", err)
		}

		if strings.TrimSpace(email) == "" {
			email = "admin@local"
		}
		if strings.TrimSpace(displayName) == "" {
			displayName = "Admin"
		}

		svc := admin.NewService(db)
		res, err := svc.Init(ctx, admin.InitOptions{Email: email, DisplayName: displayName, Password: password})
		if err != nil {
			if errors.Is(err, admin.ErrAlreadyInitialized) {
				fmt.Printf("admin already initialized\n")
				return nil
			}
			if errors.Is(err, admin.ErrInvalidInput) {
				return fmt.Errorf("invalid admin init input")
			}
			return err
		}

		if res.Created {
			if res.GeneratedPassword != "" {
				fmt.Printf("admin initialized: email=%s password=%s\n", res.Email, res.GeneratedPassword)
				return nil
			}
			fmt.Printf("admin initialized: email=%s\n", res.Email)
			return nil
		}
		fmt.Printf("admin already initialized\n")
		return nil
	case "set-password":
		fs := flag.NewFlagSet("pressluft admin set-password", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		email := strings.TrimSpace(os.Getenv("PRESSLUFT_ADMIN_EMAIL"))
		password := strings.TrimSpace(os.Getenv("PRESSLUFT_ADMIN_PASSWORD"))
		fs.StringVar(&email, "email", email, "admin email")
		fs.StringVar(&password, "password", password, "new admin password")
		if err := fs.Parse(args[1:]); err != nil {
			return fmt.Errorf("parse flags: %w", err)
		}
		if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
			return fmt.Errorf("usage: pressluft admin set-password -email <email> -password <password>")
		}

		svc := admin.NewService(db)
		if err := svc.SetPassword(ctx, email, password); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("admin not found")
			}
			return err
		}
		fmt.Printf("admin password updated\n")
		return nil
	default:
		return fmt.Errorf("unknown admin subcommand %q", sub)
	}
}

func runWorker(ctx context.Context, db *sql.DB, nodeProvisionPlaybook, siteImportPlaybook, backupCleanupPlaybook string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.Canceled {
				return nil
			}
			return ctx.Err()
		case <-ticker.C:
			// Best-effort loop: execute at most one of each supported job type per tick.
			executedAny := false

			executed, err := jobs.ExecuteQueuedNodeProvision(ctx, db, jobs.ExecRunner{}, nodeProvisionPlaybook)
			if err != nil {
				return err
			}
			executedAny = executedAny || executed

			executed, err = migration.ExecuteQueuedSiteImport(ctx, db, jobs.ExecRunner{}, siteImportPlaybook)
			if err != nil {
				return err
			}
			executedAny = executedAny || executed

			executed, err = backups.ExecuteQueuedBackupCleanup(ctx, db, jobs.ExecRunner{}, backupCleanupPlaybook)
			if err != nil {
				return err
			}
			executedAny = executedAny || executed

			if executedAny {
				// Drain quickly when work exists.
				continue
			}
		}
	}
}

func defaultDBPath() string {
	if path := os.Getenv("PRESSLUFT_DB_PATH"); path != "" {
		return path
	}
	return "/var/lib/pressluft/pressluft.db"
}

func defaultSecretsDir() string {
	if path := os.Getenv("PRESSLUFT_SECRETS_DIR"); path != "" {
		return path
	}
	return "/var/lib/pressluft/secrets"
}

func defaultListenAddr() string {
	if addr := os.Getenv("PRESSLUFT_LISTEN_ADDR"); strings.TrimSpace(addr) != "" {
		return strings.TrimSpace(addr)
	}
	return ":8080"
}

func defaultWebDistDir() string {
	if path := os.Getenv("PRESSLUFT_WEB_DIST_DIR"); path != "" {
		return path
	}
	return "./web/.output/public"
}

func hasDashboardDist(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	info, err := os.Stat(filepath.Join(root, "index.html"))
	if err != nil {
		return false
	}
	return !info.IsDir()
}
