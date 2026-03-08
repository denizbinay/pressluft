package profiles

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"pressluft/internal/platform"
)

// Profile describes a server provisioning profile that maps to auditable
// operations artifacts in the ops/profiles directory.
type Profile struct {
	Key                string                `json:"key"`
	Name               string                `json:"name"`
	Description        string                `json:"description"`
	Image              string                `json:"image"`
	ArtifactPath       string                `json:"artifact_path"`
	SupportLevel       platform.SupportLevel `json:"support_level"`
	ConfigureGuarantee string                `json:"configure_guarantee"`
	SupportReason      string                `json:"support_reason,omitempty"`
}

type Artifact struct {
	Key                string               `yaml:"key" json:"key"`
	Name               string               `yaml:"name" json:"name"`
	Version            string               `yaml:"version" json:"version"`
	Description        string               `yaml:"description" json:"description"`
	BaseImage          string               `yaml:"base_image" json:"base_image"`
	ImagePolicy        ImagePolicy          `yaml:"image_policy" json:"image_policy"`
	ConfigureGuarantee string               `yaml:"configure_guarantee" json:"configure_guarantee"`
	Support            ArtifactSupport      `yaml:"support" json:"support"`
	Artifacts          ArtifactReferences   `yaml:"artifacts" json:"artifacts"`
	Baseline           BaselineContract     `yaml:"baseline" json:"baseline"`
	Verification       VerificationContract `yaml:"verification" json:"verification"`
}

type ArtifactSupport struct {
	Level  platform.SupportLevel `yaml:"level" json:"level"`
	Reason string                `yaml:"reason" json:"reason"`
}

type ArtifactReferences struct {
	Playbook     string `yaml:"playbook" json:"playbook"`
	Role         string `yaml:"role" json:"role"`
	LoadingModel string `yaml:"loading_model" json:"loading_model"`
}

type ImagePolicy struct {
	AllowOverride bool     `yaml:"allow_override" json:"allow_override"`
	Allowed       []string `yaml:"allowed" json:"allowed"`
}

type BaselineContract struct {
	PackageUpdate string               `yaml:"package_update" json:"package_update"`
	Directories   []DirectoryContract  `yaml:"directories" json:"directories"`
	Users         []UserContract       `yaml:"users" json:"users"`
	Packages      []string             `yaml:"packages" json:"packages"`
	Firewall      FirewallContract     `yaml:"firewall" json:"firewall"`
	Verification  VerificationContract `yaml:"verification" json:"verification"`
}

type DirectoryContract struct {
	Path  string `yaml:"path" json:"path"`
	Owner string `yaml:"owner" json:"owner"`
	Group string `yaml:"group" json:"group"`
	Mode  string `yaml:"mode" json:"mode"`
}

type UserContract struct {
	Name   string `yaml:"name" json:"name"`
	Group  string `yaml:"group" json:"group"`
	Home   string `yaml:"home" json:"home"`
	Shell  string `yaml:"shell" json:"shell"`
	System bool   `yaml:"system" json:"system"`
}

type FirewallContract struct {
	Enabled         bool   `yaml:"enabled" json:"enabled"`
	DefaultIncoming string `yaml:"default_incoming" json:"default_incoming"`
	DefaultOutgoing string `yaml:"default_outgoing" json:"default_outgoing"`
	AllowedTCPPorts []int  `yaml:"allowed_tcp_ports" json:"allowed_tcp_ports"`
}

type VerificationContract struct {
	Services     []ServiceVerification  `yaml:"services" json:"services"`
	Files        []FileVerification     `yaml:"files" json:"files"`
	Listeners    []ListenerVerification `yaml:"listeners" json:"listeners"`
	HealthChecks []HealthCheck          `yaml:"health_checks" json:"health_checks"`
}

type ServiceVerification struct {
	Name  string `yaml:"name" json:"name"`
	State string `yaml:"state" json:"state"`
}

type FileVerification struct {
	Path string `yaml:"path" json:"path"`
}

type ListenerVerification struct {
	Name  string `yaml:"name" json:"name"`
	Host  string `yaml:"host" json:"host"`
	Port  int    `yaml:"port" json:"port"`
	State string `yaml:"state" json:"state"`
}

type HealthCheck struct {
	Name    string `yaml:"name" json:"name"`
	Command string `yaml:"command" json:"command"`
}

var registry = mustLoadRegistry()

// All returns all available server profiles.
func All() []Profile {
	out := make([]Profile, len(registry))
	copy(out, registry)
	return out
}

// Get returns a profile by key.
func Get(key string) (Profile, bool) {
	for _, profile := range registry {
		if profile.Key == key {
			return profile, true
		}
	}
	return Profile{}, false
}

// Selectable returns whether the profile may be used for new server workflows.
func (p Profile) Selectable() bool {
	return p.SupportLevel == platform.SupportLevelSupported || p.SupportLevel == platform.SupportLevelExperimental
}

func LoadArtifact(path string) (Artifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Artifact{}, err
	}
	var artifact Artifact
	if err := yaml.Unmarshal(data, &artifact); err != nil {
		return Artifact{}, err
	}
	return artifact, nil
}

func mustLoadRegistry() []Profile {
	profiles, err := loadRegistry()
	if err != nil {
		panic(err)
	}
	return profiles
}

func loadRegistry() ([]Profile, error) {
	root, err := repoRoot()
	if err != nil {
		return nil, err
	}
	artifactPaths, err := discoverArtifactPaths(root)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(artifactPaths))
	for _, artifactPath := range artifactPaths {
		artifact, err := LoadArtifact(filepath.Join(root, filepath.FromSlash(artifactPath)))
		if err != nil {
			return nil, fmt.Errorf("load profile artifact %q: %w", artifactPath, err)
		}
		out = append(out, Profile{
			Key:                artifact.Key,
			Name:               artifact.Name,
			Description:        artifact.Description,
			Image:              artifact.BaseImage,
			ArtifactPath:       artifactPath,
			SupportLevel:       artifact.Support.Level,
			ConfigureGuarantee: artifact.ConfigureGuarantee,
			SupportReason:      artifact.Support.Reason,
		})
	}
	return out, nil
}

func discoverArtifactPaths(root string) ([]string, error) {
	entries, err := os.ReadDir(filepath.Join(root, "ops", "profiles"))
	if err != nil {
		return nil, fmt.Errorf("read ops/profiles: %w", err)
	}
	artifactPaths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		artifactPath := filepath.Join("ops", "profiles", entry.Name(), "profile.yaml")
		fullPath := filepath.Join(root, artifactPath)
		if _, err := os.Stat(fullPath); err != nil {
			return nil, fmt.Errorf("profile artifact missing for %q: %w", entry.Name(), err)
		}
		artifactPaths = append(artifactPaths, filepath.ToSlash(artifactPath))
	}
	sort.Strings(artifactPaths)
	return artifactPaths, nil
}

func repoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve registry source path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(filename), "../../..")), nil
}

func ValidateRegistryArtifacts(repoRoot string) error {
	entries, err := os.ReadDir(filepath.Join(repoRoot, "ops", "profiles"))
	if err != nil {
		return fmt.Errorf("read ops/profiles: %w", err)
	}

	expectedPaths, err := discoverArtifactPaths(repoRoot)
	if err != nil {
		return err
	}
	expectedByKey := make(map[string]string, len(expectedPaths))
	for _, artifactPath := range expectedPaths {
		artifact, err := LoadArtifact(filepath.Join(repoRoot, artifactPath))
		if err != nil {
			return fmt.Errorf("load artifact %q: %w", artifactPath, err)
		}
		expectedByKey[artifact.Key] = artifactPath
	}

	artifactKeys := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		artifactPath := filepath.Join(repoRoot, "ops", "profiles", entry.Name(), "profile.yaml")
		if _, err := os.Stat(artifactPath); err != nil {
			return fmt.Errorf("profile artifact missing for %q: %w", entry.Name(), err)
		}
		artifact, err := LoadArtifact(artifactPath)
		if err != nil {
			return fmt.Errorf("load artifact %q: %w", artifactPath, err)
		}
		artifactKeys = append(artifactKeys, artifact.Key)

		profile, ok := Get(artifact.Key)
		if !ok {
			return fmt.Errorf("artifact key %q missing from registry", artifact.Key)
		}
		if profile.ArtifactPath != expectedByKey[artifact.Key] {
			return fmt.Errorf("registry artifact path mismatch for %q: got %q", artifact.Key, profile.ArtifactPath)
		}
		if profile.Name != artifact.Name {
			return fmt.Errorf("registry name mismatch for %q", artifact.Key)
		}
		if profile.Image != artifact.BaseImage {
			return fmt.Errorf("registry image mismatch for %q", artifact.Key)
		}
		if profile.SupportLevel != artifact.Support.Level {
			return fmt.Errorf("registry support level mismatch for %q", artifact.Key)
		}
		if normalizeSpace(profile.ConfigureGuarantee) != normalizeSpace(artifact.ConfigureGuarantee) {
			return fmt.Errorf("registry configure guarantee mismatch for %q", artifact.Key)
		}
		if normalizeSpace(profile.SupportReason) != normalizeSpace(artifact.Support.Reason) {
			return fmt.Errorf("registry support reason mismatch for %q", artifact.Key)
		}
	}

	registryKeys := make([]string, 0, len(registry))
	for _, profile := range registry {
		registryKeys = append(registryKeys, profile.Key)
	}
	if !sameStrings(registryKeys, artifactKeys) {
		sort.Strings(registryKeys)
		sort.Strings(artifactKeys)
		return fmt.Errorf("registry keys %v do not match ops/profiles keys %v", registryKeys, artifactKeys)
	}

	return nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func normalizeSpace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
