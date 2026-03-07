# Pressluft Security Best Practices Audit

Date: 2026-03-07

## Executive Summary

The repository already shows good intent in a few core areas: registration tokens are hashed before storage, the agent production path uses HTTPS registration followed by `wss` plus mTLS, and SSH private keys / CA material are encrypted at rest. The main platform-level gaps are elsewhere: the operator-facing control plane is currently unauthenticated, provider API credentials are stored in plaintext, agent bootstrap credentials are written to world-readable config files, and the HTTP surface is still missing basic abuse-resistance controls.

Those are foundation issues, not edge polish. They should be fixed before the platform grows more workflows, because every future feature will inherit these trust decisions.

## Scope And Method

- Reviewed the Go control plane, agent, persistence layer, PKI/token handling, Ansible bootstrap/configure flow, and Nuxt frontend integration points.
- Findings are based only on repository-visible code and config.
- Runtime edge controls such as reverse-proxy auth, WAF, CSP/header injection, or network ACLs are not visible here; where relevant, I call that out explicitly.

## Critical

### SBP-001

- Rule ID: GO-AUTH-001
- Severity: Critical
- Location:
  - `cmd/main.go:164-170`
  - `internal/server/handler.go:31-91`
  - `internal/server/handler_providers.go:20-31`
  - `internal/server/handler_servers.go:33-46`
  - `internal/server/handler_jobs.go:60-70`
  - `internal/server/handler_activity.go:18-27`
- Evidence:
  - The control plane installs the full route tree directly into `http.Server` with only request logging middleware:
    - `Handler: server.WithRequestLogging(server.NewHandlerWithHub(db.DB, hub, wsHTTPHandler, nodeHandler), logger)`
  - `NewHandlerWithHub` mounts `/api/providers`, `/api/servers`, `/api/jobs`, and `/api/activity` directly.
  - The mutating handlers execute privileged actions immediately after request decoding, with no session, bearer-token, mTLS, or authorization checks.
- Impact: Any party that can reach the control plane can enumerate infrastructure state, add or delete provider credentials, queue provisioning and destructive jobs, and read or mutate activity data. This is a full platform compromise of the operator plane.
- Fix:
  - Put mandatory authentication and authorization in front of every operator-facing route under `/api/`.
  - Separate operator auth from agent auth; the agent mTLS and dev token flow is not a substitute for user/operator auth.
  - Add role checks before provider management, job creation, destructive server actions, and activity mutation.
  - Fail closed by default: if auth is not configured, mutating routes should not be reachable.
- Mitigation:
  - Until app-level auth exists, bind the control plane to a private management network only and enforce upstream authentication at the reverse proxy.
  - Treat the current UI/API as admin-only and not internet-safe.
- False positive notes:
  - A reverse proxy may add auth outside the repo, but there is no repository-visible guarantee that it is mandatory. Verify runtime deployment before relying on it.

## High

### SBP-002

- Rule ID: GO-CONFIG-001
- Severity: High
- Location:
  - `internal/provider/store.go:32-38`
  - `internal/provider/store.go:80-97`
  - `internal/database/migrations/00001_create_providers.sql:2-10`
- Evidence:
  - Provider credentials are inserted directly into SQLite as `api_token`:
    - `INSERT INTO providers (type, name, api_token, status, created_at, updated_at)`
  - The schema stores `api_token TEXT NOT NULL`.
  - Later reads fetch the token back verbatim for provider operations.
- Impact: Anyone who obtains the database file or a SQL read primitive gets live cloud provider credentials and can manage infrastructure outside Pressluft.
- Fix:
  - Encrypt provider API tokens at rest before persisting them, using the same class of envelope protection already used for SSH keys / CA material or, preferably, a dedicated secret store/KMS.
  - Keep only ciphertext plus key metadata in SQLite.
  - Add a rotation path for already stored provider tokens.
- Mitigation:
  - Restrict database file permissions and host access tightly.
  - Prefer short-lived or scoped provider credentials where the provider supports them.
  - Rotate all existing provider tokens after the storage model changes.
- False positive notes:
  - Hiding `APIToken` from JSON output reduces accidental API disclosure, but it does not protect the secret at rest.

### SBP-003

- Rule ID: GO-SECRETS-002
- Severity: High
- Location:
  - `ops/ansible/playbooks/configure.yml:91-96`
  - `ops/ansible/playbooks/templates/agent-config.yaml.j2:3-10`
  - `internal/agent/config.go:59-70`
- Evidence:
  - The configure playbook writes `/etc/pressluft/agent.yaml` with mode `0644`.
  - That file contains either `registration_token` or `dev_ws_token`.
  - After bootstrap, the agent rewrites the config with `os.WriteFile(path, data, 0644)`, preserving world-readable permissions.
- Impact: Any local unprivileged user or process on the managed machine can read the bootstrap or dev agent credential and impersonate the node. In dev mode, the long-lived `dev_ws_token` remains exposed for the lifetime of the file.
- Fix:
  - Change the agent config file to `0600` owned by root.
  - Split bootstrap credentials from the long-lived config, and remove them immediately after successful use.
  - Do not persist the dev bearer token in a world-readable file; keep it in a separate protected path if it must remain on disk.
  - Preserve restrictive permissions when the agent rewrites config locally.
- Mitigation:
  - If this must run before a proper fix, assume the host is single-tenant and trusted only by root.
  - Shorten token lifetimes and rotate dev tokens after each configure run.
- False positive notes:
  - If the target hosts truly never have unprivileged local users or third-party agents, the practical exposure is lower, but the default remains unsafe for a platform baseline.

## Medium

### SBP-004

- Rule ID: GO-HTTP-001 / GO-HTTP-002
- Severity: Medium
- Location:
  - `cmd/main.go:164-168`
  - `internal/server/handler_nodes.go:84-92`
  - `internal/server/handler_providers.go:72-75`
  - `internal/server/handler_jobs.go:130-133`
- Evidence:
  - The server only sets `ReadHeaderTimeout`; `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes` are absent.
  - `handleNodeRegister` performs `io.ReadAll(r.Body)` with no size cap.
  - Provider and job creation handlers decode request bodies directly without wrapping `r.Body` in `http.MaxBytesReader`.
- Impact: Slowloris-style connections and oversized request bodies can consume memory and worker time on the control plane. Because the operator plane is currently unauthenticated, the abuse threshold is lower.
- Fix:
  - Configure a full `http.Server` baseline: `ReadTimeout`, `WriteTimeout`, `IdleTimeout`, and `MaxHeaderBytes`.
  - Apply explicit body-size limits with `http.MaxBytesReader` on every JSON or CSR endpoint.
  - Add rate limiting or abuse controls around registration, job creation, and other expensive endpoints.
- Mitigation:
  - Enforce strict body limits and timeouts in the reverse proxy immediately.
  - Keep the service off the public internet until app-level controls exist.
- False positive notes:
  - A hardened reverse proxy can absorb some of this risk, but app-level limits are still recommended because direct access, misconfiguration, and internal traffic remain possible.

## Positive Notes

- Registration tokens are generated with CSPRNG output and stored as SHA-256 hashes rather than plaintext.
- Node registration consumes tokens inside the certificate persistence transaction, which avoids burning a token when CA/storage work fails.
- Production agent transport correctly requires HTTPS registration plus `wss` and mTLS for reconnect.
- SSH private keys and the CA private key are encrypted at rest rather than stored raw.

## Recommended Order

1. Add mandatory operator authentication and authorization to the control plane.
2. Encrypt provider credentials at rest and rotate existing tokens.
3. Lock down agent config permissions and remove plaintext bootstrap/dev tokens from world-readable files.
4. Add HTTP body limits, complete timeouts, and basic rate limiting.
