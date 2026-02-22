package hetzner

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hetznercloud/hcloud-go/hcloud"
)

var ErrCredentialMissing = errors.New("hetzner credential missing")
var ErrManagedKeyMissing = errors.New("managed ssh key missing")
var ErrActionTimeout = errors.New("hetzner action polling timeout")

type APIStatusError struct {
	StatusCode int
	Code       hcloud.ErrorCode
	Message    string
}

func (e APIStatusError) Error() string {
	message := strings.TrimSpace(e.Message)
	if message == "" {
		message = "hetzner api request failed"
	}
	if e.StatusCode > 0 {
		if e.Code != "" {
			return fmt.Sprintf("hetzner api status %d (%s): %s", e.StatusCode, e.Code, message)
		}
		return fmt.Sprintf("hetzner api status %d: %s", e.StatusCode, message)
	}
	if e.Code != "" {
		return fmt.Sprintf("hetzner api error (%s): %s", e.Code, message)
	}
	return message
}

type AcquireInput struct {
	Token string
	Name  string
}

type AcquireTarget struct {
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
	ServerID          int
	ActionID          int
}

type Acquirer struct {
	baseURL        string
	httpClient     *http.Client
	managedKeyName string
	privateKeyPath string
	publicKeyPath  string
	pollInterval   time.Duration
	pollTimeout    time.Duration
}

func NewAcquirer() *Acquirer {
	homeDir, _ := os.UserHomeDir()
	baseKeyPath := filepath.Join(homeDir, ".ssh", "pressluft-provider-ed25519")
	return &Acquirer{
		baseURL:        hcloud.Endpoint,
		httpClient:     &http.Client{Timeout: 15 * time.Second},
		managedKeyName: "pressluft-managed",
		privateKeyPath: baseKeyPath,
		publicKeyPath:  baseKeyPath + ".pub",
		pollInterval:   2 * time.Second,
		pollTimeout:    2 * time.Minute,
	}
}

func (a *Acquirer) Acquire(ctx context.Context, input AcquireInput) (AcquireTarget, error) {
	token := strings.TrimSpace(input.Token)
	if token == "" {
		return AcquireTarget{}, ErrCredentialMissing
	}

	keyData, err := os.ReadFile(a.publicKeyPath)
	if err != nil {
		return AcquireTarget{}, fmt.Errorf("%w: read %s: %v", ErrManagedKeyMissing, a.publicKeyPath, err)
	}

	client := hcloud.NewClient(a.clientOptions(token)...)

	sshKeyID, err := a.ensureSSHKey(ctx, client, strings.TrimSpace(string(keyData)))
	if err != nil {
		return AcquireTarget{}, err
	}

	serverID, actionID, err := a.createServer(ctx, client, strings.TrimSpace(input.Name), sshKeyID)
	if err != nil {
		return AcquireTarget{}, err
	}

	if err := a.waitForAction(ctx, client, actionID); err != nil {
		return AcquireTarget{}, err
	}

	hostname, publicIP, err := a.fetchServerTarget(ctx, client, serverID)
	if err != nil {
		return AcquireTarget{}, err
	}

	return AcquireTarget{
		Hostname:          hostname,
		PublicIP:          publicIP,
		SSHPort:           22,
		SSHUser:           "root",
		SSHPrivateKeyPath: a.privateKeyPath,
		ServerID:          serverID,
		ActionID:          actionID,
	}, nil
}

func (a *Acquirer) clientOptions(token string) []hcloud.ClientOption {
	options := []hcloud.ClientOption{
		hcloud.WithToken(token),
		hcloud.WithPollInterval(a.pollInterval),
	}
	if strings.TrimSpace(a.baseURL) != "" {
		options = append(options, hcloud.WithEndpoint(a.baseURL))
	}
	if a.httpClient != nil {
		options = append(options, hcloud.WithHTTPClient(a.httpClient))
	}
	return options
}

func (a *Acquirer) ensureSSHKey(ctx context.Context, client *hcloud.Client, publicKey string) (int, error) {
	sshKey, _, err := client.SSHKey.GetByName(ctx, a.managedKeyName)
	if err == nil && sshKey != nil {
		return sshKey.ID, nil
	}
	if err != nil {
		var apiErr hcloud.Error
		if !errors.As(err, &apiErr) || apiErr.Code != hcloud.ErrorCodeNotFound {
			return 0, mapAPIError(err)
		}
	}

	created, _, err := client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{Name: a.managedKeyName, PublicKey: publicKey})
	if err != nil {
		return 0, mapAPIError(err)
	}
	if created == nil || created.ID <= 0 {
		return 0, fmt.Errorf("decode ssh key create: missing id")
	}
	return created.ID, nil
}

func (a *Acquirer) createServer(ctx context.Context, client *hcloud.Client, name string, sshKeyID int) (int, int, error) {
	if strings.TrimSpace(name) == "" {
		name = "pressluft-node"
	}
	createResult, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       name,
		ServerType: &hcloud.ServerType{Name: "cpx11"},
		Image:      &hcloud.Image{Name: "ubuntu-24.04"},
		SSHKeys:    []*hcloud.SSHKey{{ID: sshKeyID}},
	})
	if err != nil {
		return 0, 0, mapAPIError(err)
	}
	if createResult.Server == nil || createResult.Server.ID <= 0 || createResult.Action == nil || createResult.Action.ID <= 0 {
		return 0, 0, fmt.Errorf("decode server create: missing ids")
	}
	return createResult.Server.ID, createResult.Action.ID, nil
}

func (a *Acquirer) waitForAction(ctx context.Context, client *hcloud.Client, actionID int) error {
	if actionID <= 0 {
		return fmt.Errorf("decode action: missing action id")
	}

	deadline := time.Now().Add(a.pollTimeout)
	for {
		if time.Now().After(deadline) {
			return ErrActionTimeout
		}

		action, _, err := client.Action.GetByID(ctx, actionID)
		if err != nil {
			return mapAPIError(err)
		}
		if action == nil {
			return fmt.Errorf("decode action: missing action")
		}

		switch action.Status {
		case hcloud.ActionStatusSuccess:
			return nil
		case hcloud.ActionStatusRunning:
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(a.pollInterval):
			}
		case hcloud.ActionStatusError:
			if strings.TrimSpace(action.ErrorMessage) != "" {
				return fmt.Errorf("hetzner action failed: %s", strings.TrimSpace(action.ErrorMessage))
			}
			return fmt.Errorf("hetzner action failed")
		default:
			return fmt.Errorf("hetzner action unexpected status: %s", action.Status)
		}
	}
}

func (a *Acquirer) fetchServerTarget(ctx context.Context, client *hcloud.Client, serverID int) (string, string, error) {
	server, _, err := client.Server.GetByID(ctx, serverID)
	if err != nil {
		return "", "", mapAPIError(err)
	}
	if server == nil {
		return "", "", fmt.Errorf("decode server: missing server")
	}
	if server.PublicNet.IPv4.IsUnspecified() {
		return "", "", fmt.Errorf("decode server: missing ipv4")
	}
	host := strings.TrimSpace(server.PublicNet.IPv4.IP.String())
	if host == "" {
		return "", "", fmt.Errorf("decode server: missing ipv4")
	}
	return host, host, nil
}

func mapAPIError(err error) error {
	if err == nil {
		return nil
	}

	var apiErr hcloud.Error
	if errors.As(err, &apiErr) {
		statusCode := 0
		if response := apiErr.Response(); response != nil && response.Response != nil {
			statusCode = response.StatusCode
		}
		return APIStatusError{
			StatusCode: statusCode,
			Code:       apiErr.Code,
			Message:    strings.TrimSpace(apiErr.Message),
		}
	}

	return err
}
