package ssh

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrEnvironmentNotActive = errors.New("environment is not active")
var ErrNodeUnreachable = errors.New("node unreachable")
var ErrWPCliError = errors.New("wp-cli error")

const magicLoginNodeQueryTimeout = 10 * time.Second
const magicLoginTokenLifetime = 60 * time.Second

type Runner interface {
	Run(ctx context.Context, host string, port int, user string, remoteArgs ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, host string, port int, user string, remoteArgs ...string) (string, error) {
	args := []string{
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10",
		"-p", strconv.Itoa(port),
		fmt.Sprintf("%s@%s", user, host),
		"--",
	}
	args = append(args, remoteArgs...)

	cmd := exec.CommandContext(ctx, "ssh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("run ssh command: %w", err)
	}

	return string(output), nil
}

type Service struct {
	db     *sql.DB
	runner Runner
}

type MagicLoginResult struct {
	LoginURL  string `json:"login_url"`
	ExpiresAt string `json:"expires_at"`
}

type environmentQueryRef struct {
	EnvironmentID string
	Status        string
	NodeHostname  string
	SSHPort       int
	SSHUser       string
}

func NewService(db *sql.DB, runner Runner) *Service {
	return &Service{db: db, runner: runner}
}

func (s *Service) CreateMagicLogin(ctx context.Context, environmentID string) (MagicLoginResult, error) {
	environmentID = strings.TrimSpace(environmentID)
	if environmentID == "" {
		return MagicLoginResult{}, ErrInvalidInput
	}

	ref, err := s.loadEnvironmentForNodeQuery(ctx, environmentID)
	if err != nil {
		return MagicLoginResult{}, err
	}

	queryCtx, cancel := context.WithTimeout(ctx, magicLoginNodeQueryTimeout)
	defer cancel()

	remotePath := fmt.Sprintf("/var/www/sites/%s/current", ref.EnvironmentID)
	output, runErr := s.runner.Run(
		queryCtx,
		ref.NodeHostname,
		ref.SSHPort,
		ref.SSHUser,
		"wp",
		"--path="+remotePath,
		"user",
		"session",
		"create",
		"admin",
		"--porcelain",
	)
	if runErr != nil {
		return MagicLoginResult{}, classifyMagicLoginError(runErr, output)
	}

	loginURL := strings.TrimSpace(output)
	if loginURL == "" {
		return MagicLoginResult{}, ErrWPCliError
	}

	return MagicLoginResult{
		LoginURL:  loginURL,
		ExpiresAt: time.Now().UTC().Add(magicLoginTokenLifetime).Format(time.RFC3339),
	}, nil
}

func (s *Service) loadEnvironmentForNodeQuery(ctx context.Context, environmentID string) (environmentQueryRef, error) {
	var ref environmentQueryRef
	err := s.db.QueryRowContext(ctx, `
		SELECT e.id, e.status, n.hostname, n.ssh_port, n.ssh_user
		FROM environments e
		JOIN nodes n ON n.id = e.node_id
		WHERE e.id = ?
		LIMIT 1
	`, environmentID).Scan(&ref.EnvironmentID, &ref.Status, &ref.NodeHostname, &ref.SSHPort, &ref.SSHUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return environmentQueryRef{}, ErrEnvironmentNotFound
		}
		return environmentQueryRef{}, fmt.Errorf("query environment for magic login: %w", err)
	}

	if ref.Status != "active" {
		return environmentQueryRef{}, ErrEnvironmentNotActive
	}

	return ref, nil
}

func classifyMagicLoginError(runErr error, output string) error {
	if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(runErr, context.Canceled) {
		return ErrNodeUnreachable
	}

	lower := strings.ToLower(strings.TrimSpace(output + " " + runErr.Error()))
	if strings.Contains(lower, "connection refused") ||
		strings.Contains(lower, "connection timed out") ||
		strings.Contains(lower, "timed out") ||
		strings.Contains(lower, "no route to host") ||
		strings.Contains(lower, "could not resolve hostname") ||
		strings.Contains(lower, "connection closed") {
		return ErrNodeUnreachable
	}

	return ErrWPCliError
}
