package ssh

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

var (
	ErrNodeUnreachable = errors.New("node unreachable")
	ErrWPCLIError      = errors.New("wp-cli error")
	ErrQueryTimeout    = errors.New("query timeout")
)

const DefaultQueryTimeout = 10 * time.Second

type QueryRunner interface {
	WordPressVersion(ctx context.Context, host string, port int, user string, siteSlug string, envSlug string) (string, error)
	CheckNodePrerequisites(ctx context.Context, host string, port int, user string, isLocal bool) ([]string, error)
}

type Runner struct {
	timeout time.Duration
	execCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewRunner() *Runner {
	return &Runner{
		timeout: DefaultQueryTimeout,
		execCmd: exec.CommandContext,
	}
}

func (r *Runner) WordPressVersion(ctx context.Context, host string, port int, user string, siteSlug string, envSlug string) (string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	wpPath := fmt.Sprintf("/srv/www/%s/%s/current", siteSlug, envSlug)
	sshTarget := fmt.Sprintf("%s@%s", user, host)
	sshArgs := []string{
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(r.timeout.Seconds())),
		sshTarget,
		fmt.Sprintf("wp core version --path=%s", wpPath),
	}

	cmd := r.execCmd(queryCtx, "ssh", sshArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%w: %v", ErrQueryTimeout, err)
		}
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			outputStr := strings.TrimSpace(string(output))
			if strings.Contains(outputStr, "Connection refused") ||
				strings.Contains(outputStr, "Connection timed out") ||
				strings.Contains(outputStr, "No route to host") ||
				strings.Contains(outputStr, "Permission denied") {
				return "", fmt.Errorf("%w: %s", ErrNodeUnreachable, outputStr)
			}
			return "", fmt.Errorf("%w: %s", ErrWPCLIError, outputStr)
		}
		return "", fmt.Errorf("%w: %v", ErrNodeUnreachable, err)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("%w: empty version output", ErrWPCLIError)
	}

	return version, nil
}

func (r *Runner) CheckNodePrerequisites(ctx context.Context, host string, port int, user string, isLocal bool) ([]string, error) {
	_ = isLocal

	reasons := make([]string, 0, 3)
	if err := r.runRemoteCheck(ctx, host, port, user, "true"); err != nil {
		return []string{"node_unreachable"}, nil
	}
	if err := r.runRemoteCheck(ctx, host, port, user, "sudo -n true >/dev/null 2>&1"); err != nil {
		reasons = append(reasons, "sudo_unavailable")
	}
	if err := r.runRemoteCheck(ctx, host, port, user, "command -v wp >/dev/null 2>&1"); err != nil {
		reasons = append(reasons, "runtime_missing")
	}

	return reasons, nil
}
func (r *Runner) runRemoteCheck(ctx context.Context, host string, port int, user string, command string) error {
	checkCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	sshTarget := fmt.Sprintf("%s@%s", user, host)
	sshArgs := []string{
		"-p", fmt.Sprintf("%d", port),
		"-o", "StrictHostKeyChecking=no",
		"-o", "BatchMode=yes",
		"-o", fmt.Sprintf("ConnectTimeout=%d", int(r.timeout.Seconds())),
		sshTarget,
		command,
	}

	output, err := r.execCmd(checkCtx, "ssh", sshArgs...).CombinedOutput()
	if err != nil {
		if checkCtx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("%w: %v", ErrQueryTimeout, err)
		}
		outputStr := strings.TrimSpace(string(output))
		if strings.Contains(outputStr, "Connection refused") ||
			strings.Contains(outputStr, "Connection timed out") ||
			strings.Contains(outputStr, "No route to host") ||
			strings.Contains(outputStr, "Permission denied") {
			return fmt.Errorf("%w: %s", ErrNodeUnreachable, outputStr)
		}
		return fmt.Errorf("remote check failed: %s", outputStr)
	}

	return nil
}

type LocalRunner struct {
	timeout time.Duration
	execCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewLocalRunner() *LocalRunner {
	return &LocalRunner{
		timeout: DefaultQueryTimeout,
		execCmd: exec.CommandContext,
	}
}

func (r *LocalRunner) WordPressVersion(ctx context.Context, host string, port int, user string, siteSlug string, envSlug string) (string, error) {
	queryCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	wpPath := fmt.Sprintf("/srv/www/%s/%s/current", siteSlug, envSlug)
	cmd := r.execCmd(queryCtx, "wp", "core", "version", fmt.Sprintf("--path=%s", wpPath))
	output, err := cmd.CombinedOutput()
	if err != nil {
		if queryCtx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("%w: %v", ErrQueryTimeout, err)
		}
		outputStr := strings.TrimSpace(string(output))
		return "", fmt.Errorf("%w: %s", ErrWPCLIError, outputStr)
	}

	version := strings.TrimSpace(string(output))
	if version == "" {
		return "", fmt.Errorf("%w: empty version output", ErrWPCLIError)
	}

	return version, nil
}

func (r *LocalRunner) CheckNodePrerequisites(ctx context.Context, host string, port int, user string, isLocal bool) ([]string, error) {
	reasons := make([]string, 0, 2)

	checkCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if _, err := r.execCmd(checkCtx, "sh", "-lc", "sudo -n true >/dev/null 2>&1").CombinedOutput(); err != nil {
		if checkCtx.Err() == context.DeadlineExceeded {
			return []string{"node_unreachable"}, nil
		}
		reasons = append(reasons, "sudo_unavailable")
	}

	checkCtx, cancel = context.WithTimeout(ctx, r.timeout)
	defer cancel()
	if _, err := r.execCmd(checkCtx, "sh", "-lc", "command -v wp >/dev/null 2>&1").CombinedOutput(); err != nil {
		if checkCtx.Err() == context.DeadlineExceeded {
			return []string{"node_unreachable"}, nil
		}
		reasons = append(reasons, "runtime_missing")
	}

	return reasons, nil
}
