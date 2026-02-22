#!/usr/bin/env bash
set -euo pipefail

failed=0

check() {
  local label="$1"
  local command="$2"
  local remediation="$3"

  if bash -lc "${command}" >/dev/null 2>&1; then
    echo "[ok] ${label}"
    return 0
  fi

  echo "[fail] ${label}"
  echo "       remediation: ${remediation}"
  failed=1
}

echo "[check] local acquisition prerequisites"
check "multipass installed (command -v multipass)" \
  "command -v multipass" \
  "install multipass and ensure 'multipass' is on PATH"

if [[ ${failed} -ne 0 ]]; then
  echo "[result] missing prerequisites for Wave 5 acquired-node smoke success paths"
  exit 1
fi

echo "[result] all required local runtime prerequisites are available"
