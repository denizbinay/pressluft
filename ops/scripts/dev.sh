#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
DEV_API_PORT=${DEV_API_PORT:-8081}
DEV_UI_PORT=${DEV_UI_PORT:-8080}
DEV_UI_HOST=${DEV_UI_HOST:-0.0.0.0}
WEB_DIR=${WEB_DIR:-"$ROOT_DIR/web"}
GO_CMD=${GO:-go}
NPM_CMD=${NPM:-npm}
DEV_WORKFLOW=${DEV_WORKFLOW:-dev}

CLOUDFLARED_PID=""
GO_PID=""
TAIL_PID=""
BACKEND_LOG=""
TUNNEL_LOG=""

cleanup() {
  if [ -n "$GO_PID" ]; then
    kill "$GO_PID" 2>/dev/null || true
  fi
  if [ -n "$TAIL_PID" ]; then
    kill "$TAIL_PID" 2>/dev/null || true
  fi
  if [ -n "$CLOUDFLARED_PID" ]; then
    kill "$CLOUDFLARED_PID" 2>/dev/null || true
  fi
  if [ -n "$BACKEND_LOG" ] && [ -f "$BACKEND_LOG" ]; then
    rm -f "$BACKEND_LOG"
  fi
  if [ -n "$TUNNEL_LOG" ] && [ -f "$TUNNEL_LOG" ]; then
    rm -f "$TUNNEL_LOG"
  fi
}

trap cleanup EXIT INT TERM

case "${1:-}" in
  -h|--help|help)
    printf '%s\n' 'pressluft dev script'
    printf '%s\n' '  DEV_API_PORT    Backend port (default: 8081)'
    printf '%s\n' '  DEV_UI_PORT     Nuxt port (default: 8080)'
    printf '%s\n' '  DEV_UI_HOST     Nuxt host (default: 0.0.0.0)'
    printf '%s\n' '  NUXT_DEV_PROXY_TARGET'
    printf '%s\n' '                 Optional Nuxt dev proxy target (default: http://localhost:${DEV_API_PORT}/api)'
    printf '%s\n' '  PRESSLUFT_CONTROL_PLANE_URL'
    printf '%s\n' '                 Stable public URL; required for DEV_WORKFLOW=lab, optional for dev'
    printf '%s\n' '  DEV_WORKFLOW    dev (default) or lab'
    exit 0
    ;;
esac

run_devctl() {
  PORT="$DEV_API_PORT" PRESSLUFT_CONTROL_PLANE_URL="${PRESSLUFT_CONTROL_PLANE_URL:-}" "$GO_CMD" run ./cmd/pressluft-devctl "$@"
}

start_quick_tunnel() {
  if ! command -v cloudflared >/dev/null 2>&1; then
    echo "cloudflared not found. Install it or set PRESSLUFT_CONTROL_PLANE_URL to a stable public URL."
    exit 1
  fi

  TUNNEL_LOG=$(mktemp)
  cloudflared tunnel --url "http://localhost:${DEV_API_PORT}" --no-autoupdate --logfile "$TUNNEL_LOG" --loglevel info >/dev/null 2>&1 &
  CLOUDFLARED_PID=$!

  for _ in $(seq 1 30); do
    TUNNEL_URL=$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$TUNNEL_LOG" | head -n 1 || true)
    if [ -n "$TUNNEL_URL" ]; then
      export PRESSLUFT_CONTROL_PLANE_URL="$TUNNEL_URL"
      break
    fi
    sleep 1
  done

  if [ -z "${PRESSLUFT_CONTROL_PLANE_URL:-}" ]; then
    echo "Failed to obtain Cloudflare tunnel URL."
    echo "Check log: $TUNNEL_LOG"
    exit 1
  fi

  echo "Cloudflare tunnel: $PRESSLUFT_CONTROL_PLANE_URL"
}

print_workflow_banner() {
  case "$DEV_WORKFLOW" in
    lab)
      echo "Workflow: dev-lab"
      echo "Callback durability: stable public URL required"
      echo "Remote connectivity: durable reconnect expected"
      ;;
    *)
      echo "Workflow: dev"
      echo "Remote connectivity: session-scoped"
      if [ -n "${PRESSLUFT_CONTROL_PLANE_URL:-}" ] && echo "$PRESSLUFT_CONTROL_PLANE_URL" | grep -q '\.trycloudflare\.com'; then
        echo "WARNING: Cloudflare quick tunnels are ephemeral. Remote agents configured against this URL will not reconnect after control-plane restart."
      fi
      ;;
  esac
}

start_backend() {
  BACKEND_LOG=$(mktemp)
  PORT="$DEV_API_PORT" PRESSLUFT_CONTROL_PLANE_URL="$PRESSLUFT_CONTROL_PLANE_URL" "$GO_CMD" run -tags dev ./cmd >"$BACKEND_LOG" 2>&1 &
  GO_PID=$!

  for _ in $(seq 1 20); do
    if ! kill -0 "$GO_PID" 2>/dev/null; then
      return 1
    fi
    if grep -q "pressluft listening" "$BACKEND_LOG"; then
      tail -n +1 -f "$BACKEND_LOG" &
      TAIL_PID=$!
      return 0
    fi
    sleep 1
  done

  if kill -0 "$GO_PID" 2>/dev/null; then
    tail -n +1 -f "$BACKEND_LOG" &
    TAIL_PID=$!
    return 0
  fi
  return 1
}

show_backend_failure() {
  if [ -n "$BACKEND_LOG" ] && [ -f "$BACKEND_LOG" ]; then
    cat "$BACKEND_LOG"
  fi
  if [ -n "$BACKEND_LOG" ] && grep -q "address already in use" "$BACKEND_LOG"; then
    echo "Go backend failed to start because port $DEV_API_PORT is already in use."
    echo "Choose another port, for example: make $([ "$DEV_WORKFLOW" = "lab" ] && printf '%s' 'dev-lab' || printf '%s' 'dev') DEV_API_PORT=8082"
    return
  fi
  echo "Go backend failed during startup."
  echo "Run make dev-status to inspect local state. If the state bundle is disposable, reset it with make dev-reset CONFIRM=1."
}

case "$DEV_WORKFLOW" in
  dev)
    if [ -z "${PRESSLUFT_CONTROL_PLANE_URL:-}" ]; then
      start_quick_tunnel
    fi
    ;;
  lab)
    if [ -z "${PRESSLUFT_CONTROL_PLANE_URL:-}" ]; then
      echo "dev-lab requires PRESSLUFT_CONTROL_PLANE_URL with a stable public URL."
      run_devctl status
      exit 1
    fi
    ;;
  *)
    echo "Unsupported DEV_WORKFLOW: $DEV_WORKFLOW"
    exit 1
    ;;
esac

print_workflow_banner

if ! run_devctl preflight --workflow="$DEV_WORKFLOW"; then
  exit 1
fi

if ! start_backend; then
  show_backend_failure
  exit 1
fi

NUXT_DEV_PROXY_TARGET="${NUXT_DEV_PROXY_TARGET:-http://localhost:${DEV_API_PORT}/api}" \
  NUXT_HOST="$DEV_UI_HOST" \
  NUXT_PORT="$DEV_UI_PORT" \
  "$NPM_CMD" --prefix "$WEB_DIR" run dev
