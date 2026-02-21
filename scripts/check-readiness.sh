#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT_DIR" <<'PY'
import json
import os
import re
import sys

root = sys.argv[1]
docs_root = os.path.join(root, "docs")

required_headers = [
    "Status:",
    "Owner:",
    "Last Reviewed:",
    "Depends On:",
    "Supersedes:",
]

missing = []
for dirpath, _, filenames in os.walk(docs_root):
    for filename in filenames:
        if not filename.endswith(".md"):
            continue
        full = os.path.join(dirpath, filename)
        with open(full, "r", encoding="utf-8") as f:
            lines = [f.readline().rstrip("\n") for _ in range(8)]
        for header in required_headers:
            if not any(line.startswith(header) for line in lines):
                missing.append((os.path.relpath(full, root), header))

if missing:
    print("Readiness check failed: missing metadata headers")
    for path, header in missing:
        print(f"  - {path}: missing '{header}'")
    sys.exit(1)

migration_files = os.listdir(os.path.join(root, "migrations"))
up = sorted([m for m in migration_files if m.endswith(".up.sql")])
down = sorted([m for m in migration_files if m.endswith(".down.sql")])

if not up or not down:
    print("Readiness check failed: missing baseline migration files")
    sys.exit(1)

required_paths = [
    "SPEC.md",
    "ARCHITECTURE.md",
    "CONTRACTS.md",
    "opencode.json",
    "contracts/openapi.yaml",
    "docs/contract-traceability.md",
    "docs/adr/README.md",
    "docs/unattended-orchestration.md",
    "docs/error-codes.md",
    "docs/job-types.md",
    "coordination/locks.md",
    "PLAN.md",
    "PROGRESS.md",
]

missing_paths = [p for p in required_paths if not os.path.exists(os.path.join(root, p))]
if missing_paths:
    print("Readiness check failed: missing required files")
    for path in missing_paths:
        print(f"  - {path}")
    sys.exit(1)

required_agent_files = [
    ".opencode/agents/architect.md",
    ".opencode/agents/implementer.md",
    ".opencode/agents/reviewer.md",
    ".opencode/agents/spec-auditor.md",
]

missing_agent_files = [p for p in required_agent_files if not os.path.exists(os.path.join(root, p))]
if missing_agent_files:
    print("Readiness check failed: missing required OpenCode agent files")
    for path in missing_agent_files:
        print(f"  - {path}")
    sys.exit(1)

required_command_files = [
    ".opencode/commands/readiness.md",
    ".opencode/commands/backend-gates.md",
    ".opencode/commands/frontend-gates.md",
    ".opencode/commands/session-kickoff.md",
    ".opencode/commands/run-plan.md",
    ".opencode/commands/resume-run.md",
    ".opencode/commands/triage-failures.md",
]

missing_command_files = [p for p in required_command_files if not os.path.exists(os.path.join(root, p))]
if missing_command_files:
    print("Readiness check failed: missing required OpenCode command files")
    for path in missing_command_files:
        print(f"  - {path}")
    sys.exit(1)


config_files = [
    "opencode.json",
]

required_instructions = [
    "AGENTS.md",
    "docs/spec-index.md",
    "PLAN.md",
    "PROGRESS.md",
]

for config_name in config_files:
    opencode_path = os.path.join(root, config_name)
    with open(opencode_path, "r", encoding="utf-8") as f:
        try:
            opencode = json.load(f)
        except json.JSONDecodeError as err:
            print(f"Readiness check failed: invalid {config_name} ({err})")
            sys.exit(1)

    instructions = opencode.get("instructions")
    if not isinstance(instructions, list) or not instructions:
        print(f"Readiness check failed: {config_name} must define a non-empty instructions array")
        sys.exit(1)

    missing_instructions = [p for p in required_instructions if p not in instructions]
    if missing_instructions:
        print(f"Readiness check failed: {config_name} missing required instruction paths")
        for path in missing_instructions:
            print(f"  - {path}")
        sys.exit(1)

    missing_instruction_files = [p for p in instructions if not os.path.exists(os.path.join(root, p))]
    if missing_instruction_files:
        print(f"Readiness check failed: {config_name} references missing instruction files")
        for path in missing_instruction_files:
            print(f"  - {path}")
        sys.exit(1)

    permission = opencode.get("permission")
    if not isinstance(permission, dict):
        print(f"Readiness check failed: {config_name} must define a permission object")
        sys.exit(1)

    for key in ("edit", "bash", "webfetch"):
        if key not in permission:
            print(f"Readiness check failed: {config_name} permission missing '{key}'")
            sys.exit(1)

    agents = opencode.get("agent")
    if not isinstance(agents, dict):
        print(f"Readiness check failed: {config_name} must define an agent object")
        sys.exit(1)

    for name in ("build", "plan"):
        agent = agents.get(name)
        if not isinstance(agent, dict):
            print(f"Readiness check failed: {config_name} agent missing '{name}' config")
            sys.exit(1)
        agent_perm = agent.get("permission")
        if not isinstance(agent_perm, dict):
            print(f"Readiness check failed: {config_name} agent '{name}' missing permission object")
            sys.exit(1)
        task_perm = agent_perm.get("task")
        if not isinstance(task_perm, dict):
            print(f"Readiness check failed: {config_name} agent '{name}' missing permission.task object")
            sys.exit(1)
        if "*" not in task_perm:
            print(f"Readiness check failed: {config_name} agent '{name}' permission.task must include '*' rule")
            sys.exit(1)

print("Readiness check passed")
PY

"$ROOT_DIR/scripts/check-contract-traceability.sh"
"$ROOT_DIR/scripts/check-job-error-registry.sh"
"$ROOT_DIR/scripts/check-parallel-locks.sh"

echo "All readiness checks passed"
