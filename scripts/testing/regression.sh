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

source "$ROOT_DIR/scripts/testing/go_env.sh"

start_ts=$(date +%s)
echo "=== Regression workflow start ==="

# Run smoke workflow first
./scripts/testing/smoke.sh

echo "[R-1] Running full Go test suite"
go test ./framework/backend/go/... ./skeleton/backend/go-gin/... -coverprofile=tmp/coverage-regression.out
go tool cover -html=tmp/coverage-regression.out -o tmp/coverage.html

BACKEND_HOST="${REGRESSION_BACKEND_HOST:-127.0.0.1}"
backend_port_forced=0
frontend_port_forced=0

pick_free_port() {
  local host="$1"; shift
  local preferred_csv="${1:-}"; shift || true
  local avoid_csv="${1:-}"; shift || true

  python3 - "$host" "$preferred_csv" "$avoid_csv" <<'PY'
import socket
import sys

host = sys.argv[1] if len(sys.argv) > 1 and sys.argv[1] else "127.0.0.1"
preferred_csv = sys.argv[2] if len(sys.argv) > 2 else ""
avoid_csv = sys.argv[3] if len(sys.argv) > 3 else ""

def parse_ports(csv: str) -> list[int]:
  out: list[int] = []
  for raw in (csv or "").split(","):
    raw = raw.strip()
    if not raw:
      continue
    try:
      out.append(int(raw))
    except ValueError:
      continue
  return out

preferred = parse_ports(preferred_csv)
avoid = set(parse_ports(avoid_csv))

def can_bind(port: int) -> bool:
  try:
    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
    s.bind((host, port))
    s.close()
    return True
  except OSError:
    return False

for port in preferred:
  if port in avoid:
    continue
  if can_bind(port):
    print(port)
    raise SystemExit(0)

for _ in range(200):
  s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
  s.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
  s.bind((host, 0))
  port = s.getsockname()[1]
  s.close()
  if port in avoid:
    continue
  if can_bind(port):
    print(port)
    raise SystemExit(0)

raise SystemExit("failed to pick a free port")
PY
}

wait_for() {
  local url="$1"
  local name="$2"
  local retries=30
  local wait=2
  local attempt=0
  local status=""
  for ((attempt=0; attempt<retries; attempt++)); do
    status="$(curl --noproxy '*' -s -o /dev/null -w "%{http_code}" "$url" || echo "000")"
    if [[ "$status" =~ ^([23][0-9][0-9]|404)$ ]]; then
      echo "${name} ready (HTTP $status): $url"
      return 0
    fi
    sleep "$wait"
  done
  echo "Error: ${name} not ready after $((retries*wait)) seconds (last HTTP ${status}, see logs)" >&2
  return 1
}

if [ -n "${REGRESSION_BACKEND_PORT:-}" ]; then
  BACKEND_PORT="$REGRESSION_BACKEND_PORT"
  backend_port_forced=1
else
  # Avoid common dev ports by default; use stable fallback ports first.
  BACKEND_PORT="$(pick_free_port "$BACKEND_HOST" "18078,18079,18080" "8078,8086,3131,3231,3000")"
fi
BACKEND_BASE_URL="http://${BACKEND_HOST}:${BACKEND_PORT}"
API_BASE_URL="${BACKEND_BASE_URL}"
API_PREFIX="/api/v1"
echo "[info] Backend base URL: ${API_BASE_URL}${API_PREFIX}"

BACKEND_LOG="${ROOT_DIR}/tmp/regression-backend.log"
FRONTEND_LOG="${ROOT_DIR}/tmp/regression-frontend.log"
if [ -n "${REGRESSION_FRONTEND_PORT:-}" ]; then
  FRONTEND_PORT="$REGRESSION_FRONTEND_PORT"
  frontend_port_forced=1
elif [ -n "${PLAYWRIGHT_BASE_URL:-}" ]; then
  FRONTEND_PORT="$(python3 - <<'PY'
import os
from urllib.parse import urlparse

u = os.environ.get("PLAYWRIGHT_BASE_URL", "")
p = urlparse(u)
if not p.scheme:
  # tolerate "127.0.0.1:3000" style
  p = urlparse("http://" + u)

if p.port:
  print(p.port)
  raise SystemExit(0)

default = 443 if p.scheme == "https" else 80
print(default)
PY
  )"
else
  FRONTEND_HOST="${REGRESSION_FRONTEND_HOST:-127.0.0.1}"
  FRONTEND_PORT="$(pick_free_port "$FRONTEND_HOST" "13031,13032,13033" "3131,3231,3000,8078,8086")"
fi
echo "[info] Frontend port: $FRONTEND_PORT"
mkdir -p "${ROOT_DIR}/tmp"

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
start_backend() {
  local attempts=8
  local attempt=0
  local backend_cfg=""
  backend_cfg="${ROOT_DIR}/tmp/regression-backend.yaml"
  for ((attempt=1; attempt<=attempts; attempt++)); do
    : >"$BACKEND_LOG"
    cat >"$backend_cfg" <<YAML
server:
  bind_addr: ":${BACKEND_PORT}"
  log_level: "debug"
  dev_mode: true
database:
  driver: "memory"
  dsn: "file:powerxplugin-regression?mode=memory&cache=shared"
grpc_server:
  enable: false
gateway:
  base_url: ""
  tool_token: ""
  refresh_token: ""
  tenant_uuid: ""
  use_mock: []
YAML

    CONFIG_PATH="$backend_cfg" POWERX_BIND_ADDR=":${BACKEND_PORT}" PORT="${BACKEND_PORT}" \
      go run ./skeleton/backend/go-gin/cmd/plugin >"$BACKEND_LOG" 2>&1 &
    backend_pid=$!
    sleep 1
    if kill -0 "$backend_pid" 2>/dev/null; then
      if wait_for "${BACKEND_BASE_URL}/healthz" "Backend"; then
        return 0
      fi
    fi

    # Backend exited early or never became healthy.
    if [ "$backend_port_forced" -eq 1 ]; then
      echo "Backend failed to start on forced port ${BACKEND_PORT}; see ${BACKEND_LOG}" >&2
      return 1
    fi

    if kill -0 "$backend_pid" 2>/dev/null; then
      kill "$backend_pid" >/dev/null 2>&1 || true
      wait "$backend_pid" 2>/dev/null || true
    fi

    BACKEND_PORT="$(pick_free_port "$BACKEND_HOST" "" "8078,8086,3131,3231,3000,${BACKEND_PORT}")"
    BACKEND_BASE_URL="http://${BACKEND_HOST}:${BACKEND_PORT}"
    API_BASE_URL="${BACKEND_BASE_URL}"
    echo "[warn] Backend port occupied; retrying with ${BACKEND_PORT} (attempt ${attempt}/${attempts})"
  done
  echo "Backend failed to start after retries; see ${BACKEND_LOG}" >&2
  return 1
}

start_backend

echo "[R-3] Preparing frontend dependencies"
pushd skeleton/web-admin/nuxt > /dev/null
# act 会把宿主工作区整个拷贝进容器，可能包含不同平台的 node_modules（例如 macOS）。
# 为避免原生依赖（如 lightningcss）在 Linux 容器内加载失败，强制清理并用 lockfile 做干净安装。
rm -rf node_modules .nuxt .output
# 统一在工作区内缓存，避免在容器/不同用户下写入 $HOME 失败或污染宿主缓存。
export npm_config_cache="${ROOT_DIR}/tmp/npm-cache-web-admin"
npm ci --include=optional
# lightningcss 的平台二进制包可能因为 lockfile/跨平台拷贝等原因缺失，导致 Nuxt build 失败。
# 这里按当前平台显式补齐对应的 lightningcss-<platform> 包（不改 package-lock）。
need_lightningcss_pkg=0
node - <<'NODE' || need_lightningcss_pkg=$?
const parts = [process.platform, process.arch]
if (process.platform === 'linux') {
  // best-effort libc detection (glibc vs musl) without extra deps
  const report = typeof process.report?.getReport === 'function' ? process.report.getReport() : undefined
  const glibc = report?.header?.glibcVersionRuntime
  if (process.arch === 'arm') {
    parts.push('gnueabihf')
  } else if (glibc) {
    parts.push('gnu')
  } else {
    parts.push('musl')
  }
} else if (process.platform === 'win32') {
  parts.push('msvc')
}

const pkg = `lightningcss-${parts.join('-')}`
try {
  require.resolve(pkg)
  process.exit(0)
} catch {
  console.log(`[regression] missing ${pkg}; will install it`)
  process.exit(2)
}
NODE

if [ "$need_lightningcss_pkg" -eq 2 ]; then
  platform_suffix="$(node - <<'NODE'
let parts = [process.platform, process.arch]
if (process.platform === 'linux') {
  const report =
    process.report && typeof process.report.getReport === 'function'
      ? process.report.getReport()
      : undefined
  const glibc = report && report.header ? report.header.glibcVersionRuntime : undefined
  if (process.arch === 'arm') {
    parts.push('gnueabihf')
  } else if (glibc) {
    parts.push('gnu')
  } else {
    parts.push('musl')
  }
} else if (process.platform === 'win32') {
  parts.push('msvc')
}
process.stdout.write(parts.join('-'))
NODE
)"
  npm i --no-save --no-package-lock --include=optional "lightningcss-${platform_suffix}"
elif [ "$need_lightningcss_pkg" -ne 0 ]; then
  exit "$need_lightningcss_pkg"
fi
npm run lint
NUXT_PUBLIC_API_BASE="$API_BASE_URL" NUXT_PUBLIC_API_PREFIX="$API_PREFIX" npm run build
FRONTEND_HOST="${REGRESSION_FRONTEND_HOST:-127.0.0.1}"
start_frontend() {
  local attempts=8
  local attempt=0
  for ((attempt=1; attempt<=attempts; attempt++)); do
    : >"$FRONTEND_LOG"
    NUXT_PUBLIC_API_BASE="$API_BASE_URL" NUXT_PUBLIC_API_PREFIX="$API_PREFIX" \
      NITRO_HOST="$FRONTEND_HOST" HOST="$FRONTEND_HOST" \
      NITRO_PORT="$FRONTEND_PORT" PORT="$FRONTEND_PORT" \
      NUXT_TELEMETRY_DISABLED=1 \
      node .output/server/index.mjs >"$FRONTEND_LOG" 2>&1 &
    frontend_pid=$!

    sleep 1
    if ! kill -0 "$frontend_pid" 2>/dev/null; then
      if [ "$frontend_port_forced" -eq 1 ] || [ -n "${PLAYWRIGHT_BASE_URL:-}" ]; then
        echo "Frontend failed to start on port ${FRONTEND_PORT}; see ${FRONTEND_LOG}" >&2
        return 1
      fi
      FRONTEND_PORT="$(pick_free_port "$FRONTEND_HOST" "" "3131,3231,3000,8078,8086,${FRONTEND_PORT}")"
      echo "[warn] Frontend port occupied; retrying with ${FRONTEND_PORT} (attempt ${attempt}/${attempts})"
      continue
    fi

    if wait_for "http://${FRONTEND_HOST}:${FRONTEND_PORT}" "Frontend"; then
      return 0
    fi

    if [ "$frontend_port_forced" -eq 1 ] || [ -n "${PLAYWRIGHT_BASE_URL:-}" ]; then
      echo "Frontend failed to become ready on port ${FRONTEND_PORT}; see ${FRONTEND_LOG}" >&2
      tail -n 80 "$FRONTEND_LOG" >&2 || true
      return 1
    fi

    kill "$frontend_pid" >/dev/null 2>&1 || true
    wait "$frontend_pid" 2>/dev/null || true
    FRONTEND_PORT="$(pick_free_port "$FRONTEND_HOST" "" "3131,3231,3000,8078,8086,${FRONTEND_PORT}")"
    echo "[warn] Frontend not ready; retrying with ${FRONTEND_PORT} (attempt ${attempt}/${attempts})"
  done
  echo "Frontend failed to start after retries; see ${FRONTEND_LOG}" >&2
  return 1
}

start_frontend
popd > /dev/null

if [ -z "${PLAYWRIGHT_BASE_URL:-}" ]; then
  PLAYWRIGHT_BASE_URL="http://${FRONTEND_HOST}:${FRONTEND_PORT}"
fi
if ! curl --noproxy '*' -s -o /dev/null "$PLAYWRIGHT_BASE_URL" 2>/dev/null; then
  alt_base="$(python3 - <<'PY'
import os
from urllib.parse import urlparse, urlunparse

u = os.environ.get("PLAYWRIGHT_BASE_URL", "")
p = urlparse(u)
if not p.scheme:
  p = urlparse("http://" + u)

host = p.hostname or ""
if host == "127.0.0.1":
  netloc = "localhost"
  if p.port:
    netloc = f"{netloc}:{p.port}"
  p = p._replace(netloc=netloc)
  print(urlunparse(p))
PY
  )"
  if [ -n "$alt_base" ] && curl --noproxy '*' -s -o /dev/null "$alt_base" 2>/dev/null; then
    PLAYWRIGHT_BASE_URL="$alt_base"
  fi
fi
wait_for "$PLAYWRIGHT_BASE_URL" "Frontend"

echo "[R-4] Running Playwright tests against $PLAYWRIGHT_BASE_URL"
(
  cd skeleton/web-admin/nuxt
  PLAYWRIGHT_BASE_URL="$PLAYWRIGHT_BASE_URL" \
    NUXT_PUBLIC_API_BASE="$API_BASE_URL" NUXT_PUBLIC_API_PREFIX="$API_PREFIX" \
    node ./node_modules/@playwright/test/cli.js test "$@"
)

echo "Playwright report directory: skeleton/web-admin/nuxt/test-results/"
echo "Logs stored at ${BACKEND_LOG} and ${FRONTEND_LOG}"

end_ts=$(date +%s)
elapsed=$((end_ts - start_ts))
echo "=== Regression workflow complete in ${elapsed}s ==="
