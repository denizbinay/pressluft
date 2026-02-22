<!-- Context: development/navigation | Priority: critical | Version: 1.0 | Updated: 2026-02-15 -->

# Infrastructure Navigation

**Purpose**: DevOps and deployment patterns

**Status**: Active

---

## Planned Structure

```
infrastructure/
├── navigation.md
│
├── go-vps-bootstrap/
│   ├── navigation.md
│   ├── concepts/
│   ├── examples/
│   ├── guides/
│   ├── lookup/
│   └── errors/
│
├── docker/
│   ├── dockerfile-patterns.md
│   ├── compose-patterns.md
│   └── optimization.md
│
└── ci-cd/
    ├── github-actions.md
    ├── deployment-patterns.md
    └── testing-pipelines.md
```

---

## Available Context

| Area | Path | Purpose |
|------|------|---------|
| Go + Nuxt VPS bootstrap | `go-vps-bootstrap/navigation.md` | Single-binary hosting patterns and safeguards |

---

## Related Context

- **Core Standards** → `../../core/standards/code-quality.md`
- **Testing** → `../../core/standards/test-coverage.md`
