# Server Profiles

Profiles define the managed intent for server classes used by Pressluft.

A profile contract is declarative. It should describe:
- identity and version (`key`, `name`, `version`, `description`)
- image policy (`base_image`, `image_policy`)
- service baseline (`services`)
- hardening expectations (`hardening`)
- execution hook order (`hooks`)
- artifact references (`artifacts`)

Contribution notes:
- Keep profile files provider-agnostic where possible.
- Treat profile version bumps as contract changes.
- Do not encode secrets in profiles.
- Validate changes against `ops/schemas/profile.schema.json`.

Current phase:
- Profiles are now the canonical source of operational intent.
- Full convergence execution lifecycle is scaffolded separately in backend orchestration.
