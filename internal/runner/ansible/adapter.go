package ansible

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"pressluft/internal/runner"
)

// Adapter executes Ansible playbooks with strict command guardrails.
type Adapter struct {
	binaryPath       string
	workingDirectory string
	allowedPlaybooks map[string]struct{}
}

func NewAdapter(binaryPath, workingDirectory string, allowedPlaybooks []string) *Adapter {
	allow := make(map[string]struct{}, len(allowedPlaybooks))
	for _, p := range allowedPlaybooks {
		allow[filepath.Clean(p)] = struct{}{}
	}
	return &Adapter{
		binaryPath:       strings.TrimSpace(binaryPath),
		workingDirectory: strings.TrimSpace(workingDirectory),
		allowedPlaybooks: allow,
	}
}

func (a *Adapter) Name() string {
	return "ansible"
}

func (a *Adapter) Run(ctx context.Context, req runner.Request, sink runner.EventSink) error {
	if err := a.validateRequest(req); err != nil {
		return err
	}

	if sink != nil {
		_ = sink.Emit(ctx, runner.Event{Type: "runner_preflight", Level: "info", Message: "running ansible syntax check"})
	}

	syntaxCmd := a.buildSyntaxCheckCommand(ctx, req)
	if output, err := syntaxCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("ansible syntax check failed: %w (%s)", err, strings.TrimSpace(string(output)))
	}

	if req.CheckOnly {
		if sink != nil {
			_ = sink.Emit(ctx, runner.Event{Type: "runner_complete", Level: "info", Message: "check-only run complete"})
		}
		return nil
	}

	if sink != nil {
		_ = sink.Emit(ctx, runner.Event{Type: "runner_pending", Level: "warning", Message: "apply execution not yet enabled"})
	}
	return nil
}

func (a *Adapter) validateRequest(req runner.Request) error {
	if strings.TrimSpace(a.binaryPath) == "" {
		return fmt.Errorf("ansible binary path is required")
	}
	if strings.TrimSpace(a.workingDirectory) == "" {
		return fmt.Errorf("ansible working directory is required")
	}
	if strings.TrimSpace(req.InventoryPath) == "" {
		return fmt.Errorf("inventory path is required")
	}
	playbook := filepath.Clean(req.PlaybookPath)
	if playbook == "" {
		return fmt.Errorf("playbook path is required")
	}
	if _, ok := a.allowedPlaybooks[playbook]; !ok {
		return fmt.Errorf("playbook %q is not allowlisted", req.PlaybookPath)
	}
	return nil
}

func (a *Adapter) buildSyntaxCheckCommand(ctx context.Context, req runner.Request) *exec.Cmd {
	args := []string{
		"-i", req.InventoryPath,
		filepath.Clean(req.PlaybookPath),
		"--syntax-check",
	}
	args = append(args, flattenExtraVars(req.ExtraVars)...)

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	cmd.Dir = a.workingDirectory
	return cmd
}

func flattenExtraVars(extraVars map[string]string) []string {
	if len(extraVars) == 0 {
		return nil
	}

	keys := make([]string, 0, len(extraVars))
	for key := range extraVars {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	args := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		args = append(args, "--extra-vars", fmt.Sprintf("%s=%s", key, extraVars[key]))
	}
	return args
}
