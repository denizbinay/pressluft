package agent

import (
	"fmt"
	"net/url"
	"path"
	"strings"
)

func (c *Config) registrationURL() (string, error) {
	base, err := c.controlPlaneURL()
	if err != nil {
		return "", err
	}
	if base.Scheme != "https" {
		return "", fmt.Errorf("production agent registration requires an https control_plane URL")
	}
	base.Path = path.Join(base.Path, fmt.Sprintf("/api/nodes/%d/register", c.ServerID))
	return base.String(), nil
}

func (c *Config) websocketURL() (string, error) {
	base, err := c.controlPlaneURL()
	if err != nil {
		return "", err
	}
	switch base.Scheme {
	case "http":
		base.Scheme = "ws"
	case "https":
		base.Scheme = "wss"
	default:
		return "", fmt.Errorf("unsupported control plane scheme %q", base.Scheme)
	}
	base.Path = path.Join(base.Path, "/ws/agent")
	return base.String(), nil
}

func (c *Config) controlPlaneURL() (*url.URL, error) {
	raw := strings.TrimSpace(c.ControlPlane)
	if raw == "" {
		return nil, fmt.Errorf("control_plane is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse control_plane: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("control_plane must be an absolute URL")
	}
	return parsed, nil
}
