<!-- Context: project-intelligence/nav | Priority: high | Version: 1.2 | Updated: 2026-02-23 -->

# Project Intelligence

> Start here for quick project understanding. These files bridge business and technical domains.

## Structure

```
.opencode/context/project-intelligence/
├── navigation.md              # This file - quick overview
├── business-domain.md         # Business context and problem statement
├── technical-domain.md        # Stack, architecture, technical decisions
├── business-tech-bridge.md    # How business needs map to solutions
├── decisions-log.md           # Major decisions with rationale
└── living-notes.md            # Active issues, debt, open questions
```

## Quick Routes

| What You Need | File | Description |
|---------------|------|-------------|
| Understand the "why" | `business-domain.md` | Problem, users, value proposition |
| Understand the "how" | `technical-domain.md` | Stack, architecture, integrations |
| See the connection | `business-tech-bridge.md` | Business → technical mapping |
| Know the context | `decisions-log.md` | Why decisions were made |
| Current state | `living-notes.md` | Active issues and open questions |
| Provider architecture | `technical-domain.md` + `decisions-log.md` | SQLite persistence, provider interface, Hetzner integration |
| Server provisioning architecture | `technical-domain.md` + `decisions-log.md` | Provider-agnostic server contracts, profiles, and Hetzner create flow |
| All of the above | Read all files in order | Full project intelligence |

## Usage

**New Team Member / Agent**:
1. Start with `navigation.md` (this file)
2. Read all files in order for complete understanding
3. Follow onboarding checklist in each file

**Quick Reference**:
- Business focus → `business-domain.md`
- Technical focus → `technical-domain.md`
- Decision context → `decisions-log.md`

## Integration

This folder is referenced from:
- `.opencode/context/core/standards/project-intelligence.md` (standards and patterns)
- `.opencode/context/core/system/context-guide.md` (context loading)

See `.opencode/context/core/context-system.md` for the broader context architecture.

## Maintenance

Keep this folder current:
- Update when business direction changes
- Document decisions as they're made
- Review `living-notes.md` regularly
- Archive resolved items from decisions-log.md

**Management Guide**: See `.opencode/context/core/standards/project-intelligence-management.md` for complete lifecycle management including:
- How to update, add, and remove files
- How to create new subfolders
- Version tracking and frontmatter standards
- Quality checklists and anti-patterns
- Governance and ownership

See `.opencode/context/core/standards/project-intelligence.md` for the standard itself.

## 📂 Codebase References

- `internal/database/` - Persistence and migration implementation details
- `internal/provider/` - Provider abstractions and concrete adapters
- `internal/server/handler_providers.go` - Provider API routing and handlers
- `internal/server/handler_servers.go` - Server provisioning API routing and handlers
- `internal/server/store_servers.go` - Server persistence and provisioning state
- `web/app/components/SettingsProviders.vue` - Providers settings UX implementation
- `web/app/components/SettingsServers.vue` - Servers settings UX implementation
- `web/app/composables/useServers.ts` - Servers frontend API client
