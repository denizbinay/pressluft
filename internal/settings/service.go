package settings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"pressluft/internal/store"
)

var ErrInvalidInput = errors.New("invalid input")

var providerPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

const keyControlPlaneDomain = "control_plane_domain"
const keyPreviewDomain = "preview_domain"
const keyDNS01Provider = "dns01_provider"
const keyDNS01CredentialsJSON = "dns01_credentials_json"

type SecretStore interface {
	Put(ctx context.Context, name string, plaintext []byte) (string, error)
	Delete(ctx context.Context, reference string) error
}

type DomainConfig struct {
	ControlPlaneDomain     *string `json:"control_plane_domain"`
	PreviewDomain          *string `json:"preview_domain"`
	DNS01Provider          *string `json:"dns01_provider"`
	DNS01CredentialsJSON   any     `json:"dns01_credentials_json"`
	DNS01CredentialsExists bool    `json:"dns01_credentials_configured"`
}

type UpdateDomainConfigInput struct {
	ControlPlaneDomain   *string
	PreviewDomain        *string
	DNS01Provider        *string
	DNS01CredentialsJSON map[string]string
}

type Service struct {
	db      *sql.DB
	secrets SecretStore
}

func NewService(db *sql.DB, secrets SecretStore) *Service {
	return &Service{db: db, secrets: secrets}
}

func (s *Service) GetDomainConfig(ctx context.Context) (DomainConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT key, value
		FROM settings
		WHERE key IN (?, ?, ?, ?)
	`, keyControlPlaneDomain, keyPreviewDomain, keyDNS01Provider, keyDNS01CredentialsJSON)
	if err != nil {
		return DomainConfig{}, fmt.Errorf("query settings: %w", err)
	}
	defer rows.Close()

	cfg := DomainConfig{DNS01CredentialsJSON: nil}
	for rows.Next() {
		var key string
		var value string
		if err := rows.Scan(&key, &value); err != nil {
			return DomainConfig{}, fmt.Errorf("scan settings row: %w", err)
		}

		trimmed := strings.TrimSpace(value)
		switch key {
		case keyControlPlaneDomain:
			cfg.ControlPlaneDomain = optionalString(trimmed)
		case keyPreviewDomain:
			cfg.PreviewDomain = optionalString(trimmed)
		case keyDNS01Provider:
			cfg.DNS01Provider = optionalString(trimmed)
		case keyDNS01CredentialsJSON:
			if trimmed != "" {
				cfg.DNS01CredentialsExists = true
			}
		}
	}

	if err := rows.Err(); err != nil {
		return DomainConfig{}, fmt.Errorf("iterate settings rows: %w", err)
	}

	return cfg, nil
}

func (s *Service) UpdateDomainConfig(ctx context.Context, input UpdateDomainConfigInput) (DomainConfig, error) {
	normalized, err := normalizeAndValidate(input)
	if err != nil {
		return DomainConfig{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	credentialsRef := ""
	if normalized.DNS01CredentialsJSON != nil {
		payload, err := json.Marshal(normalized.DNS01CredentialsJSON)
		if err != nil {
			return DomainConfig{}, fmt.Errorf("marshal dns credentials json: %w", err)
		}
		ref, err := s.secrets.Put(ctx, keyDNS01CredentialsJSON, payload)
		if err != nil {
			return DomainConfig{}, fmt.Errorf("persist dns credentials secret: %w", err)
		}
		credentialsRef = ref
	}

	var previousRef string
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if err := tx.QueryRowContext(ctx, `
			SELECT value
			FROM settings
			WHERE key = ?
			LIMIT 1
		`, keyDNS01CredentialsJSON).Scan(&previousRef); err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("load existing dns credentials reference: %w", err)
		}

		if err := upsertOrDeleteSetting(ctx, tx, keyControlPlaneDomain, pointerToValue(normalized.ControlPlaneDomain), now); err != nil {
			return err
		}
		if err := upsertOrDeleteSetting(ctx, tx, keyPreviewDomain, pointerToValue(normalized.PreviewDomain), now); err != nil {
			return err
		}
		if err := upsertOrDeleteSetting(ctx, tx, keyDNS01Provider, pointerToValue(normalized.DNS01Provider), now); err != nil {
			return err
		}

		if normalized.DNS01CredentialsJSON == nil {
			if err := deleteSetting(ctx, tx, keyDNS01CredentialsJSON); err != nil {
				return err
			}
		} else {
			if err := upsertOrDeleteSetting(ctx, tx, keyDNS01CredentialsJSON, credentialsRef, now); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return DomainConfig{}, err
	}

	if previousRef != "" && previousRef != credentialsRef {
		if err := s.secrets.Delete(ctx, previousRef); err != nil {
			return DomainConfig{}, fmt.Errorf("delete previous dns credentials secret: %w", err)
		}
	}

	return s.GetDomainConfig(ctx)
}

func normalizeAndValidate(input UpdateDomainConfigInput) (UpdateDomainConfigInput, error) {
	controlPlaneDomain := normalizeOptional(input.ControlPlaneDomain)
	previewDomain := normalizeOptional(input.PreviewDomain)
	dns01Provider := normalizeOptional(input.DNS01Provider)

	if controlPlaneDomain != nil && !isValidHostname(*controlPlaneDomain) {
		return UpdateDomainConfigInput{}, ErrInvalidInput
	}

	if previewDomain != nil && !isValidHostname(*previewDomain) {
		return UpdateDomainConfigInput{}, ErrInvalidInput
	}

	if previewDomain == nil {
		if dns01Provider != nil || input.DNS01CredentialsJSON != nil {
			return UpdateDomainConfigInput{}, ErrInvalidInput
		}
		return UpdateDomainConfigInput{ControlPlaneDomain: controlPlaneDomain}, nil
	}

	if dns01Provider == nil || !providerPattern.MatchString(*dns01Provider) {
		return UpdateDomainConfigInput{}, ErrInvalidInput
	}

	if len(input.DNS01CredentialsJSON) == 0 {
		return UpdateDomainConfigInput{}, ErrInvalidInput
	}

	normalizedCredentials := make(map[string]string, len(input.DNS01CredentialsJSON))
	for key, value := range input.DNS01CredentialsJSON {
		normalizedKey := strings.TrimSpace(key)
		normalizedValue := strings.TrimSpace(value)
		if normalizedKey == "" || normalizedValue == "" {
			return UpdateDomainConfigInput{}, ErrInvalidInput
		}
		normalizedCredentials[normalizedKey] = normalizedValue
	}

	return UpdateDomainConfigInput{
		ControlPlaneDomain:   controlPlaneDomain,
		PreviewDomain:        previewDomain,
		DNS01Provider:        dns01Provider,
		DNS01CredentialsJSON: normalizedCredentials,
	}, nil
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	copy := value
	return &copy
}

func normalizeOptional(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.ToLower(strings.TrimSpace(*value))
	if trimmed == "" {
		return nil
	}
	copy := trimmed
	return &copy
}

func pointerToValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func upsertOrDeleteSetting(ctx context.Context, tx *sql.Tx, key, value, now string) error {
	if strings.TrimSpace(value) == "" {
		return deleteSetting(ctx, tx, key)
	}

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO settings (key, value, updated_at)
		VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at
	`, key, value, now); err != nil {
		return fmt.Errorf("upsert setting %s: %w", key, err)
	}

	return nil
}

func deleteSetting(ctx context.Context, tx *sql.Tx, key string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM settings WHERE key = ?`, key); err != nil {
		return fmt.Errorf("delete setting %s: %w", key, err)
	}
	return nil
}

func isValidHostname(hostname string) bool {
	if strings.Contains(hostname, "://") || strings.Contains(hostname, "/") {
		return false
	}
	if strings.HasPrefix(hostname, ".") || strings.HasSuffix(hostname, ".") {
		return false
	}
	if strings.Contains(hostname, "..") || !strings.Contains(hostname, ".") {
		return false
	}

	for _, label := range strings.Split(hostname, ".") {
		if label == "" || len(label) > 63 {
			return false
		}
		for _, r := range label {
			isAlphaNum := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
			if !isAlphaNum && r != '-' {
				return false
			}
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
	}

	return true
}
