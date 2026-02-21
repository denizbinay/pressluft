#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT_DIR" <<'PY'
import os
import re
import sys

root = sys.argv[1]
trace_path = os.path.join(root, "docs", "contract-traceability.md")
job_types_path = os.path.join(root, "docs", "job-types.md")
errors_path = os.path.join(root, "docs", "error-codes.md")
openapi_path = os.path.join(root, "contracts", "openapi.yaml")

def read(path):
    with open(path, "r", encoding="utf-8") as f:
        return f.read()

trace = read(trace_path)
jobs_doc = read(job_types_path)
errors_doc = read(errors_path)
openapi = read(openapi_path)

trace_job_pattern = re.compile(r"\|\s*`[A-Z]+\s+/api/[^`]+`\s*\|[^\n]*\|\s*async\s*\|\s*`([^`]+)`\s*\|")
trace_jobs = {m.group(1) for m in trace_job_pattern.finditer(trace)}

canonical_job_pattern = re.compile(r"\|\s*`([a-z_]+)`\s*\|\s*(?:site|global \(`site_id = NULL`\))\s*\|")
canonical_jobs = {m.group(1) for m in canonical_job_pattern.finditer(jobs_doc)}

missing_jobs = sorted(trace_jobs - canonical_jobs)

required_generic_codes = {"bad_request", "unauthorized", "not_found", "conflict"}
doc_code_pattern = re.compile(r"\|\s*`([A-Za-z0-9_]+)`\s*\|")
doc_codes = {m.group(1) for m in doc_code_pattern.finditer(errors_doc)}

missing_generic_codes = sorted(required_generic_codes - doc_codes)

openapi_code_pattern = re.compile(r"\bcode:\s*([a-z_]+)")
openapi_codes = {m.group(1) for m in openapi_code_pattern.finditer(openapi)}
missing_openapi_codes = sorted(code for code in openapi_codes if code not in doc_codes)

if missing_jobs or missing_generic_codes or missing_openapi_codes:
    print("Job/error registry check failed")
    if missing_jobs:
        print("Job types referenced by async endpoints but missing in docs/job-types.md:")
        for job in missing_jobs:
            print(f"  - {job}")
    if missing_generic_codes:
        print("Missing generic API error codes in docs/error-codes.md:")
        for code in missing_generic_codes:
            print(f"  - {code}")
    if missing_openapi_codes:
        print("Error codes used in OpenAPI examples but missing in docs/error-codes.md:")
        for code in missing_openapi_codes:
            print(f"  - {code}")
    sys.exit(1)

print(f"Job/error registry check passed ({len(trace_jobs)} async job types, {len(openapi_codes)} OpenAPI example codes)")
PY
