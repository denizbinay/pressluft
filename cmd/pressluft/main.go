package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	"pressluft/internal/promotion"
	"pressluft/internal/secrets"
	"pressluft/internal/settings"
	"pressluft/internal/sites"
	"pressluft/internal/ssh"
	"pressluft/internal/store"
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
	var listenAddr string
	var webDistDir string
	flag.StringVar(&dbPath, "db", defaultDBPath(), "sqlite database path")
	flag.StringVar(&nodeProvisionPlaybook, "node-provision-playbook", "ansible/playbooks/node-provision.yml", "path to node_provision ansible playbook")
	flag.StringVar(&siteImportPlaybook, "site-import-playbook", "ansible/playbooks/site-import.yml", "path to site_import ansible playbook")
	flag.StringVar(&listenAddr, "listen", ":8080", "http listen address")
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	switch command {
	case "bootstrap":
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
	case "worker-once":
		nodeProvisionExecuted, err := jobs.ExecuteQueuedNodeProvision(ctx, db, jobs.ExecRunner{}, nodeProvisionPlaybook)
		if err != nil {
			return err
		}
		siteImportExecuted, err := migration.ExecuteQueuedSiteImport(ctx, db, migration.ExecRunner{}, siteImportPlaybook)
		if err != nil {
			return err
		}
		fmt.Printf("worker-once complete: node_provision_executed=%t site_import_executed=%t\n", nodeProvisionExecuted, siteImportExecuted)
		return nil
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
		fmt.Printf("pressluft api listening on %s\n", listenAddr)
		return http.ListenAndServe(listenAddr, apiServer.Handler())
	default:
		return fmt.Errorf("unknown command %q", command)
	}
}

func defaultDBPath() string {
	if path := os.Getenv("PRESSLUFT_DB_PATH"); path != "" {
		return path
	}
	return "./pressluft.db"
}

func defaultSecretsDir() string {
	if path := os.Getenv("PRESSLUFT_SECRETS_DIR"); path != "" {
		return path
	}
	return "/var/lib/pressluft/secrets"
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
