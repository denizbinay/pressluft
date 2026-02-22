package sites

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

const siteCreatePlaybookPath = "ansible/playbooks/site-create.yml"

// SiteCreateExecutor executes Ansible for site creation.
type SiteCreateExecutor interface {
	RunSiteCreate(ctx context.Context, node nodes.Node, vars SiteCreateVars) error
}

// SiteCreateVars contains variables passed to site-create.yml playbook.
type SiteCreateVars struct {
	SiteID              string `json:"site_id"`
	SiteSlug            string `json:"site_slug"`
	EnvironmentID       string `json:"environment_id"`
	EnvironmentSlug     string `json:"environment_slug"`
	EnvironmentType     string `json:"environment_type"`
	PreviewURL          string `json:"preview_url"`
	FastCGICacheEnabled bool   `json:"fastcgi_cache_enabled"`
	RedisCacheEnabled   bool   `json:"redis_cache_enabled"`
}

// AnsibleSiteCreateExecutor implements SiteCreateExecutor using Ansible.
type AnsibleSiteCreateExecutor struct {
	runCommand func(ctx context.Context, name string, args ...string) ([]byte, error)
}

// NewAnsibleSiteCreateExecutor creates a new Ansible executor for site creation.
func NewAnsibleSiteCreateExecutor() *AnsibleSiteCreateExecutor {
	return &AnsibleSiteCreateExecutor{runCommand: runCommand}
}

func (e *AnsibleSiteCreateExecutor) RunSiteCreate(ctx context.Context, node nodes.Node, vars SiteCreateVars) error {
	tmpDir, err := os.MkdirTemp("", "pressluft-site-create-*")
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("create temp dir: %v", err), Retryable: false}
	}
	defer os.RemoveAll(tmpDir)

	inventoryPath := filepath.Join(tmpDir, "inventory.ini")
	extraVarsPath := filepath.Join(tmpDir, "extra-vars.json")

	if err := os.WriteFile(inventoryPath, []byte(buildInventory(node)), 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write inventory: %v", err), Retryable: false}
	}

	extraVars, err := json.Marshal(vars)
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("marshal extra vars: %v", err), Retryable: false}
	}

	if err := os.WriteFile(extraVarsPath, extraVars, 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write extra vars: %v", err), Retryable: false}
	}

	args := []string{
		"-i", inventoryPath,
		"-e", "@" + extraVarsPath,
		"--ssh-extra-args=-o StrictHostKeyChecking=accept-new",
		siteCreatePlaybookPath,
	}

	output, err := e.runCommand(ctx, "ansible-playbook", args...)
	if err == nil {
		return nil
	}

	return mapAnsibleError(ctx, err, string(output))
}

// SiteCreateHandler handles site_create job execution.
type SiteCreateHandler struct {
	siteStore store.SiteStore
	nodeStore nodes.Store
	executor  SiteCreateExecutor
	logger    *log.Logger
	now       func() time.Time
}

// NewSiteCreateHandler creates a new handler for site_create jobs.
func NewSiteCreateHandler(siteStore store.SiteStore, nodeStore nodes.Store, executor SiteCreateExecutor, logger *log.Logger) *SiteCreateHandler {
	return &SiteCreateHandler{
		siteStore: siteStore,
		nodeStore: nodeStore,
		executor:  executor,
		logger:    logger,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

// Handle executes the site_create job.
func (h *SiteCreateHandler) Handle(ctx context.Context, job jobs.Job) error {
	if job.SiteID == nil || *job.SiteID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "site_create requires site_id", Retryable: false}
	}
	if job.EnvironmentID == nil || *job.EnvironmentID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "site_create requires environment_id", Retryable: false}
	}
	if job.NodeID == nil || *job.NodeID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "site_create requires node_id", Retryable: false}
	}

	site, err := h.siteStore.GetSiteByID(ctx, *job.SiteID)
	if err != nil {
		if errors.Is(err, store.ErrSiteNotFound) {
			return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "site not found for site_create job", Retryable: false}
		}
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	env, err := h.siteStore.GetEnvironmentByID(ctx, *job.EnvironmentID)
	if err != nil {
		if errors.Is(err, store.ErrEnvironmentNotFound) {
			return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "environment not found for site_create job", Retryable: false}
		}
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	node, err := h.nodeStore.GetByID(ctx, *job.NodeID)
	if err != nil {
		if errors.Is(err, nodes.ErrNotFound) {
			return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "node not found for site_create job", Retryable: false}
		}
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	h.logger.Printf("event=site_create stage=start site_id=%s environment_id=%s node_id=%s job_id=%s", site.ID, env.ID, node.ID, job.ID)

	vars := SiteCreateVars{
		SiteID:              site.ID,
		SiteSlug:            site.Slug,
		EnvironmentID:       env.ID,
		EnvironmentSlug:     env.Slug,
		EnvironmentType:     env.EnvironmentType,
		PreviewURL:          env.PreviewURL,
		FastCGICacheEnabled: env.FastCGICacheEnabled,
		RedisCacheEnabled:   env.RedisCacheEnabled,
	}

	if err := h.executor.RunSiteCreate(ctx, node, vars); err != nil {
		h.logger.Printf("event=site_create stage=failed site_id=%s environment_id=%s node_id=%s job_id=%s error=%v", site.ID, env.ID, node.ID, job.ID, err)
		return err
	}

	h.logger.Printf("event=site_create stage=succeeded site_id=%s environment_id=%s node_id=%s job_id=%s", site.ID, env.ID, node.ID, job.ID)
	return nil
}

func buildInventory(node nodes.Node) string {
	fields := []string{node.Hostname}
	if node.SSHPort > 0 {
		fields = append(fields, "ansible_port="+strconv.Itoa(node.SSHPort))
	}
	if node.SSHUser != "" {
		fields = append(fields, "ansible_user="+node.SSHUser)
	}
	if node.SSHPrivateKeyPath != "" {
		fields = append(fields, "ansible_ssh_private_key_file="+node.SSHPrivateKeyPath)
	}
	return "[target]\n" + strings.Join(fields, " ") + "\n"
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func mapAnsibleError(ctx context.Context, err error, output string) error {
	message := strings.TrimSpace(output)
	if message == "" {
		message = err.Error()
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return jobs.ExecutionError{Code: "ANSIBLE_TIMEOUT", Message: message, Retryable: true}
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: message, Retryable: false}
	}

	switch exitErr.ExitCode() {
	case 1:
		return jobs.ExecutionError{Code: "ANSIBLE_PLAY_ERROR", Message: message, Retryable: true}
	case 2:
		return jobs.ExecutionError{Code: "ANSIBLE_HOST_FAILED", Message: message, Retryable: true}
	case 4:
		return jobs.ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: message, Retryable: true}
	case 5:
		return jobs.ExecutionError{Code: "ANSIBLE_SYNTAX_ERROR", Message: message, Retryable: false}
	case 250:
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: message, Retryable: false}
	default:
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: message, Retryable: false}
	}
}
