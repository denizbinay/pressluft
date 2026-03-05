#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
DEV_API_PORT=${DEV_API_PORT:-8081}
DEV_UI_PORT=${DEV_UI_PORT:-8080}
DEV_UI_HOST=${DEV_UI_HOST:-0.0.0.0}
WEB_DIR=${WEB_DIR:-"$ROOT_DIR/web"}
GO_CMD=${GO:-go}
NPM_CMD=${NPM:-npm}

CLOUDFLARED_PID=""
GO_PID=""

cleanup() {
  if [ -n "$GO_PID" ]; then
    kill "$GO_PID" 2>/dev/null || true
  fi
  if [ -n "$CLOUDFLARED_PID" ]; then
    kill "$CLOUDFLARED_PID" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

if [ -z "${PRESSLUFT_CONTROL_PLANE_URL:-}" ]; then
  if ! command -v cloudflared >/dev/null 2>&1; then
    echo "cloudflared not found. Install it or set PRESSLUFT_CONTROL_PLANE_URL to a tunnel URL."
    exit 1
  fi

  LOG_FILE=$(mktemp)
  cloudflared tunnel --url "http://localhost:${DEV_API_PORT}" --no-autoupdate --logfile "$LOG_FILE" --loglevel info >/dev/null 2>&1 &
  CLOUDFLARED_PID=$!

  for _ in $(seq 1 30); do
    TUNNEL_URL=$(grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' "$LOG_FILE" | head -n 1 || true)
    if [ -n "$TUNNEL_URL" ]; then
      export PRESSLUFT_CONTROL_PLANE_URL="$TUNNEL_URL"
      break
    fi
    sleep 1
  done

  if [ -z "${PRESSLUFT_CONTROL_PLANE_URL:-}" ]; then
    echo "Failed to obtain Cloudflare tunnel URL."
    echo "Check log: $LOG_FILE"
    exit 1
  fi

  echo "Cloudflare tunnel: $PRESSLUFT_CONTROL_PLANE_URL"
fi

PORT="$DEV_API_PORT" PRESSLUFT_CONTROL_PLANE_URL="$PRESSLUFT_CONTROL_PLANE_URL" "$GO_CMD" run -tags dev ./cmd &
GO_PID=$!
sleep 1
if ! kill -0 "$GO_PID" 2>/dev/null; then
  echo "Go backend failed to start on port $DEV_API_PORT. Stop the process using that port or choose another port (e.g. make dev DEV_API_PORT=8082)."
  exit 1
fi

NUXT_API_BASE="http://localhost:${DEV_API_PORT}/api" NUXT_HOST="$DEV_UI_HOST" NUXT_PORT="$DEV_UI_PORT" "$NPM_CMD" --prefix "$WEB_DIR" run dev
