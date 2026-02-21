#!/usr/bin/env bash
set -euo pipefail

ROLLBACK_HINT="Rollback hint: stop pressluft service, remove partial binary/database artifacts, rerun install.sh from clean host checkpoint."

on_error() {
  local exit_code=$?
  printf 'install failed (exit=%s)\n%s\n' "$exit_code" "$ROLLBACK_HINT" >&2
  exit "$exit_code"
}

trap on_error ERR

if [[ -f /etc/os-release ]]; then
  # shellcheck source=/dev/null
  source /etc/os-release
  if [[ "${ID:-}" != "ubuntu" || "${VERSION_ID:-}" != "24.04" ]]; then
    printf 'unsupported OS: expected Ubuntu 24.04, got %s %s\n' "${ID:-unknown}" "${VERSION_ID:-unknown}" >&2
    exit 1
  fi
else
  printf '/etc/os-release missing; cannot validate OS\n' >&2
  exit 1
fi

DB_PATH="${PRESSLUFT_DB_PATH:-./pressluft.db}"

go run ./migrations/migrate.go up
go build -o ./bin/pressluft ./cmd/pressluft
./bin/pressluft -db "$DB_PATH" bootstrap

printf 'install completed successfully\n'
