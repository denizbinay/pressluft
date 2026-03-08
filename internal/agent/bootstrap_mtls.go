//go:build !dev

package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

func (a *Agent) bootstrap(ctx context.Context) error {
	state := a.config.CertificateState(time.Now())
	serverID := a.config.ServerID
	switch state.Status {
	case CertificateValid:
		if strings.TrimSpace(a.config.ResolveRegistrationToken()) != "" {
			if err := a.config.ClearRegistrationToken(""); err != nil {
				a.logger.Warn("agent registration token cleanup failed", "server_id", serverID, "error", err)
			}
		}
		return nil
	case CertificateExpiringSoon:
		if strings.TrimSpace(a.config.ResolveRegistrationToken()) == "" {
			a.logger.Warn("agent certificate nearing expiry without reissue token", "server_id", serverID, "expires_at", state.Leaf.NotAfter.UTC().Format(time.RFC3339), "reissue_window", CertificateReissueWindow)
			return nil
		}
		a.logger.Info("agent certificate reissue started", "server_id", serverID, "expires_at", state.Leaf.NotAfter.UTC().Format(time.RFC3339))
	case CertificateMissing:
		if strings.TrimSpace(a.config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("agent bootstrap requires a registration token when no client certificate is present")
		}
		a.logger.Info("agent registration started", "server_id", serverID, "registration_reason", "certificate_missing")
	case CertificateExpired:
		if strings.TrimSpace(a.config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("client certificate expired at %s and no registration token is available for reissue", state.Leaf.NotAfter.UTC().Format(time.RFC3339))
		}
		a.logger.Info("agent registration started", "server_id", serverID, "registration_reason", "certificate_expired", "expired_at", state.Leaf.NotAfter.UTC().Format(time.RFC3339))
	case CertificateInvalid:
		if strings.TrimSpace(a.config.ResolveRegistrationToken()) == "" {
			return fmt.Errorf("client certificate is unusable: %w", state.Underlying)
		}
		a.logger.Warn("agent registration started", "server_id", serverID, "registration_reason", "certificate_invalid", "error", state.Underlying)
	}

	if err := Register(a.config, ""); err != nil {
		if errors.Is(err, ErrExistingValidCertificate) {
			return nil
		}
		a.logger.Error("agent registration failed", "server_id", serverID, "error", err)
		return fmt.Errorf("registration failed: %w", err)
	}
	if ctx.Err() == nil {
		a.logger.Info("agent registration completed", "server_id", serverID)
	}
	return ctx.Err()
}
