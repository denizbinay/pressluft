#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

python3 - "$ROOT_DIR" <<'PY'
import datetime as dt
import os
import re
import sys

root = sys.argv[1]
lock_path = os.path.join(root, "coordination", "locks.md")

if not os.path.exists(lock_path):
    print("Parallel lock check failed: missing coordination/locks.md")
    sys.exit(1)

with open(lock_path, "r", encoding="utf-8") as f:
    lines = f.readlines()

active_start = None
reclaimed_start = None
for i, line in enumerate(lines):
    if line.strip() == "## Active Locks":
        active_start = i
    if line.strip() == "## Reclaimed Locks":
        reclaimed_start = i

if active_start is None or reclaimed_start is None or reclaimed_start <= active_start:
    print("Parallel lock check failed: lock sections are missing or malformed")
    sys.exit(1)

table_lines = [ln.strip() for ln in lines[active_start + 1 : reclaimed_start] if ln.strip()]
if len(table_lines) < 2:
    print("Parallel lock check failed: Active Locks table header is missing")
    sys.exit(1)

header = table_lines[0]
separator = table_lines[1]
expected_header = "| lock_id | owner | paths | claimed_at_utc | expected_minutes | status | note |"
if header != expected_header or not separator.startswith("|---"):
    print("Parallel lock check failed: Active Locks header does not match expected format")
    sys.exit(1)

raw_rows = table_lines[2:]
iso_pattern = re.compile(r"^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$")
now = dt.datetime.now(dt.UTC)
stale_rows = []
parsed_statuses = []

for row in raw_rows:
    if set(row) == {"|", "-"}:
        continue
    parts = [p.strip() for p in row.strip("|").split("|")]
    if len(parts) != 7:
        print(f"Parallel lock check failed: malformed row -> {row}")
        sys.exit(1)

    lock_id, owner, paths, claimed_at_utc, expected_minutes, status, note = parts
    if not lock_id or not owner or not paths or not status:
        print(f"Parallel lock check failed: required fields missing -> {row}")
        sys.exit(1)
    if status not in {"active", "released"}:
        print(f"Parallel lock check failed: invalid status '{status}' in row -> {row}")
        sys.exit(1)
    if not expected_minutes.isdigit() or int(expected_minutes) <= 0:
        print(f"Parallel lock check failed: expected_minutes must be a positive integer -> {row}")
        sys.exit(1)
    if not iso_pattern.match(claimed_at_utc):
        print(f"Parallel lock check failed: claimed_at_utc must be ISO-8601 UTC -> {row}")
        sys.exit(1)

    claimed_at = dt.datetime.strptime(claimed_at_utc, "%Y-%m-%dT%H:%M:%SZ").replace(tzinfo=dt.UTC)
    age_minutes = (now - claimed_at).total_seconds() / 60
    if status == "active" and age_minutes > 120:
        stale_rows.append((lock_id, owner, int(age_minutes), note))

    parsed_statuses.append(status)

if stale_rows:
    print("Parallel lock check failed: stale active locks detected (>120 minutes)")
    for lock_id, owner, age_minutes, note in stale_rows:
        print(f"  - {lock_id} owned by {owner}, age={age_minutes}m, note={note}")
    sys.exit(1)

total_rows = len(parsed_statuses)
active_rows = sum(1 for s in parsed_statuses if s == "active")
print(f"Parallel lock check passed (rows={total_rows}, active={active_rows})")
PY
