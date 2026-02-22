<!-- Context: openagents-repo/navigation | Priority: critical | Version: 1.1 | Updated: 2026-02-22 -->

# OpenAgents Repo Navigation

**Purpose**: Repository-specific context for agent development, testing, publishing, and maintenance.

---

## Structure

```
openagents-repo/
├── navigation.md
├── quick-start.md
├── core-concepts/
├── concepts/
├── examples/
├── guides/
├── lookup/
├── errors/
├── quality/
└── plugins/
```

---

## Quick Routes

| Task | Path |
|------|------|
| **Get oriented** | `quick-start.md` |
| **Understand agents** | `core-concepts/agents.md` |
| **Test subagents** | `guides/testing-subagents.md` |
| **Test agents** | `guides/testing-agent.md` |
| **Add agent (basics)** | `guides/adding-agent-basics.md` |
| **Add agent tests** | `guides/adding-agent-testing.md` |
| **Debug issues** | `guides/debugging.md` |
| **Update registry** | `guides/updating-registry.md` |
| **Publish package** | `guides/npm-publishing.md` |
| **Find commands** | `lookup/commands.md` |

---

## By Type

**Core Concepts**
- `core-concepts/agents.md`
- `core-concepts/evals.md`
- `core-concepts/registry.md`

**Concepts**
- `concepts/subagent-testing-modes.md`

**Examples**
- `examples/subagent-prompt-structure.md`
- `examples/context-bundle-example.md`

**Guides**
- `guides/adding-agent-basics.md`
- `guides/testing-subagents.md`
- `guides/external-libraries-workflow.md`
- `guides/creating-release.md`

**Lookup**
- `lookup/commands.md`
- `lookup/file-locations.md`
- `lookup/subagent-test-commands.md`

**Errors**
- `errors/tool-permission-errors.md`

**Quality / Plugins**
- `quality/registry-dependencies.md`
- `plugins/navigation.md`

---

## Loading Strategy

**For subagent testing**:
1. `concepts/subagent-testing-modes.md`
2. `guides/testing-subagents.md`
3. `lookup/subagent-test-commands.md`
4. `errors/tool-permission-errors.md` if tests fail

**For release prep**:
1. `guides/updating-registry.md`
2. `quality/registry-dependencies.md`
3. `guides/creating-release.md`

---

## Related Context

- `../core/navigation.md`
- `../development/navigation.md`
