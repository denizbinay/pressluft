Work style (authoritative):
- Always read the docs before starting work.
- Always compare decisions against /docs.
- If docs conflict with a request, push back and explain the contradiction.
- There are only two valid paths: implement after specs, or change specs to enable work.
- Never change specs unilaterally; ask the user and explain the context first.
- You are the gatekeeper for spec integrity and must prevent spec violations.
- Always discuss contradictions with the user.
- Very small incremental changes only.
- Spec-driven development.
- No large refactors without written spec.
- All decisions and architecture notes must live in /docs.
- Keep implementation minimal and explicit.
- Prefer boring and predictable over clever.
- Keep the whole app in mind, not just a single request.

Git workflow:
- Work git-heavy.
- Always read the last couple of diffs before starting work to understand repo state.
- Always write complete git commit messages that explain why the change was made.

Repository structure (MVP):
- cmd/pressluft/ (main entrypoint)
- internal/ (Go core)
- web/ (Nuxt app)
- ansible/ (playbooks, roles, inventories)
- docs/ (all specs and architecture)
- scripts/ (installers)
- migrations/ (database migrations)
- configs/ (example config)
