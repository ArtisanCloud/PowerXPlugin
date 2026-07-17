#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

status=0

scan() {
  local label="$1"
  local pattern="$2"
  shift 2
  local matches
  matches="$(rg -n "$pattern" "$@" 2>/dev/null || true)"
  if [[ -n "$matches" ]]; then
    echo "[realtime-transport] forbidden ${label}:"
    echo "$matches"
    status=1
  fi
}

scan "frontend protocol primitives" \
  'new EventSource|new WebSocket|body\.getReader\(' \
  -g '!**/composables/api/useStream.ts' \
  skeleton/web-admin/nuxt/app

scan "backend gin-contrib/sse usage" \
  'gin-contrib/sse|github\.com/gin-contrib/sse' \
  skeleton/backend/go-gin/internal skeleton/backend/go-gin/go.mod

if [[ "$status" -ne 0 ]]; then
  echo "[realtime-transport] FAILED"
  exit "$status"
fi

echo "[realtime-transport] PASSED"
