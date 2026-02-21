#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT_DIR" <<'PY'
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
    "contracts/openapi.yaml",
    "docs/contract-traceability.md",
    "docs/error-codes.md",
    "docs/job-types.md",
    "PLAN.md",
    "PROGRESS.md",
]

missing_paths = [p for p in required_paths if not os.path.exists(os.path.join(root, p))]
if missing_paths:
    print("Readiness check failed: missing required files")
    for path in missing_paths:
        print(f"  - {path}")
    sys.exit(1)

print("Readiness check passed")
PY

"$ROOT_DIR/scripts/check-contract-traceability.sh"
"$ROOT_DIR/scripts/check-job-error-registry.sh"

echo "All readiness checks passed"
