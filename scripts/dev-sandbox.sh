#!/usr/bin/env bash
set -euo pipefail

# Disposable local runner:
# - Uses temp DB + secrets dir by default
# - Builds Go binary
# - Optionally builds web assets
# - Runs API + worker until Ctrl-C

KEEP="${PRESSLUFT_KEEP_SANDBOX:-0}"
LISTEN_ADDR="${PRESSLUFT_LISTEN_ADDR:-0.0.0.0:18080}"

db_path="${PRESSLUFT_DB_PATH:-}"
secrets_dir="${PRESSLUFT_SECRETS_DIR:-}"

tmp_db=0
tmp_secrets=0
cleaned=0

if [[ -z "$db_path" ]]; then
  db_path="$(mktemp)"
  tmp_db=1
fi

if [[ -z "$secrets_dir" ]]; then
  secrets_dir="$(mktemp -d)"
  tmp_secrets=1
fi

cleanup() {
  if [[ "$cleaned" == "1" ]]; then
    return
  fi
  cleaned=1

  set +e
  terminate_pid() {
    local pid="${1:-}"
    if [[ -z "$pid" ]]; then
      return
    fi
    kill -TERM "$pid" 2>/dev/null || true
    for _ in $(seq 1 20); do
      kill -0 "$pid" 2>/dev/null || return
      sleep 0.1
    done
    kill -KILL "$pid" 2>/dev/null || true
  }

  terminate_pid "${api_pid:-}"
  terminate_pid "${worker_pid:-}"

  if [[ -n "${api_pid:-}" ]]; then wait "$api_pid" 2>/dev/null || true; fi
  if [[ -n "${worker_pid:-}" ]]; then wait "$worker_pid" 2>/dev/null || true; fi

  if [[ "$KEEP" != "1" ]]; then
    if [[ "$tmp_db" == "1" ]]; then rm -f "$db_path" || true; fi
    if [[ "$tmp_secrets" == "1" ]]; then rm -rf "$secrets_dir" || true; fi
  else
    printf 'sandbox kept: PRESSLUFT_DB_PATH=%s PRESSLUFT_SECRETS_DIR=%s\n' "$db_path" "$secrets_dir"
  fi
}
trap cleanup EXIT
trap 'cleanup; exit 130' INT
trap 'cleanup; exit 143' TERM

printf 'building pressluft\n'
go build -o ./bin/pressluft ./cmd/pressluft

if [[ "${PRESSLUFT_SKIP_WEB_BUILD:-0}" != "1" ]]; then
  if [[ ! -f web/.output/public/index.html ]]; then
    printf 'building web assets (pnpm)\n'
    (cd web && pnpm install --frozen-lockfile && pnpm build)
  fi
fi

export PRESSLUFT_DB_PATH="$db_path"
export PRESSLUFT_SECRETS_DIR="$secrets_dir"
export PRESSLUFT_WEB_DIST_DIR="$(pwd)/web/.output/public"
export ANSIBLE_CONFIG="$(pwd)/ansible/ansible.cfg"

printf 'migrating database\n'
./bin/pressluft migrate up

printf 'bootstrapping local node\n'
./bin/pressluft bootstrap

printf 'seeding dev admin (admin@local / 0000)\n'
./bin/pressluft admin init -email admin@local -display-name Admin -password 0000 >/dev/null

printf 'starting api (%s) and worker\n' "$LISTEN_ADDR"
./bin/pressluft -listen "$LISTEN_ADDR" serve &
api_pid=$!

sleep 0.2
if ! kill -0 "$api_pid" 2>/dev/null; then
  printf 'api failed to start (port in use?)\n' >&2
  exit 1
fi

./bin/pressluft worker &
worker_pid=$!

sleep 0.2
if ! kill -0 "$worker_pid" 2>/dev/null; then
  printf 'worker failed to start\n' >&2
  exit 1
fi

port="${LISTEN_ADDR##*:}"
printf 'open: http://127.0.0.1:%s/\n' "$port"
printf 'open: http://localhost:%s/\n' "$port"
printf 'login: admin@local / 0000\n'
printf 'Ctrl-C to stop\n'

wait "$api_pid"
