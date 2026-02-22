package store

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

var (
	ErrSiteNotFound             = errors.New("site not found")
	ErrSiteSlugConflict         = errors.New("site slug already exists")
	ErrEnvironmentNotFound      = errors.New("environment not found")
	ErrEnvironmentSlugConflict  = errors.New("environment slug already exists")
	ErrInvalidEnvironmentCreate = errors.New("invalid environment create request")
	ErrInvalidEnvironmentStatus = errors.New("invalid environment status transition")
)

type Site struct {
	ID                   string    `json:"id"`
	Name                 string    `json:"name"`
	Slug                 string    `json:"slug"`
	Status               string    `json:"status"`
	PrimaryEnvironmentID *string   `json:"primary_environment_id"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
	StateVersion         int       `json:"state_version"`
}

type Environment struct {
	ID                  string     `json:"id"`
	SiteID              string     `json:"site_id"`
	Name                string     `json:"name"`
	Slug                string     `json:"slug"`
	EnvironmentType     string     `json:"environment_type"`
	Status              string     `json:"status"`
	NodeID              string     `json:"node_id"`
	SourceEnvironmentID *string    `json:"source_environment_id"`
	PromotionPreset     string     `json:"promotion_preset"`
	PreviewURL          string     `json:"preview_url"`
	PrimaryDomainID     *string    `json:"primary_domain_id"`
	CurrentReleaseID    *string    `json:"current_release_id"`
	DriftStatus         string     `json:"drift_status"`
	DriftCheckedAt      *time.Time `json:"drift_checked_at"`
	LastDriftCheckID    *string    `json:"last_drift_check_id"`
	FastCGICacheEnabled bool       `json:"fastcgi_cache_enabled"`
	RedisCacheEnabled   bool       `json:"redis_cache_enabled"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	StateVersion        int        `json:"state_version"`
}

type SiteStore interface {
	CountSites(ctx context.Context) (int, error)
	ListSites(ctx context.Context) ([]Site, error)
	GetSiteByID(ctx context.Context, id string) (Site, error)
	ListEnvironmentsBySiteID(ctx context.Context, siteID string) ([]Environment, error)
	GetEnvironmentByID(ctx context.Context, id string) (Environment, error)
	CreateSiteWithProductionEnvironment(ctx context.Context, input CreateSiteInput) (Site, Environment, error)
	CreateEnvironment(ctx context.Context, input CreateEnvironmentInput) (Environment, error)
	MarkEnvironmentRestoring(ctx context.Context, environmentID string, now time.Time) (Environment, Site, error)
	MarkEnvironmentRestoreResult(ctx context.Context, environmentID string, succeeded bool, now time.Time) (Environment, Site, error)
}

type CreateSiteInput struct {
	Name       string
	Slug       string
	NodeID     string
	NodePublic string
	Now        time.Time
}

type CreateEnvironmentInput struct {
	SiteID              string
	Name                string
	Slug                string
	EnvironmentType     string
	SourceEnvironmentID *string
	PromotionPreset     string
	Now                 time.Time
}

type InMemorySiteStore struct {
	mu            sync.RWMutex
	sites         []Site
	byID          map[string]Site
	bySlug        map[string]string
	environments  map[string]Environment
	envBySite     map[string][]string
	envSlugBySite map[string]map[string]string
}

var (
	globalSiteStoreMu sync.RWMutex
	globalSiteStore   SiteStore
)

func NewInMemorySiteStore(totalCount int) *InMemorySiteStore {
	seedNow := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	seed := make([]Site, 0, totalCount)
	for i := 0; i < totalCount; i++ {
		id := fmt.Sprintf("00000000-0000-0000-0000-%012d", i+1)
		envID := fmt.Sprintf("10000000-0000-0000-0000-%012d", i+1)
		slug := fmt.Sprintf("seed-site-%d", i+1)
		seed = append(seed, Site{
			ID:                   id,
			Name:                 fmt.Sprintf("Seed Site %d", i+1),
			Slug:                 slug,
			Status:               "active",
			PrimaryEnvironmentID: stringPtr(envID),
			CreatedAt:            seedNow,
			UpdatedAt:            seedNow,
			StateVersion:         1,
		})
	}

	store := newInMemorySiteStoreWithSites(seed)
	setDefaultSiteStore(store)
	return store
}

func NewInMemorySiteStoreWithSeed(seed []Site) *InMemorySiteStore {
	store := newInMemorySiteStoreWithSites(seed)
	setDefaultSiteStore(store)
	return store
}

func newInMemorySiteStoreWithSites(seed []Site) *InMemorySiteStore {
	byID := make(map[string]Site, len(seed))
	bySlug := make(map[string]string, len(seed))
	items := make([]Site, 0, len(seed))
	environments := make(map[string]Environment)
	envBySite := make(map[string][]string)
	envSlugBySite := make(map[string]map[string]string)

	for _, site := range seed {
		if site.ID == "" || site.Slug == "" {
			continue
		}
		byID[site.ID] = site
		bySlug[site.Slug] = site.ID
		items = append(items, site)
	}

	return &InMemorySiteStore{sites: items, byID: byID, bySlug: bySlug, environments: environments, envBySite: envBySite, envSlugBySite: envSlugBySite}
}

func DefaultSiteStore() SiteStore {
	globalSiteStoreMu.RLock()
	current := globalSiteStore
	globalSiteStoreMu.RUnlock()
	if current != nil {
		return current
	}
	return NewInMemorySiteStore(0)
}

func setDefaultSiteStore(store SiteStore) {
	globalSiteStoreMu.Lock()
	defer globalSiteStoreMu.Unlock()
	globalSiteStore = store
}

func (s *InMemorySiteStore) CountSites(context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.sites), nil
}

func (s *InMemorySiteStore) ListSites(context.Context) ([]Site, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cloned := make([]Site, len(s.sites))
	copy(cloned, s.sites)
	return cloned, nil
}

func (s *InMemorySiteStore) GetSiteByID(_ context.Context, id string) (Site, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	site, ok := s.byID[id]
	if !ok {
		return Site{}, ErrSiteNotFound
	}
	return site, nil
}

func (s *InMemorySiteStore) CreateSiteWithProductionEnvironment(_ context.Context, input CreateSiteInput) (Site, Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	normalizedSlug := strings.TrimSpace(input.Slug)
	if _, exists := s.bySlug[normalizedSlug]; exists {
		return Site{}, Environment{}, ErrSiteSlugConflict
	}

	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	siteID, err := generateUUIDv4()
	if err != nil {
		return Site{}, Environment{}, fmt.Errorf("generate site id: %w", err)
	}
	environmentID, err := generateUUIDv4()
	if err != nil {
		return Site{}, Environment{}, fmt.Errorf("generate environment id: %w", err)
	}

	envSlug := "production"
	nodeIP := sanitizeNodeIP(input.NodePublic)
	preview := fmt.Sprintf("http://%s.%s.sslip.io", environmentID[:8], nodeIP)

	environment := Environment{
		ID:                  environmentID,
		SiteID:              siteID,
		Name:                "Production",
		Slug:                envSlug,
		EnvironmentType:     "production",
		Status:              "active",
		NodeID:              input.NodeID,
		PromotionPreset:     "content-protect",
		PreviewURL:          preview,
		DriftStatus:         "unknown",
		FastCGICacheEnabled: true,
		RedisCacheEnabled:   true,
		CreatedAt:           now,
		UpdatedAt:           now,
		StateVersion:        1,
	}

	site := Site{
		ID:                   siteID,
		Name:                 strings.TrimSpace(input.Name),
		Slug:                 normalizedSlug,
		Status:               "active",
		PrimaryEnvironmentID: stringPtr(environmentID),
		CreatedAt:            now,
		UpdatedAt:            now,
		StateVersion:         1,
	}

	s.byID[site.ID] = site
	s.bySlug[site.Slug] = site.ID
	s.sites = append(s.sites, site)
	s.insertEnvironmentLocked(environment)

	return site, environment, nil
}

func (s *InMemorySiteStore) GetEnvironmentByID(_ context.Context, id string) (Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	environment, ok := s.environments[id]
	if !ok {
		return Environment{}, ErrEnvironmentNotFound
	}
	return environment, nil
}

func (s *InMemorySiteStore) ListEnvironmentsBySiteID(_ context.Context, siteID string) ([]Environment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.byID[siteID]; !ok {
		return nil, ErrSiteNotFound
	}

	ids := s.envBySite[siteID]
	result := make([]Environment, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.environments[id])
	}

	return result, nil
}

func (s *InMemorySiteStore) CreateEnvironment(_ context.Context, input CreateEnvironmentInput) (Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	site, ok := s.byID[input.SiteID]
	if !ok {
		return Environment{}, ErrSiteNotFound
	}

	normalizedType := strings.TrimSpace(input.EnvironmentType)
	if normalizedType != "staging" && normalizedType != "clone" {
		return Environment{}, ErrInvalidEnvironmentCreate
	}

	normalizedPreset := strings.TrimSpace(input.PromotionPreset)
	if normalizedPreset != "content-protect" && normalizedPreset != "commerce-protect" {
		return Environment{}, ErrInvalidEnvironmentCreate
	}

	normalizedName := strings.TrimSpace(input.Name)
	normalizedSlug := strings.TrimSpace(input.Slug)
	if normalizedName == "" || normalizedSlug == "" {
		return Environment{}, ErrInvalidEnvironmentCreate
	}

	if _, ok := s.envSlugBySite[input.SiteID]; !ok {
		s.envSlugBySite[input.SiteID] = make(map[string]string)
	}
	if _, exists := s.envSlugBySite[input.SiteID][normalizedSlug]; exists {
		return Environment{}, ErrEnvironmentSlugConflict
	}

	if input.SourceEnvironmentID == nil || strings.TrimSpace(*input.SourceEnvironmentID) == "" {
		return Environment{}, ErrInvalidEnvironmentCreate
	}

	sourceID := strings.TrimSpace(*input.SourceEnvironmentID)
	sourceEnvironment, ok := s.environments[sourceID]
	if !ok || sourceEnvironment.SiteID != input.SiteID {
		return Environment{}, ErrInvalidEnvironmentCreate
	}

	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	environmentID, err := generateUUIDv4()
	if err != nil {
		return Environment{}, fmt.Errorf("generate environment id: %w", err)
	}

	preview := deterministicPreviewURL(environmentID, previewSuffixFromEnvironment(sourceEnvironment))
	environment := Environment{
		ID:                  environmentID,
		SiteID:              input.SiteID,
		Name:                normalizedName,
		Slug:                normalizedSlug,
		EnvironmentType:     normalizedType,
		Status:              "cloning",
		NodeID:              sourceEnvironment.NodeID,
		SourceEnvironmentID: stringPtr(sourceID),
		PromotionPreset:     normalizedPreset,
		PreviewURL:          preview,
		DriftStatus:         "unknown",
		FastCGICacheEnabled: true,
		RedisCacheEnabled:   true,
		CreatedAt:           now,
		UpdatedAt:           now,
		StateVersion:        1,
	}

	s.insertEnvironmentLocked(environment)

	if site.Status == "active" {
		site.Status = "cloning"
	}
	site.UpdatedAt = now
	site.StateVersion++
	s.byID[input.SiteID] = site
	s.replaceSiteInListLocked(site)

	return environment, nil
}

func (s *InMemorySiteStore) MarkEnvironmentRestoring(_ context.Context, environmentID string, now time.Time) (Environment, Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	environment, ok := s.environments[environmentID]
	if !ok {
		return Environment{}, Site{}, ErrEnvironmentNotFound
	}
	if environment.Status != "active" {
		return Environment{}, Site{}, ErrInvalidEnvironmentStatus
	}

	site, ok := s.byID[environment.SiteID]
	if !ok {
		return Environment{}, Site{}, ErrSiteNotFound
	}

	updatedAt := now.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	environment.Status = "restoring"
	environment.UpdatedAt = updatedAt
	environment.StateVersion++
	s.environments[environment.ID] = environment

	site.Status = "restoring"
	site.UpdatedAt = updatedAt
	site.StateVersion++
	s.byID[site.ID] = site
	s.replaceSiteInListLocked(site)

	return environment, site, nil
}

func (s *InMemorySiteStore) MarkEnvironmentRestoreResult(_ context.Context, environmentID string, succeeded bool, now time.Time) (Environment, Site, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	environment, ok := s.environments[environmentID]
	if !ok {
		return Environment{}, Site{}, ErrEnvironmentNotFound
	}
	if environment.Status != "restoring" {
		return Environment{}, Site{}, ErrInvalidEnvironmentStatus
	}

	site, ok := s.byID[environment.SiteID]
	if !ok {
		return Environment{}, Site{}, ErrSiteNotFound
	}

	updatedAt := now.UTC()
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	if succeeded {
		environment.Status = "active"
		site.Status = "active"
	} else {
		environment.Status = "failed"
		site.Status = "failed"
	}
	environment.UpdatedAt = updatedAt
	environment.StateVersion++
	s.environments[environment.ID] = environment

	site.UpdatedAt = updatedAt
	site.StateVersion++
	s.byID[site.ID] = site
	s.replaceSiteInListLocked(site)

	return environment, site, nil
}

func (s *InMemorySiteStore) insertEnvironmentLocked(environment Environment) {
	s.environments[environment.ID] = environment
	s.envBySite[environment.SiteID] = append(s.envBySite[environment.SiteID], environment.ID)
	if _, ok := s.envSlugBySite[environment.SiteID]; !ok {
		s.envSlugBySite[environment.SiteID] = make(map[string]string)
	}
	s.envSlugBySite[environment.SiteID][environment.Slug] = environment.ID
}

func (s *InMemorySiteStore) replaceSiteInListLocked(site Site) {
	for i := range s.sites {
		if s.sites[i].ID == site.ID {
			s.sites[i] = site
			return
		}
	}
}

func previewSuffixFromEnvironment(environment Environment) string {
	if environment.PreviewURL == "" {
		return "127-0-0-1"
	}

	parsed, err := url.Parse(environment.PreviewURL)
	if err != nil || parsed.Hostname() == "" {
		return "127-0-0-1"
	}

	host := parsed.Hostname()
	parts := strings.Split(host, ".")
	if len(parts) < 3 {
		return "127-0-0-1"
	}

	suffix := strings.Join(parts[1:len(parts)-2], ".")
	if suffix == "" {
		return "127-0-0-1"
	}

	return suffix
}

func deterministicPreviewURL(environmentID string, suffix string) string {
	trimmed := strings.TrimSpace(suffix)
	if trimmed == "" {
		trimmed = "127-0-0-1"
	}
	return fmt.Sprintf("http://%s.%s.sslip.io", environmentID[:8], trimmed)
}

func sanitizeNodeIP(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		normalized = "127.0.0.1"
	}
	return strings.ReplaceAll(normalized, ".", "-")
}

func generateUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

func stringPtr(v string) *string {
	vv := v
	return &vv
}
