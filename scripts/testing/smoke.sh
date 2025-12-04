#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

start_ts=$(date +%s)
echo "=== Smoke workflow start ==="

mkdir -p tmp

publish_server_pid=""
MOCK_SERVER_SCRIPT=""
# Provide deterministic tenant UUID for publish flow; override via SMOKE_TENANT_UUID if needed
SMOKE_TENANT_UUID="${SMOKE_TENANT_UUID:-00000000-0000-0000-0000-000000000001}"
cleanup() {
  if [ -n "$publish_server_pid" ] && kill -0 "$publish_server_pid" 2>/dev/null; then
    kill "$publish_server_pid" >/dev/null 2>&1 || true
    wait "$publish_server_pid" 2>/dev/null || true
  fi
  if [ "${KEEP_TEMP_DIR:-}" != "1" ] && [ -n "${SMOKE_DIR:-}" ] && [ -d "$SMOKE_DIR" ]; then
    rm -rf "$SMOKE_DIR"
  fi
  if [ "${KEEP_TEMP_DIR:-}" != "1" ] && [ -n "$MOCK_SERVER_SCRIPT" ] && [ -f "$MOCK_SERVER_SCRIPT" ]; then
    rm -f "$MOCK_SERVER_SCRIPT"
  fi
}
trap cleanup EXIT

# 1. Core Go checks
echo "[1/5] Running framework bootstrap tests"
go test ./framework/backend/go/bootstrap/... -coverprofile=tmp/coverage.out

echo "[2/5] Running skeleton router tests"
go test ./skeleton/backend/internal/router/... -v

echo "Generating coverage report tmp/coverage.html"
go tool cover -html=tmp/coverage.out -o tmp/coverage.html

# 2. Contract validation
echo "[3/5] Validating contracts"
./scripts/testing/validate-contracts.sh

# 3. CLI scaffold sanity
echo "[4/5] Building px-plugin and scaffolding smoke project"
PX_PLUGIN_BIN="${PX_PLUGIN_BIN:-$ROOT_DIR/bin/px-plugin}"
# Always rebuild CLI to ensure latest flags/features are present
go build -o "$PX_PLUGIN_BIN" ./tools/cli/cmd/px-plugin
SMOKE_DIR=$(mktemp -d)
if [ "${KEEP_TEMP_DIR:-}" = "1" ]; then
  echo "Keeping smoke temp dir: $SMOKE_DIR"
fi
pushd "$SMOKE_DIR" > /dev/null
"$PX_PLUGIN_BIN" init com.powerx.smoke --force --module github.com/example/smoke > /dev/null
if [ ! -f com.powerx.smoke/plugin.yaml ]; then
  echo "Error: CLI scaffold missing plugin.yaml" >&2
  exit 1
fi
popd > /dev/null

echo "[5/5] Validating CLI package/publish workflow"
SMOKE_PLUGIN_DIR="$SMOKE_DIR/mock-plugin"
mkdir -p "$SMOKE_PLUGIN_DIR/docs/contracts"
cat > "$SMOKE_PLUGIN_DIR/plugin.yaml" <<'YAML'
id: com.powerx.smoke
version: 0.1.0
backend:
  entry: backend/cmd/plugin
YAML
cat > "$SMOKE_PLUGIN_DIR/docs/contracts/manifest.json" <<'JSON'
{
  "name": "CLI Smoke",
  "version": "0.1.0",
  "routes": [],
  "menus": {}
}
JSON
cat > "$SMOKE_PLUGIN_DIR/docs/contracts/rbac.json" <<'JSON'
{
  "resources": [],
  "policies": []
}
JSON

"$PX_PLUGIN_BIN" package --entry "$SMOKE_PLUGIN_DIR" --skip-frontend --skip-backend > tmp/package-smoke.log

MOCK_PORT=$(python3 - <<'PY'
import socket
sock = socket.socket()
sock.bind(("127.0.0.1", 0))
port = sock.getsockname()[1]
sock.close()
print(port)
PY
)
MOCK_LOG="tmp/mock_publish_server.log"
MOCK_SERVER_SCRIPT=$(mktemp)
cat > "$MOCK_SERVER_SCRIPT" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        length = int(self.headers.get("content-length", "0") or "0")
        if length:
            self.rfile.read(length)
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.end_headers()
        payload = {
            "code": 200,
            "message": "success",
            "data": {
                "publishId": "SMOKE-TEST",
                "reviewUrl": "http://127.0.0.1/mock-review",
                "status": "pending",
            },
        }
        self.wfile.write(json.dumps(payload).encode("utf-8"))

    def log_message(self, *_):
        return


if __name__ == "__main__":
    port = int(sys.argv[1])
    HTTPServer(("127.0.0.1", port), Handler).serve_forever()
PY
python3 "$MOCK_SERVER_SCRIPT" "$MOCK_PORT" > "$MOCK_LOG" 2>&1 &
publish_server_pid=$!
sleep 1
PX_PUBLISH_API_TOKEN="smoke-token" "$PX_PLUGIN_BIN" publish \
  --entry "$SMOKE_PLUGIN_DIR" \
  --channel smoke \
  --notes "smoke test" \
  --tenant "$SMOKE_TENANT_UUID" \
  --publish-api "http://127.0.0.1:${MOCK_PORT}" \
  --publish-token "smoke-token" > /dev/null
kill "$publish_server_pid" >/dev/null 2>&1 || true
wait "$publish_server_pid" 2>/dev/null || true
publish_server_pid=""

end_ts=$(date +%s)
elapsed=$((end_ts - start_ts))
echo "=== Smoke workflow complete in ${elapsed}s ==="
