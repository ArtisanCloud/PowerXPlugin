#!/usr/bin/env bash
set -euo pipefail

color() {
  local code="$1"; shift
  if [ -t 1 ]; then
    printf '\033[%sm%s\033[0m' "$code" "$*"
  else
    printf '%s' "$*"
  fi
}

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

start_ts=$(date +%s)
echo "=== Regression workflow start ==="

# Run smoke workflow first
./scripts/testing/smoke.sh

echo "[R-1] Running full Go test suite"
go test ./framework/... ./skeleton/backend/... -coverprofile=tmp/coverage-regression.out
go tool cover -html=tmp/coverage-regression.out -o tmp/coverage.html

BACKEND_PORT="${REGRESSION_BACKEND_PORT:-8078}"
BACKEND_HOST="${REGRESSION_BACKEND_HOST:-127.0.0.1}"
BACKEND_BASE_URL="http://${BACKEND_HOST}:${BACKEND_PORT}"
API_BASE_URL="${BACKEND_BASE_URL}/api/v1"
echo "[info] Backend base URL: ${API_BASE_URL}"

BACKEND_LOG="tmp/regression-backend.log"
FRONTEND_LOG="tmp/regression-frontend.log"
if [ -n "${REGRESSION_FRONTEND_PORT:-}" ]; then
  FRONTEND_PORT="$REGRESSION_FRONTEND_PORT"
else
  FRONTEND_PORT=$(python3 - <<'PY'
import socket
sock = socket.socket()
sock.bind(('', 0))
port = sock.getsockname()[1]
sock.close()
print(port)
PY
  )
fi
echo "[info] Frontend port: $FRONTEND_PORT"
mkdir -p tmp

# ensure local services bypass proxies
export http_proxy=""
export https_proxy=""
export HTTP_PROXY=""
export HTTPS_PROXY=""
export NO_PROXY="127.0.0.1,localhost"

backend_pid=""
frontend_pid=""
cleanup() {
  if [ -n "$frontend_pid" ] && kill -0 "$frontend_pid" 2>/dev/null; then
    kill "$frontend_pid" >/dev/null 2>&1 || true
    wait "$frontend_pid" 2>/dev/null || true
  fi
  if [ -n "$backend_pid" ] && kill -0 "$backend_pid" 2>/dev/null; then
    kill "$backend_pid" >/dev/null 2>&1 || true
    wait "$backend_pid" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "[R-2] Starting backend service"
POWERX_LISTEN=":${BACKEND_PORT}" go run ./skeleton/backend/cmd/plugin >"$BACKEND_LOG" 2>&1 &
backend_pid=$!

echo "[R-3] Preparing frontend dependencies"
pushd skeleton/web-admin > /dev/null
npm install
npm run lint
NUXT_PUBLIC_API_BASE="$API_BASE_URL" npm run build
NUXT_PUBLIC_API_BASE="$API_BASE_URL" NITRO_PORT="$FRONTEND_PORT" PORT="$FRONTEND_PORT" \
  npx nuxi preview --hostname 127.0.0.1 --port "$FRONTEND_PORT" >"$ROOT_DIR/$FRONTEND_LOG" 2>&1 &
frontend_pid=$!
popd > /dev/null

wait_for() {
  local url="$1"
  local name="$2"
  local retries=30
  local wait=2
  for ((i=0; i<retries; i++)); do
    status=$(curl --noproxy '*' -s -o /dev/null -w "%{http_code}" "$url" || echo "000")
    if [[ "$status" =~ ^(200|204|301|302|404)$ ]]; then
      echo "${name} ready (HTTP $status): $url"
      return 0
    fi
    sleep "$wait"
  done
  echo "Error: ${name} not ready after $((retries*wait)) seconds (see logs)" >&2
  return 1
}

wait_for "${BACKEND_BASE_URL}/healthz" "Backend"
PLAYWRIGHT_BASE_URL="${PLAYWRIGHT_BASE_URL:-http://127.0.0.1:$FRONTEND_PORT}"
wait_for "$PLAYWRIGHT_BASE_URL" "Frontend"

echo "[R-4] Running Playwright tests against $PLAYWRIGHT_BASE_URL"
(
  cd skeleton/web-admin
  PLAYWRIGHT_BASE_URL="$PLAYWRIGHT_BASE_URL" NUXT_PUBLIC_API_BASE="$API_BASE_URL" npx playwright test
)

echo "Playwright report directory: skeleton/web-admin/test-results/"

echo "Logs stored at $BACKEND_LOG and $FRONTEND_LOG"

end_ts=$(date +%s)
elapsed=$((end_ts - start_ts))
echo "=== Regression workflow complete in ${elapsed}s ==="
