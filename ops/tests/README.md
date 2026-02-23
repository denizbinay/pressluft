# Ops Validation

Validation flow for ops contributions:
- Validate profile contracts against `ops/schemas/profile.schema.json`.
- Run Ansible syntax checks for changed playbooks and roles.
- Add targeted role tests as execution behavior expands.

This repository currently scaffolds the checks. CI wiring can enforce these gates in a later milestone.
