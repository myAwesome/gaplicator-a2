#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"

BRIDGE_HOST="${BRIDGE_HOST:-127.0.0.1}"
BRIDGE_PORT="${BRIDGE_PORT:-8787}"
WEB_PORT="${WEB_PORT:-5173}"

cleanup() {
  if [ -n "${BRIDGE_PID:-}" ]; then
    kill "$BRIDGE_PID" 2>/dev/null || true
  fi
  if [ -n "${WEB_PID:-}" ]; then
    kill "$WEB_PID" 2>/dev/null || true
  fi
}

trap cleanup INT TERM EXIT

echo "▸ Starting bridge on http://${BRIDGE_HOST}:${BRIDGE_PORT}..."
GOTOOLCHAIN=local go run . serve --host "$BRIDGE_HOST" --port "$BRIDGE_PORT" &
BRIDGE_PID=$!

echo "▸ Starting web app on http://127.0.0.1:${WEB_PORT}..."
(cd web && npm run dev -- --host 127.0.0.1 --port "$WEB_PORT") &
WEB_PID=$!

wait "$BRIDGE_PID" "$WEB_PID"
