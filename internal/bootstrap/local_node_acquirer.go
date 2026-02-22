package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var ErrLocalAcquisitionUnavailable = errors.New("local acquisition unavailable")

type LocalNodeTarget struct {
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
}

type LocalNodeAcquirer interface {
	Acquire(ctx context.Context) (LocalNodeTarget, error)
}

type commandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)
type lookPathFunc func(file string) (string, error)

const (
	defaultMultipassInstanceName = "pressluft-local-node"
	defaultMultipassImage        = "24.04"
)

func NewLocalNodeAcquirer() LocalNodeAcquirer {
	return &AutoLocalNodeAcquirer{
		env:       NewEnvLocalNodeAcquirer(),
		multipass: NewMultipassLocalNodeAcquirer(),
	}
}

type AutoLocalNodeAcquirer struct {
	env       *EnvLocalNodeAcquirer
	multipass *MultipassLocalNodeAcquirer
}

func (a *AutoLocalNodeAcquirer) Acquire(ctx context.Context) (LocalNodeTarget, error) {
	if strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_HOST")) != "" {
		return a.env.Acquire(ctx)
	}
	return a.multipass.Acquire(ctx)
}

type EnvLocalNodeAcquirer struct{}

func NewEnvLocalNodeAcquirer() *EnvLocalNodeAcquirer {
	return &EnvLocalNodeAcquirer{}
}

func (a *EnvLocalNodeAcquirer) Acquire(_ context.Context) (LocalNodeTarget, error) {
	host := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_HOST"))
	if host == "" {
		return LocalNodeTarget{}, fmt.Errorf("%w: set PRESSLUFT_LOCAL_NODE_HOST", ErrLocalAcquisitionUnavailable)
	}

	publicIP := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_PUBLIC_IP"))
	if publicIP == "" {
		publicIP = host
	}

	sshUser := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_SSH_USER"))
	if sshUser == "" {
		sshUser = "ubuntu"
	}

	sshPort := 22
	if rawPort := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_SSH_PORT")); rawPort != "" {
		parsed, err := strconv.Atoi(rawPort)
		if err != nil || parsed < 1 || parsed > 65535 {
			return LocalNodeTarget{}, fmt.Errorf("%w: PRESSLUFT_LOCAL_NODE_SSH_PORT must be 1-65535", ErrLocalAcquisitionUnavailable)
		}
		sshPort = parsed
	}

	return LocalNodeTarget{
		Hostname:          host,
		PublicIP:          publicIP,
		SSHPort:           sshPort,
		SSHUser:           sshUser,
		SSHPrivateKeyPath: strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_SSH_PRIVATE_KEY_PATH")),
	}, nil
}

type MultipassLocalNodeAcquirer struct {
	run      commandRunner
	lookPath lookPathFunc
}

func NewMultipassLocalNodeAcquirer() *MultipassLocalNodeAcquirer {
	return &MultipassLocalNodeAcquirer{run: runCommand, lookPath: exec.LookPath}
}

func (a *MultipassLocalNodeAcquirer) Acquire(ctx context.Context) (LocalNodeTarget, error) {
	if _, err := a.lookPath("multipass"); err != nil {
		return LocalNodeTarget{}, fmt.Errorf("%w: multipass CLI not found in PATH", ErrLocalAcquisitionUnavailable)
	}

	instanceName := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_NAME"))
	if instanceName == "" {
		instanceName = defaultMultipassInstanceName
	}

	if err := a.ensureInstance(ctx, instanceName); err != nil {
		return LocalNodeTarget{}, err
	}

	instance, err := a.instanceInfo(ctx, instanceName)
	if err != nil {
		return LocalNodeTarget{}, err
	}

	if len(instance.IPv4) == 0 || strings.TrimSpace(instance.IPv4[0]) == "" {
		return LocalNodeTarget{}, fmt.Errorf("%w: multipass instance %q has no IPv4 address", ErrLocalAcquisitionUnavailable, instanceName)
	}

	privateKeyPath, err := resolveManagedKeyPath()
	if err != nil {
		return LocalNodeTarget{}, err
	}
	if err := a.ensureManagedKeyPair(ctx, privateKeyPath); err != nil {
		return LocalNodeTarget{}, err
	}
	publicKey, err := readPublicKey(privateKeyPath + ".pub")
	if err != nil {
		return LocalNodeTarget{}, err
	}
	if err := a.injectAuthorizedKey(ctx, instanceName, publicKey); err != nil {
		return LocalNodeTarget{}, err
	}

	return LocalNodeTarget{
		Hostname:          strings.TrimSpace(instance.IPv4[0]),
		PublicIP:          strings.TrimSpace(instance.IPv4[0]),
		SSHPort:           22,
		SSHUser:           "ubuntu",
		SSHPrivateKeyPath: privateKeyPath,
	}, nil
}

func resolveManagedKeyPath() (string, error) {
	privateKeyPath := strings.TrimSpace(os.Getenv("PRESSLUFT_LOCAL_NODE_SSH_PRIVATE_KEY_PATH"))
	if privateKeyPath != "" {
		return privateKeyPath, nil
	}
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("%w: resolve user config dir: %v", ErrLocalAcquisitionUnavailable, err)
	}
	return filepath.Join(configDir, "pressluft", "keys", "local-node-ed25519"), nil
}

func (a *MultipassLocalNodeAcquirer) ensureManagedKeyPair(ctx context.Context, privateKeyPath string) error {
	if privateKeyPath == "" {
		return fmt.Errorf("%w: local SSH private key path is empty", ErrLocalAcquisitionUnavailable)
	}

	if _, err := os.Stat(privateKeyPath); err == nil {
		if _, pubErr := os.Stat(privateKeyPath + ".pub"); pubErr == nil {
			return nil
		}
	}

	if err := os.MkdirAll(filepath.Dir(privateKeyPath), 0o700); err != nil {
		return fmt.Errorf("%w: create key directory: %v", ErrLocalAcquisitionUnavailable, err)
	}

	output, err := a.run(ctx, "ssh-keygen", "-t", "ed25519", "-N", "", "-f", privateKeyPath, "-q")
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%w: ssh-keygen failed: %s", ErrLocalAcquisitionUnavailable, message)
	}

	if err := os.Chmod(privateKeyPath, 0o600); err != nil {
		return fmt.Errorf("%w: set private key permissions: %v", ErrLocalAcquisitionUnavailable, err)
	}

	return nil
}

func readPublicKey(publicKeyPath string) (string, error) {
	content, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return "", fmt.Errorf("%w: read public key: %v", ErrLocalAcquisitionUnavailable, err)
	}
	key := strings.TrimSpace(string(content))
	if key == "" {
		return "", fmt.Errorf("%w: empty public key at %s", ErrLocalAcquisitionUnavailable, publicKeyPath)
	}
	return key, nil
}

func (a *MultipassLocalNodeAcquirer) injectAuthorizedKey(ctx context.Context, instanceName string, publicKey string) error {
	escapedPublicKey := strings.ReplaceAll(publicKey, "'", "'\"'\"'")
	remoteCommand := "mkdir -p ~/.ssh && chmod 700 ~/.ssh && touch ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys && grep -qxF '" + escapedPublicKey + "' ~/.ssh/authorized_keys || printf '%s\\n' '" + escapedPublicKey + "' >> ~/.ssh/authorized_keys"

	output, err := a.run(ctx, "multipass", "exec", instanceName, "--", "sh", "-lc", remoteCommand)
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("%w: inject SSH key into local node: %s", ErrLocalAcquisitionUnavailable, message)
	}

	return nil
}

type multipassInfoResult struct {
	Errors    []string                   `json:"errors"`
	Info      map[string]multipassRecord `json:"info"`
	ErrorText string                     `json:"error"`
}

type multipassRecord struct {
	State string   `json:"state"`
	IPv4  []string `json:"ipv4"`
}

func (a *MultipassLocalNodeAcquirer) ensureInstance(ctx context.Context, name string) error {
	instance, err := a.instanceInfo(ctx, name)
	if err != nil {
		if !errors.Is(err, ErrLocalAcquisitionUnavailable) {
			return err
		}
		launchOutput, launchErr := a.run(ctx, "multipass", "launch", defaultMultipassImage, "--name", name)
		if launchErr != nil {
			message := strings.TrimSpace(string(launchOutput))
			if message == "" {
				message = launchErr.Error()
			}
			return fmt.Errorf("%w: multipass launch failed: %s", ErrLocalAcquisitionUnavailable, message)
		}
		return nil
	}

	if !strings.EqualFold(strings.TrimSpace(instance.State), "Running") {
		startOutput, startErr := a.run(ctx, "multipass", "start", name)
		if startErr != nil {
			message := strings.TrimSpace(string(startOutput))
			if message == "" {
				message = startErr.Error()
			}
			return fmt.Errorf("%w: multipass start failed: %s", ErrLocalAcquisitionUnavailable, message)
		}
	}

	return nil
}

func (a *MultipassLocalNodeAcquirer) instanceInfo(ctx context.Context, name string) (multipassRecord, error) {
	output, err := a.run(ctx, "multipass", "info", name, "--format", "json")
	if err != nil {
		message := strings.TrimSpace(string(output))
		if message == "" {
			message = err.Error()
		}
		return multipassRecord{}, fmt.Errorf("%w: multipass info failed: %s", ErrLocalAcquisitionUnavailable, message)
	}

	var payload multipassInfoResult
	if err := json.Unmarshal(output, &payload); err != nil {
		return multipassRecord{}, fmt.Errorf("%w: parse multipass info JSON: %v", ErrLocalAcquisitionUnavailable, err)
	}

	if payload.ErrorText != "" {
		return multipassRecord{}, fmt.Errorf("%w: %s", ErrLocalAcquisitionUnavailable, strings.TrimSpace(payload.ErrorText))
	}
	if len(payload.Errors) > 0 {
		return multipassRecord{}, fmt.Errorf("%w: %s", ErrLocalAcquisitionUnavailable, strings.Join(payload.Errors, "; "))
	}

	instance, ok := payload.Info[name]
	if !ok {
		return multipassRecord{}, fmt.Errorf("%w: multipass instance %q not found", ErrLocalAcquisitionUnavailable, name)
	}

	return instance, nil
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
