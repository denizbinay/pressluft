#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT_DIR" <<'PY'
import os
import re
import sys

root = sys.argv[1]
openapi_path = os.path.join(root, "contracts", "openapi.yaml")
traceability_path = os.path.join(root, "docs", "contract-traceability.md")

with open(openapi_path, "r", encoding="utf-8") as f:
    lines = f.readlines()

methods = {"get", "post", "put", "patch", "delete", "head", "options"}
path_pattern = re.compile(r"^\s{2}(/[^\s:]+):\s*$")
method_pattern = re.compile(r"^\s{4}([a-z]+):\s*$")

openapi_endpoints = set()
current_path = None
for line in lines:
    p = path_pattern.match(line)
    if p:
        current_path = p.group(1)
        continue
    m = method_pattern.match(line)
    if m and current_path:
        method = m.group(1)
        if method in methods:
            openapi_endpoints.add(f"{method.upper()} /api{current_path}")

with open(traceability_path, "r", encoding="utf-8") as f:
    tlines = f.readlines()

row_pattern = re.compile(r"\|\s*`([A-Z]+\s+/api/[^`]+)`\s*\|")
traceability_rows = []
for line in tlines:
    match = row_pattern.search(line)
    if match:
        traceability_rows.append(match.group(1))

traceability_endpoints = set(traceability_rows)

duplicates = sorted({ep for ep in traceability_rows if traceability_rows.count(ep) > 1})
missing_in_trace = sorted(openapi_endpoints - traceability_endpoints)
extra_in_trace = sorted(traceability_endpoints - openapi_endpoints)

if duplicates or missing_in_trace or extra_in_trace:
    print("Contract traceability check failed")
    if duplicates:
        print("Duplicate traceability entries:")
        for ep in duplicates:
            print(f"  - {ep}")
    if missing_in_trace:
        print("Endpoints missing in traceability matrix:")
        for ep in missing_in_trace:
            print(f"  - {ep}")
    if extra_in_trace:
        print("Traceability entries not found in OpenAPI:")
        for ep in extra_in_trace:
            print(f"  - {ep}")
    sys.exit(1)

print(f"Contract traceability check passed ({len(openapi_endpoints)} endpoints)")
PY
