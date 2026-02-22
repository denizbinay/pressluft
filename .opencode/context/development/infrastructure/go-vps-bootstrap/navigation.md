<!-- Context: development/navigation | Priority: high | Version: 1.0 | Updated: 2026-02-22 -->

# Go VPS Bootstrap Context

**Purpose**: Minimal patterns for running a Go API with a Nuxt dashboard on a VPS, with safe orchestration foundations.

---

## Quick Navigation

### Concepts
| File | Description | Priority |
|------|-------------|----------|
| `concepts/go-embedded-dashboard.md` | Embed and serve dashboard assets from Go | critical |
| `concepts/nuxt-go-deployment-mode.md` | Choose static vs SSR for Go hosting | high |

### Guides
| File | Description | Priority |
|------|-------------|----------|
| `guides/static-dashboard-build-flow.md` | Build and publish static Nuxt output for Go | high |

### Examples
| File | Description | Priority |
|------|-------------|----------|
| `examples/minimal-embed-handler.md` | Minimal Go embed + file server pattern | medium |

### Lookup
| File | Description | Priority |
|------|-------------|----------|
| `lookup/ansible-execution-guardrails.md` | Safe defaults for running Ansible from Go | high |

### Errors
| File | Description | Priority |
|------|-------------|----------|
| `errors/dashboard-assets-missing.md` | Missing embedded dashboard output at runtime | medium |

---

## Loading Strategy

1. Start with `concepts/go-embedded-dashboard.md`.
2. Read `concepts/nuxt-go-deployment-mode.md` for hosting mode decisions.
3. Use `guides/static-dashboard-build-flow.md` for build/deploy flow.
4. Copy from `examples/minimal-embed-handler.md` for bootstrap implementation.
5. Apply `lookup/ansible-execution-guardrails.md` before orchestration work.
6. Use `errors/dashboard-assets-missing.md` when root UI is missing.
7. Validate from repo root with `make check` before commit/deploy.
