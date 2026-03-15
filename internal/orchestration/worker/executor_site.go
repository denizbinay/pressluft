package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/controlplane/activity"
	serverpkg "pressluft/internal/controlplane/server"
	"pressluft/internal/infra/runner"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/security"
)

func (e *Executor) executeDeploySite(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for site deployment job")
	}
	if e.siteStore == nil {
		return e.failJob(ctx, job, "site store not configured")
	}
	if e.domainStore == nil {
		return e.failJob(ctx, job, "domain store not configured")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}

	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning, CurrentStep: "validate"}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceSite,
		ParentResourceID:   e.siteIDForJob(*job),
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	payload, err := orchestrator.UnmarshalDeploySitePayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	site, err := e.siteStore.GetByID(ctx, payload.SiteID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("site not found: %v", err))
	}
	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}
	primaryDomain, err := e.primaryDomainForSite(ctx, site.ID)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	if strings.TrimSpace(site.PrimaryDomain) == "" {
		return e.failJob(ctx, job, "site primary hostname is required for deployment")
	}

	_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateDeploying, fmt.Sprintf("Deploying WordPress to %s.", primaryDomain.Hostname), job.ID, "")
	_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStatePending, "Deploy is running. Runtime health will be verified before the site is marked live.", "")
	_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStatePending, "Applying server routing for this hostname.", time.Now().UTC())

	e.emitStepStart(ctx, job.ID, "validate", "Validating site deployment inputs")
	if strings.TrimSpace(server.ProfileKey) != "nginx-stack" {
		return e.failJob(ctx, job, fmt.Sprintf("server profile %q is not supported for site deployment", server.ProfileKey))
	}
	if server.Status != platform.ServerStatusReady {
		return e.failJob(ctx, job, "server must be ready before deploying a site")
	}
	if server.SetupState != platform.SetupStateReady {
		return e.failJob(ctx, job, "server setup must be ready before deploying a site")
	}
	storedKey, err := e.serverStore.GetKey(ctx, server.ID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to read SSH key: %v", err))
	}
	if storedKey == nil {
		return e.failJob(ctx, job, "missing SSH key for server")
	}
	decryptedKey, err := security.Decrypt(storedKey.PrivateKeyEncrypted)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to decrypt SSH key: %v", err))
	}
	e.emitStepComplete(ctx, job.ID, "validate", "Site deployment request validated")

	e.updateStep(ctx, job.ID, "deploy")
	e.emitStepStart(ctx, job.ID, "deploy", "Creating site files, database, and WordPress config")
	if err := e.runSiteDeployPlaybook(ctx, job.ID, server, site, primaryDomain, string(decryptedKey), payload.TLSContactEmail); err != nil {
		_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateIssue, err.Error(), time.Now().UTC())
		_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateFailed, err.Error(), job.ID, "")
		_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateIssue, err.Error(), time.Now().UTC().Format(time.RFC3339))
		return e.failJob(ctx, job, fmt.Sprintf("site deploy failed: %v", err))
	}
	e.emitStepComplete(ctx, job.ID, "deploy", "Site files and WordPress installation applied")

	e.updateStep(ctx, job.ID, "verify")
	e.emitStepStart(ctx, job.ID, "verify", "Verifying deployed hostname and WordPress runtime")
	if err := e.verifySiteDeployment(ctx, *site, *primaryDomain); err != nil {
		_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateIssue, err.Error(), time.Now().UTC())
		_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateFailed, err.Error(), job.ID, "")
		_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateIssue, err.Error(), time.Now().UTC().Format(time.RFC3339))
		return e.failJob(ctx, job, fmt.Sprintf("site verification failed: %v", err))
	}
	_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateReady, "Hostname routing verified over HTTPS.", time.Now().UTC())
	_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateHealthy, fmt.Sprintf("WordPress rendered successfully at https://%s/.", primaryDomain.Hostname), time.Now().UTC().Format(time.RFC3339))
	e.emitStepComplete(ctx, job.ID, "verify", "Hostname routing and WordPress runtime verified")

	now := time.Now().UTC().Format(time.RFC3339)
	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing site deployment")
	_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateReady, fmt.Sprintf("Site is live at https://%s/.", primaryDomain.Hostname), job.ID, now)
	e.emitStepComplete(ctx, job.ID, "finalize", "Site deployment complete")
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventSiteDeployed,
		Category:           activity.CategorySite,
		Level:              activity.LevelSuccess,
		ResourceType:       activity.ResourceSite,
		ResourceID:         site.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   site.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("Site '%s' deployed", site.Name),
		Message:            fmt.Sprintf("WordPress is live at https://%s/.", primaryDomain.Hostname),
	})
	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) primaryDomainForSite(ctx context.Context, siteID string) (*serverpkg.StoredDomain, error) {
	domains, err := e.domainStore.ListBySite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list site domains: %w", err)
	}
	for i := range domains {
		if domains[i].IsPrimary {
			return &domains[i], nil
		}
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("site has no assigned hostname")
	}
	return &domains[0], nil
}

func (e *Executor) runSiteDeployPlaybook(ctx context.Context, jobID string, server *serverpkg.StoredServer, site *serverpkg.StoredSite, primaryDomain *serverpkg.StoredDomain, privateKey, tlsContactEmail string) error {
	if server == nil || site == nil || primaryDomain == nil {
		return fmt.Errorf("site deployment context is incomplete")
	}
	effectiveTLSContactEmail, err := e.resolveACMEContactEmail(strings.TrimSpace(tlsContactEmail), strings.TrimSpace(site.WordPressAdminEmail))
	if err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "pressluft-site-deploy-")
	if err != nil {
		return fmt.Errorf("failed to create deploy workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	privateKeyPath := filepath.Join(workspace, "server.key")
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	inventoryPath := filepath.Join(workspace, "deploy.ini")
	inventory := fmt.Sprintf("server ansible_host=%s ansible_user=root ansible_ssh_private_key_file=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'\n", server.IPv4, privateKeyPath)
	if err := os.WriteFile(inventoryPath, []byte(inventory), 0o600); err != nil {
		return fmt.Errorf("failed to write deploy inventory: %w", err)
	}

	dbSuffix := strings.ReplaceAll(site.ID[:8], "-", "")
	dbName := "pl_" + dbSuffix
	dbUser := "pl_" + dbSuffix
	dbPassword, err := randomHex(24)
	if err != nil {
		return fmt.Errorf("generate database password: %w", err)
	}
	adminPassword, err := randomHex(24)
	if err != nil {
		return fmt.Errorf("generate admin password: %w", err)
	}
	secretKey, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("generate secret key: %w", err)
	}
	deployPath := effectiveWordPressPath(*site)
	request := runner.Request{
		JobID:         jobID,
		InventoryPath: inventoryPath,
		PlaybookPath:  e.siteDeployPlaybook(),
		ExtraVars: map[string]string{
			"profile_key":       server.ProfileKey,
			"site_id":           site.ID,
			"site_name":         site.Name,
			"hostname":          primaryDomain.Hostname,
			"site_path":         deployPath,
			"php_version":       firstNonEmpty(site.PHPVersion, "8.3"),
			"wordpress_version": firstNonEmpty(site.WordPressVersion, "6.8"),
			"db_name":           dbName,
			"db_user":           dbUser,
			"db_password":       dbPassword,
			"admin_user":        "pressluft",
			"admin_password":    adminPassword,
			"admin_email":       firstNonEmpty(site.WordPressAdminEmail, fmt.Sprintf("admin@%s", primaryDomain.Hostname)),
			"tls_contact_email": effectiveTLSContactEmail,
			"secret_key":        secretKey,
		},
	}
	return e.runner.Run(ctx, request, &runnerEventSink{jobStore: e.jobStore, jobID: jobID, logger: e.logger})
}

func (e *Executor) resolveACMEContactEmail(operatorEmail, siteAdminEmail string) (string, error) {
	if isUsableACMEContactEmail(operatorEmail) {
		return operatorEmail, nil
	}
	if e.executionMode == platform.ExecutionModeDev && isUsableACMEContactEmail(siteAdminEmail) {
		return siteAdminEmail, nil
	}
	if e.executionMode == platform.ExecutionModeDev {
		return "", fmt.Errorf("no usable ACME contact email available: update the site WordPress admin email to a real address")
	}
	return "", fmt.Errorf("no usable ACME contact email available for certificate issuance")
}

func isUsableACMEContactEmail(email string) bool {
	address, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return false
	}
	parts := strings.Split(address.Address, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(parts[1]))
	if domain == "" || !strings.Contains(domain, ".") {
		return false
	}
	if domain == "localhost" || domain == "localdomain" {
		return false
	}
	if domain == "example.com" || domain == "example.net" || domain == "example.org" {
		return false
	}
	if strings.HasSuffix(domain, ".example") || strings.HasSuffix(domain, ".invalid") || strings.HasSuffix(domain, ".localhost") || strings.HasSuffix(domain, ".test") {
		return false
	}
	return true
}

func (e *Executor) verifySiteDeployment(ctx context.Context, site serverpkg.StoredSite, primaryDomain serverpkg.StoredDomain) error {
	if err := serverpkg.VerifyPublicSiteRouting(ctx, site.ID, primaryDomain.Hostname); err != nil {
		return err
	}
	if err := serverpkg.VerifyPublicWordPressRuntime(ctx, primaryDomain.Hostname); err != nil {
		return err
	}
	return nil
}

func effectiveWordPressPath(site serverpkg.StoredSite) string {
	path := strings.TrimSpace(site.WordPressPath)
	if path == "" || path == "/srv/www/" {
		return filepath.ToSlash(filepath.Join("/srv/www/pressluft/sites", site.ID, "current"))
	}
	return path
}

func randomHex(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	const alphabet = "0123456789abcdef"
	out := make([]byte, length)
	for i := range out {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
