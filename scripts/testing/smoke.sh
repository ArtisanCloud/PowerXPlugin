#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

start_ts=$(date +%s)
echo "=== Smoke workflow start ==="

mkdir -p tmp

# 1. Core Go checks
echo "[1/4] Running framework bootstrap tests"
go test ./framework/backend/go/bootstrap/... -coverprofile=tmp/coverage.out

echo "[2/4] Running skeleton router tests"
go test ./skeleton/backend/internal/router/... -v

echo "Generating coverage report tmp/coverage.html"
go tool cover -html=tmp/coverage.out -o tmp/coverage.html

# 2. Contract validation
echo "[3/4] Validating contracts"
./scripts/testing/validate-contracts.sh

# 3. CLI scaffold sanity
echo "[4/4] Building px-plugin and scaffolding smoke project"
PX_PLUGIN_BIN="${PX_PLUGIN_BIN:-$ROOT_DIR/bin/px-plugin}"
if [ ! -x "$PX_PLUGIN_BIN" ]; then
  go build -o "$PX_PLUGIN_BIN" ./tools/cli/cmd/px-plugin
fi
SMOKE_DIR=$(mktemp -d)
if [ "${KEEP_TEMP_DIR:-}" = "1" ]; then
  echo "Keeping smoke temp dir: $SMOKE_DIR"
else
  trap 'rm -rf "$SMOKE_DIR"' EXIT
fi
pushd "$SMOKE_DIR" > /dev/null
"$PX_PLUGIN_BIN" init com.powerx.smoke --force --module github.com/example/smoke > /dev/null
if [ ! -f com.powerx.smoke/plugin.yaml ]; then
  echo "Error: CLI scaffold missing plugin.yaml" >&2
  exit 1
fi
popd > /dev/null

end_ts=$(date +%s)
elapsed=$((end_ts - start_ts))
echo "=== Smoke workflow complete in ${elapsed}s ==="
