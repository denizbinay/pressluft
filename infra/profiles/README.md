# Server Profiles

This directory stores auditable server profile artifacts used by Pressluft provisioning.

Goals:
- Keep infrastructure intent readable for maintainers who do not work in Go.
- Separate platform profile definitions from application handler code.
- Allow security and ops specialists to review hardening and stack defaults directly.

Each profile directory must include a `profile.yaml` file with:
- metadata (`key`, `name`, `description`, `version`)
- package/service baseline
- hardening baseline
- placeholders for bootstrap scripts or playbooks

Execution of these artifacts is intentionally out of scope for this phase. The current implementation wires profile selection and metadata only.
