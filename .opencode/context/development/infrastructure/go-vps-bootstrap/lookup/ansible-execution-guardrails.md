<!-- Context: development/lookup | Priority: high | Version: 1.0 | Updated: 2026-02-22 -->

# Lookup: Ansible Execution Guardrails from Go

When wiring orchestration, run Ansible with strict execution boundaries to avoid command injection and unsafe secrets handling. Keep this as a checklist before enabling runtime orchestration.

## Key Points

- Use `exec.CommandContext` with explicit args, never shell strings.
- Pin executable and working directory; allowlist playbook paths.
- Keep host key checking on and use vault-backed secret handling.
- Run `--syntax-check` and optionally `--check --diff` before apply.

## Minimal Example

```go
cmd := exec.CommandContext(ctx, "ansible-playbook", "-i", inv, playbook)
cmd.Dir = "/opt/ansible"
out, err := cmd.CombinedOutput()
_ = out
_ = err
```

## References

- Source archive: `.tmp/archive/harvested/2026-02-22/external-context/ansible/running-from-go-on-vps-safely.md`
- Docs: https://docs.ansible.com/ansible/latest/cli/ansible-playbook.html

## Codebase References

- `cmd/main.go` - Current orchestration entry points
- `README.md` - Safety constraints and future direction
- `docs/bootstrap-validation.md` - Operational validation notes

## Related

- `guides/static-dashboard-build-flow.md`
