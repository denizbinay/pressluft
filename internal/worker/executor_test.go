package worker

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/runner"
	"pressluft/internal/security"
	"pressluft/internal/server"

	_ "modernc.org/sqlite"

	// Register provider implementations for tests that reference provider types.
	_ "pressluft/internal/provider/hetzner"
)

func TestExecutorDeleteServerSuccessMarksDeleted(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	serverStore := &fakeServerStore{servers: map[int64]*server.StoredServer{
		1: {ID: 1, ProviderID: 11, Name: "delete-me", Status: platform.ServerStatusDeleting},
	}}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		PlaybookBasePath: "playbooks",
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindDeleteServer), ServerID: 1})
	if err := executor.Execute(context.Background(), &job); err != nil {
		t.Fatalf("execute delete: %v", err)
	}

	if got := serverStore.servers[1].Status; got != platform.ServerStatusDeleted {
		t.Fatalf("server status = %q, want %q", got, platform.ServerStatusDeleted)
	}
	storedJob := mustGetExecutorJob(t, jobStore, job.ID)
	if storedJob.Status != orchestrator.JobStatusSucceeded {
		t.Fatalf("job status = %q, want %q", storedJob.Status, orchestrator.JobStatusSucceeded)
	}
}

func TestExecutorDeleteServerFailureLeavesRecoverableStatus(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	serverStore := &fakeServerStore{servers: map[int64]*server.StoredServer{
		1: {ID: 1, ProviderID: 11, Name: "delete-me", Status: platform.ServerStatusDeleting},
	}}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{failPlaybooks: map[string]error{filepath.Join("playbooks", "hetzner", "delete.yml"): errors.New("provider delete failed")}}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		PlaybookBasePath: "playbooks",
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindDeleteServer), ServerID: 1})
	err := executor.Execute(context.Background(), &job)
	if err == nil {
		t.Fatal("expected delete to fail")
	}

	if got := serverStore.servers[1].Status; got != platform.ServerStatusFailed {
		t.Fatalf("server status = %q, want %q", got, platform.ServerStatusFailed)
	}
	storedJob := mustGetExecutorJob(t, jobStore, job.ID)
	if storedJob.Status != orchestrator.JobStatusFailed {
		t.Fatalf("job status = %q, want %q", storedJob.Status, orchestrator.JobStatusFailed)
	}
}

func TestExecutorRebuildServerSuccessReconfiguresAndUpdatesImage(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	serverStore := &fakeServerStore{
		servers: map[int64]*server.StoredServer{
			1: {ID: 1, ProviderID: 11, Name: "rebuild-me", ProfileKey: "nginx-stack", Image: "ubuntu-22.04", IPv4: "203.0.113.10", Status: platform.ServerStatusRebuilding},
		},
	}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		PlaybookBasePath:      "playbooks",
		ConfigurePlaybookPath: "configure.yml",
		ControlPlaneURL:       "https://control.example.test",
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindRebuildServer),
		ServerID: 1,
		Payload:  `{"server_image":"ubuntu-24.04"}`,
	})
	if err := executor.Execute(context.Background(), &job); err != nil {
		t.Fatalf("execute rebuild: %v", err)
	}

	server := serverStore.servers[1]
	if server.Status != platform.ServerStatusConfiguring {
		t.Fatalf("server status = %q, want %q", server.Status, platform.ServerStatusConfiguring)
	}
	if server.SetupState != platform.SetupStateRunning {
		t.Fatalf("setup state = %q, want %q", server.SetupState, platform.SetupStateRunning)
	}
	if server.Image != "ubuntu-24.04" {
		t.Fatalf("server image = %q, want %q", server.Image, "ubuntu-24.04")
	}
	if len(runner.requests) != 1 {
		t.Fatalf("runner request count = %d, want 1", len(runner.requests))
	}
	jobs, err := jobStore.ListJobsByServer(context.Background(), 1)
	if err != nil {
		t.Fatalf("list jobs by server: %v", err)
	}
	if len(jobs) != 2 {
		t.Fatalf("job count = %d, want 2", len(jobs))
	}
	if jobs[0].Kind != string(orchestrator.JobKindConfigureServer) && jobs[1].Kind != string(orchestrator.JobKindConfigureServer) {
		t.Fatalf("expected configure_server job in %+v", jobs)
	}
}

func TestExecutorRebuildServerRejectsUnavailableProfile(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	privateKey := []byte("dummy-private-key")
	keyPath := filepath.Join(t.TempDir(), "age.txt")
	os.Setenv("PRESSLUFT_AGE_KEY_PATH", keyPath)
	t.Cleanup(func() { _ = os.Unsetenv("PRESSLUFT_AGE_KEY_PATH") })
	if _, err := security.EnsureAgeKey(keyPath, true); err != nil {
		t.Fatalf("ensure age key: %v", err)
	}
	encrypted, keyID, err := security.Encrypt(privateKey)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}
	mustCreateTestAgentBinary(t)

	serverStore := &fakeServerStore{
		servers: map[int64]*server.StoredServer{
			1: {ID: 1, ProviderID: 11, Name: "rebuild-me", ProfileKey: "openlitespeed-stack", Image: "ubuntu-24.04", IPv4: "203.0.113.10", Status: platform.ServerStatusRebuilding},
		},
		keys: map[int64]*server.StoredServerKey{
			1: {ServerID: 1, PrivateKeyEncrypted: encrypted, EncryptionKeyID: keyID, PublicKey: "ssh-ed25519 AAAATEST"},
		},
	}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		PlaybookBasePath:      "playbooks",
		ConfigurePlaybookPath: "configure.yml",
		ControlPlaneURL:       "https://control.example.test",
		ExecutionMode:         platform.ExecutionModeProductionBootstrap,
		RegistrationStore:     fakeRegistrationStore{},
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindRebuildServer),
		ServerID: 1,
		Payload:  `{"server_image":"ubuntu-24.04"}`,
	})
	err = executor.Execute(context.Background(), &job)
	if err == nil {
		t.Fatal("expected rebuild to fail")
	}
	if got := serverStore.servers[1].Status; got != platform.ServerStatusFailed {
		t.Fatalf("server status = %q, want %q", got, platform.ServerStatusFailed)
	}
	if len(runner.requests) != 0 {
		t.Fatalf("runner request count = %d, want 0", len(runner.requests))
	}
}

func TestExecutorResizeServerFailureMarksFailed(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	serverStore := &fakeServerStore{servers: map[int64]*server.StoredServer{
		1: {ID: 1, ProviderID: 11, Name: "resize-me", ServerType: "cx22", Status: platform.ServerStatusResizing},
	}}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{failPlaybooks: map[string]error{filepath.Join("playbooks", "hetzner", "resize.yml"): errors.New("provider resize failed")}}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		PlaybookBasePath: "playbooks",
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindResizeServer),
		ServerID: 1,
		Payload:  `{"server_type":"cx32","upgrade_disk":true}`,
	})
	err := executor.Execute(context.Background(), &job)
	if err == nil {
		t.Fatal("expected resize to fail")
	}

	if got := serverStore.servers[1].Status; got != platform.ServerStatusFailed {
		t.Fatalf("server status = %q, want %q", got, platform.ServerStatusFailed)
	}
	if got := serverStore.servers[1].ServerType; got != "cx22" {
		t.Fatalf("server type = %q, want original type", got)
	}
}

func TestExecutorConfigureServerFailureMarksSetupDegraded(t *testing.T) {
	jobStore := mustOpenExecutorJobStore(t)
	logger := testLogger()
	privateKey := []byte("dummy-private-key")
	keyPath := filepath.Join(t.TempDir(), "age.txt")
	os.Setenv("PRESSLUFT_AGE_KEY_PATH", keyPath)
	t.Cleanup(func() { _ = os.Unsetenv("PRESSLUFT_AGE_KEY_PATH") })
	if _, err := security.EnsureAgeKey(keyPath, true); err != nil {
		t.Fatalf("ensure age key: %v", err)
	}
	encrypted, keyID, err := security.Encrypt(privateKey)
	if err != nil {
		t.Fatalf("encrypt private key: %v", err)
	}
	mustCreateTestAgentBinary(t)

	serverStore := &fakeServerStore{
		servers: map[int64]*server.StoredServer{
			1: {ID: 1, ProviderID: 11, Name: "setup-me", ProfileKey: "nginx-stack", Image: "ubuntu-24.04", IPv4: "203.0.113.10", Status: platform.ServerStatusConfiguring, SetupState: platform.SetupStateRunning},
		},
		keys: map[int64]*server.StoredServerKey{
			1: {ServerID: 1, PrivateKeyEncrypted: encrypted, EncryptionKeyID: keyID, PublicKey: "ssh-ed25519 AAAATEST"},
		},
	}
	providerStore := &fakeProviderStore{provider: &provider.StoredProvider{ID: 11, Type: "hetzner", APIToken: "token"}}
	runner := &fakeRunner{failPlaybooks: map[string]error{"configure.yml": errors.New("configure failed")}}
	executor := NewExecutor(jobStore, serverStore, providerStore, nil, runner, ExecutorConfig{
		ConfigurePlaybookPath: "configure.yml",
		ControlPlaneURL:       "http://control.example.test",
		ExecutionMode:         platform.ExecutionModeDev,
		DevTokenStore:         fakeDevTokenStore{},
	}, logger)

	job := mustClaimExecutorJob(t, jobStore, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindConfigureServer),
		ServerID: 1,
		Payload:  `{"ipv4":"203.0.113.10"}`,
	})
	err = executor.Execute(context.Background(), &job)
	if err == nil {
		t.Fatal("expected configure to fail")
	}

	server := serverStore.servers[1]
	if server.Status != platform.ServerStatusConfiguring {
		t.Fatalf("server status = %q, want %q", server.Status, platform.ServerStatusConfiguring)
	}
	if server.SetupState != platform.SetupStateDegraded {
		t.Fatalf("setup state = %q, want %q", server.SetupState, platform.SetupStateDegraded)
	}
	if server.SetupLastError == "" {
		t.Fatal("expected setup last error to be recorded")
	}
}

type fakeServerStore struct {
	servers map[int64]*server.StoredServer
	keys    map[int64]*server.StoredServerKey
}

func (s *fakeServerStore) GetByID(_ context.Context, id int64) (*server.StoredServer, error) {
	server, ok := s.servers[id]
	if !ok {
		return nil, errors.New("server not found")
	}
	copy := *server
	return &copy, nil
}

func (s *fakeServerStore) UpdateStatus(_ context.Context, id int64, status platform.ServerStatus) error {
	s.servers[id].Status = status
	return nil
}

func (s *fakeServerStore) UpdateSetupState(_ context.Context, id int64, setupState platform.SetupState, setupLastError string) error {
	s.servers[id].SetupState = setupState
	s.servers[id].SetupLastError = setupLastError
	return nil
}

func (s *fakeServerStore) UpdateProvisioning(_ context.Context, id int64, providerServerID, actionID, actionStatus string, status platform.ServerStatus, ipv4, ipv6 string) error {
	server := s.servers[id]
	server.ProviderServerID = providerServerID
	server.Status = status
	server.IPv4 = ipv4
	server.IPv6 = ipv6
	return nil
}

func (s *fakeServerStore) UpdateServerType(_ context.Context, id int64, serverType string) error {
	s.servers[id].ServerType = serverType
	return nil
}

func (s *fakeServerStore) UpdateImage(_ context.Context, id int64, image string) error {
	s.servers[id].Image = image
	return nil
}

func (s *fakeServerStore) GetKey(_ context.Context, serverID int64) (*server.StoredServerKey, error) {
	key, ok := s.keys[serverID]
	if !ok {
		return nil, nil
	}
	copy := *key
	return &copy, nil
}

func (s *fakeServerStore) CreateKey(_ context.Context, in server.CreateServerKeyInput) error {
	if s.keys == nil {
		s.keys = map[int64]*server.StoredServerKey{}
	}
	s.keys[in.ServerID] = &server.StoredServerKey{ServerID: in.ServerID, PublicKey: in.PublicKey, PrivateKeyEncrypted: in.PrivateKeyEncrypted, EncryptionKeyID: in.EncryptionKeyID}
	return nil
}

type fakeProviderStore struct {
	provider *provider.StoredProvider
}

type fakeDevTokenStore struct{}

func (fakeDevTokenStore) Create(serverID int64, expiresIn time.Duration) (string, error) {
	return "dev-token", nil
}

func (s *fakeProviderStore) GetByID(context.Context, int64) (*provider.StoredProvider, error) {
	return s.provider, nil
}

type fakeRunner struct {
	requests      []runner.Request
	failPlaybooks map[string]error
}

func (r *fakeRunner) Name() string { return "fake" }

func (r *fakeRunner) Run(_ context.Context, req runner.Request, _ runner.EventSink) error {
	r.requests = append(r.requests, req)
	if artifactPath := req.ExtraVars["artifact_path"]; artifactPath != "" {
		if err := os.WriteFile(artifactPath, []byte(`{"id":123,"ipv4":"203.0.113.10","ipv6":"2001:db8::10"}`), 0o600); err != nil {
			return err
		}
	}
	if err, ok := r.failPlaybooks[req.PlaybookPath]; ok {
		return err
	}
	return nil
}

type fakeRegistrationStore struct{}

func (fakeRegistrationStore) Create(int64, time.Duration) (string, error) {
	return "registration-token", nil
}

func mustOpenExecutorJobStore(t *testing.T) *orchestrator.Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE jobs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id    INTEGER,
			kind         TEXT    NOT NULL,
			status       TEXT    NOT NULL,
			current_step TEXT    NOT NULL DEFAULT '',
			retry_count  INTEGER NOT NULL DEFAULT 0,
			last_error   TEXT,
			payload      TEXT,
			started_at   TEXT,
			finished_at  TEXT,
			timeout_at   TEXT,
			command_id   TEXT,
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL
		);
		CREATE TABLE job_events (
			job_id     INTEGER NOT NULL,
			seq        INTEGER NOT NULL,
			event_type TEXT    NOT NULL,
			level      TEXT    NOT NULL,
			step_key   TEXT,
			status     TEXT,
			message    TEXT    NOT NULL,
			payload    TEXT,
			created_at TEXT    NOT NULL,
			PRIMARY KEY (job_id, seq)
		);
	`); err != nil {
		t.Fatalf("create jobs schema: %v", err)
	}
	return orchestrator.NewStore(db)
}

func mustCreateExecutorJob(t *testing.T, store *orchestrator.Store, in orchestrator.CreateJobInput) orchestrator.Job {
	t.Helper()
	job, err := store.CreateJob(context.Background(), in)
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	return job
}

func mustClaimExecutorJob(t *testing.T, store *orchestrator.Store, in orchestrator.CreateJobInput) orchestrator.Job {
	t.Helper()
	created := mustCreateExecutorJob(t, store, in)
	claimed, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed == nil || claimed.ID != created.ID {
		t.Fatalf("claimed job = %+v, want id %d", claimed, created.ID)
	}
	return *claimed
}

func mustGetExecutorJob(t *testing.T, store *orchestrator.Store, id int64) orchestrator.Job {
	t.Helper()
	job, err := store.GetJob(context.Background(), id)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	return job
}

func mustCreateTestAgentBinary(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll("bin", 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	path := filepath.Join("bin", "pressluft-agent")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("write agent binary: %v", err)
	}
	t.Cleanup(func() { _ = os.Remove(path) })
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
