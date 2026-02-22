#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PORT="${PORT:-18482}"
BASE_URL="http://127.0.0.1:${PORT}"
COOKIE_JAR="$(mktemp)"
SERVER_LOG="$(mktemp)"
LAST_JOB_ID=""
PROVIDER_ID="${PROVIDER_ID:-hetzner}"
PROVIDER_API_TOKEN="${PROVIDER_API_TOKEN:-${HETZNER_API_TOKEN:-}}"

cleanup() {
  if [[ -n "${SERVER_PID:-}" ]] && kill -0 "${SERVER_PID}" 2>/dev/null; then
    kill "${SERVER_PID}" 2>/dev/null || true
    wait "${SERVER_PID}" 2>/dev/null || true
  fi
  rm -f "${COOKIE_JAR}" "${SERVER_LOG}"
}
trap cleanup EXIT

fail() {
  local reason="$1"
  echo "[smoke] FAIL: ${reason}" >&2
  if [[ -n "${LAST_JOB_ID}" ]]; then
    echo "[smoke] job_id=${LAST_JOB_ID}" >&2
    curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/jobs/${LAST_JOB_ID}" >&2 || true
    echo >&2
  fi
  echo "--- server log ---" >&2
  cat "${SERVER_LOG}" >&2
  exit 1
}

json_get() {
  local json="$1"
  local expr="$2"
  python3 - <<'PY' "${json}" "${expr}"
import json, sys
data = json.loads(sys.argv[1])
expr = sys.argv[2]
print(eval(expr, {"__builtins__": {}}, {"data": data}))
PY
}

wait_job_terminal() {
  local job_id="$1"
  LAST_JOB_ID="${job_id}"
  local status=""
  for _ in $(seq 1 90); do
    local job_json
    job_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/jobs/${job_id}")" || fail "failed command: GET /api/jobs/${job_id}"
    status="$(json_get "${job_json}" "data['status']")"
    if [[ "${status}" == "succeeded" || "${status}" == "failed" ]]; then
      echo "${job_json}"
      return 0
    fi
    sleep 2
  done
  fail "timeout waiting for job terminal state"
}

ensure_provider_node_ready() {
  if [[ -z "${PROVIDER_API_TOKEN}" ]]; then
    fail "missing provider token: set PROVIDER_API_TOKEN (or HETZNER_API_TOKEN)"
  fi

  echo "[smoke] connecting provider id=${PROVIDER_ID}"
  local provider_connect_resp provider_connect_status provider_connect_body
  provider_connect_resp="$(curl -sS -b "${COOKIE_JAR}" -H "Content-Type: application/json" -d "{\"provider_id\":\"${PROVIDER_ID}\",\"api_token\":\"${PROVIDER_API_TOKEN}\"}" -w "\n%{http_code}" "${BASE_URL}/api/providers")"
  provider_connect_status="${provider_connect_resp##*$'\n'}"
  provider_connect_body="${provider_connect_resp%$'\n'*}"
  if [[ "${provider_connect_status}" != "200" ]]; then
    fail "failed command: POST /api/providers status=${provider_connect_status} body=${provider_connect_body}"
  fi

  local nodes_json ready_state
  nodes_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/nodes")" || fail "failed command: GET /api/nodes"
  ready_state="$(python3 - <<'PY' "${nodes_json}"
import json, sys
items = json.loads(sys.argv[1])
provider_nodes = [n for n in items if n.get('is_local') is False]
if not provider_nodes:
    print('missing')
    raise SystemExit(0)
ready = next((n for n in provider_nodes if n.get('readiness', {}).get('is_ready')), None)
if ready:
    print('ready')
else:
    reasons = ",".join(provider_nodes[0].get('readiness', {}).get('reason_codes', []))
    print('not_ready:' + reasons)
PY
)"
  if [[ "${ready_state}" == "ready" ]]; then
    echo "[smoke] provider-backed node already ready"
    return 0
  fi

  echo "[smoke] creating provider-backed node"
  local create_response create_status create_body
  create_response="$(curl -sS -b "${COOKIE_JAR}" -H "Content-Type: application/json" -d "{\"provider_id\":\"${PROVIDER_ID}\",\"name\":\"smoke-${PROVIDER_ID}-node\"}" -w "\n%{http_code}" "${BASE_URL}/api/nodes")"
  create_status="${create_response##*$'\n'}"
  create_body="${create_response%$'\n'*}"
  if [[ "${create_status}" != "202" ]]; then
    fail "failed command: POST /api/nodes status=${create_status} body=${create_body}"
  fi

  local node_job_id node_job_json node_job_status
  node_job_id="$(json_get "${create_body}" "data['job_id']")"
  node_job_json="$(wait_job_terminal "${node_job_id}")"
  node_job_status="$(json_get "${node_job_json}" "data['status']")"
  if [[ "${node_job_status}" != "succeeded" ]]; then
    fail "provider node provision job failed"
  fi

  echo "[smoke] waiting for provider-backed node readiness"
  for _ in $(seq 1 30); do
    local nodes_json ready
    nodes_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/nodes")" || fail "failed command: GET /api/nodes"
    ready="$(python3 - <<'PY' "${nodes_json}"
import json, sys
items = json.loads(sys.argv[1])
provider_nodes = [n for n in items if n.get('is_local') is False]
if not provider_nodes:
    print("missing")
elif any(n.get('readiness', {}).get('is_ready') for n in provider_nodes):
    print("ready")
else:
    reasons = ",".join(provider_nodes[0].get('readiness', {}).get('reason_codes', []))
    print("not_ready:" + reasons)
PY
)"
    if [[ "${ready}" == "ready" ]]; then
      return 0
    fi
    sleep 2
  done
  fail "provider-backed node readiness did not become ready"
}

curl_ok() {
  local url="$1"
  local status
  status="$(curl -s -o /dev/null -w "%{http_code}" "${url}" || true)"
  [[ "${status}" == "200" || "${status}" == "301" || "${status}" == "302" ]]
}

GO_BIN="${GO_BIN:-}"
if [[ -z "${GO_BIN}" ]]; then
  if command -v go >/dev/null 2>&1; then
    GO_BIN="$(command -v go)"
  elif [[ -x "/usr/local/go/bin/go" ]]; then
    GO_BIN="/usr/local/go/bin/go"
  else
    fail "failed command: resolve go binary"
  fi
fi

echo "[smoke] building pressluft binary"
"${GO_BIN}" build -o "${ROOT_DIR}/bin/pressluft" ./cmd/pressluft

echo "[smoke] starting dev server on ${BASE_URL}"
"${ROOT_DIR}/bin/pressluft" dev --port "${PORT}" >"${SERVER_LOG}" 2>&1 &
SERVER_PID=$!

for _ in $(seq 1 30); do
  if curl -fsS "${BASE_URL}/" >/dev/null 2>&1; then
    break
  fi
  sleep 1
done
curl -fsS "${BASE_URL}/" >/dev/null 2>&1 || fail "failed command: GET /"

curl -fsS -c "${COOKIE_JAR}" -H "Content-Type: application/json" -d '{"email":"admin@pressluft.local","password":"pressluft-dev-password"}' "${BASE_URL}/api/login" >/dev/null || fail "failed command: POST /api/login"

ensure_provider_node_ready

site_slug="smoke-restore-$(date +%s)"
site_resp="$(curl -sS -b "${COOKIE_JAR}" -H "Content-Type: application/json" -d "{\"name\":\"Smoke Restore Site\",\"slug\":\"${site_slug}\"}" -w "\n%{http_code}" "${BASE_URL}/api/sites")"
site_status="${site_resp##*$'\n'}"
site_body="${site_resp%$'\n'*}"
if [[ "${site_status}" == "409" ]]; then
  fail "site create preflight blocked with 409: ${site_body}"
fi
if [[ "${site_status}" != "202" ]]; then
  fail "failed command: POST /api/sites status=${site_status} body=${site_body}"
fi
site_resp="${site_body}"
site_job_id="$(json_get "${site_resp}" "data['job_id']")"
site_job_json="$(wait_job_terminal "${site_job_id}")"
if [[ "$(json_get "${site_job_json}" "data['status']")" != "succeeded" ]]; then
  fail "site create job failed"
fi

sites_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/sites")" || fail "failed command: GET /api/sites"
site_id="$(python3 - <<'PY' "${sites_json}" "${site_slug}"
import json, sys
items = json.loads(sys.argv[1])
slug = sys.argv[2]
site = next((s for s in items if s.get('slug') == slug), None)
if not site:
    raise SystemExit(1)
print(site['id'])
PY
)" || fail "failed to resolve created site id"

envs_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/sites/${site_id}/environments")" || fail "failed command: GET /api/sites/${site_id}/environments"
env_id="$(python3 - <<'PY' "${envs_json}"
import json, sys
items = json.loads(sys.argv[1])
prod = next((e for e in items if e.get('environment_type') == 'production'), None)
if not prod:
    raise SystemExit(1)
print(prod['id'])
PY
)" || fail "failed to resolve production environment"
preview_url="$(python3 - <<'PY' "${envs_json}" "${env_id}"
import json, sys
items = json.loads(sys.argv[1])
eid = sys.argv[2]
env = next((e for e in items if e.get('id') == eid), None)
if not env:
    raise SystemExit(1)
print(env['preview_url'])
PY
)" || fail "failed to resolve preview URL"

backup_resp="$(curl -fsS -b "${COOKIE_JAR}" -H "Content-Type: application/json" -d '{"backup_scope":"full"}' "${BASE_URL}/api/environments/${env_id}/backups")" || fail "failed command: POST /api/environments/${env_id}/backups"
backup_job_id="$(json_get "${backup_resp}" "data['job_id']")"
backup_job_json="$(wait_job_terminal "${backup_job_id}")"
if [[ "$(json_get "${backup_job_json}" "data['status']")" != "succeeded" ]]; then
  fail "backup create job failed"
fi

backups_json="$(curl -fsS -b "${COOKIE_JAR}" "${BASE_URL}/api/environments/${env_id}/backups")" || fail "failed command: GET /api/environments/${env_id}/backups"
backup_id="$(python3 - <<'PY' "${backups_json}"
import json, sys
items = json.loads(sys.argv[1])
completed = next((b for b in items if b.get('status') == 'completed' and b.get('backup_scope') == 'full'), None)
if not completed:
    raise SystemExit(1)
if not completed.get('checksum') or completed.get('size_bytes', 0) <= 0:
    raise SystemExit(1)
print(completed['id'])
PY
)" || fail "failed to resolve completed full backup with metadata"

restore_resp="$(curl -fsS -b "${COOKIE_JAR}" -H "Content-Type: application/json" -d "{\"backup_id\":\"${backup_id}\"}" "${BASE_URL}/api/environments/${env_id}/restore")" || fail "failed command: POST /api/environments/${env_id}/restore"
restore_job_id="$(json_get "${restore_resp}" "data['job_id']")"
restore_job_json="$(wait_job_terminal "${restore_job_id}")"
if [[ "$(json_get "${restore_job_json}" "data['status']")" != "succeeded" ]]; then
  fail "restore job failed"
fi

for _ in $(seq 1 20); do
  if curl_ok "${preview_url}"; then
    echo "[smoke] success: restore completed and preview reachable"
    exit 0
  fi
  sleep 3
done

fail "post-restore preview URL unreachable"
