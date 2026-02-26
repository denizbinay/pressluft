package ansible

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	if err := a.runCommand(ctx, syntaxCmd, sink, "syntax_check", "ansible syntax check"); err != nil {
		return err
	}

	if req.CheckOnly {
		if sink != nil {
			_ = sink.Emit(ctx, runner.Event{Type: "runner_complete", Level: "info", Message: "check-only run complete"})
		}
		return nil
	}

	if sink != nil {
		_ = sink.Emit(ctx, runner.Event{Type: "runner_apply", Level: "info", Message: "running ansible playbook"})
	}

	applyCmd := a.buildApplyCommand(ctx, req)
	if err := a.runCommand(ctx, applyCmd, sink, "apply", "ansible playbook run"); err != nil {
		return err
	}

	if sink != nil {
		_ = sink.Emit(ctx, runner.Event{Type: "runner_complete", Level: "info", Message: "ansible run complete"})
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
	a.configureCommand(cmd)
	return cmd
}

func (a *Adapter) buildApplyCommand(ctx context.Context, req runner.Request) *exec.Cmd {
	args := []string{
		"-i", req.InventoryPath,
		filepath.Clean(req.PlaybookPath),
	}
	args = append(args, flattenExtraVars(req.ExtraVars)...)

	cmd := exec.CommandContext(ctx, a.binaryPath, args...)
	a.configureCommand(cmd)
	return cmd
}

func (a *Adapter) configureCommand(cmd *exec.Cmd) {
	cmd.Dir = a.workingDirectory
	env := os.Environ()
	if binDir := filepath.Dir(a.binaryPath); binDir != "" {
		env = prependEnvPath(env, binDir)
		venvDir := filepath.Dir(binDir)
		if filepath.Base(binDir) == "bin" && filepath.Base(venvDir) == ".venv" {
			env = upsertEnv(env, "VIRTUAL_ENV", venvDir)
		}
	}
	env = upsertEnv(env, "ANSIBLE_STDOUT_CALLBACK", "json")
	cmd.Env = upsertEnv(env, "ANSIBLE_ROLES_PATH", "ops/ansible/roles")
}

func (a *Adapter) runCommand(ctx context.Context, cmd *exec.Cmd, sink runner.EventSink, stepKey, description string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if sink != nil {
		_ = sink.Emit(ctx, runner.Event{Type: "runner_step", Level: "info", StepKey: stepKey, Message: description})
		sanitizedArgs := redactCommandArgs(cmd.Args)
		_ = sink.Emit(ctx, runner.Event{Type: "runner_args", Level: "info", StepKey: stepKey, Message: "ansible args", Payload: strings.Join(sanitizedArgs, " ")})
	}

	err := cmd.Run()
	stdoutText := strings.TrimSpace(stdout.String())
	stderrText := strings.TrimSpace(stderr.String())
	if sink != nil {
		if stdoutText != "" {
			_ = sink.Emit(ctx, runner.Event{Type: "runner_output", Level: "info", StepKey: stepKey, Message: "stdout", Payload: stdoutText})
		}
		if stderrText != "" {
			level := "warning"
			if err != nil {
				level = "error"
			}
			_ = sink.Emit(ctx, runner.Event{Type: "runner_output", Level: level, StepKey: stepKey, Message: "stderr", Payload: stderrText})
		}
	}

	if err != nil {
		return fmt.Errorf("%s failed: %w (stdout=%s stderr=%s)", description, err, stdoutText, stderrText)
	}
	return nil
}

func upsertEnv(env []string, key, value string) []string {
	if key == "" {
		return env
	}
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

func prependEnvPath(env []string, dir string) []string {
	if dir == "" {
		return env
	}
	prefix := "PATH="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			current := strings.TrimPrefix(entry, prefix)
			if current == "" {
				env[i] = prefix + dir
			} else {
				env[i] = prefix + dir + string(os.PathListSeparator) + current
			}
			return env
		}
	}
	return append(env, prefix+dir)
}

func flattenExtraVars(extraVars map[string]string) []string {
	if len(extraVars) == 0 {
		return nil
	}

	payload, _ := json.Marshal(extraVars)
	return []string{"--extra-vars", string(payload)}
}

func redactCommandArgs(args []string) []string {
	if len(args) == 0 {
		return args
	}
	redacted := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--extra-vars" && i+1 < len(args) {
			redacted = append(redacted, arg, redactExtraVarsArg(args[i+1]))
			i++
			continue
		}
		redacted = append(redacted, arg)
	}
	return redacted
}

func redactExtraVarsArg(arg string) string {
	var payload map[string]any
	if err := json.Unmarshal([]byte(arg), &payload); err != nil {
		return arg
	}
	for key := range payload {
		lower := strings.ToLower(key)
		if strings.Contains(lower, "token") || strings.Contains(lower, "password") || strings.Contains(lower, "secret") {
			payload[key] = "<redacted>"
		}
	}
	redacted, err := json.Marshal(payload)
	if err != nil {
		return arg
	}
	return string(redacted)
}
