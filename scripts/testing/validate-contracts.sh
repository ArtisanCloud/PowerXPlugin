#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

source "$ROOT_DIR/scripts/testing/go_env.sh"

echo "=== Validate contracts: JSON schema & OpenAPI ==="

# Validate JSON contracts
for file in docs/contracts/manifest.json docs/contracts/rbac.json; do
  if [ ! -f "$file" ]; then
    echo "Error: $file not found" >&2
    exit 1
  fi
  echo "Validating $file via python3 -m json.tool"
  python3 -m json.tool "$file" > /dev/null
done

echo "Checking docs/contracts/openapi.yaml"
if command -v npx >/dev/null 2>&1; then
  npx --yes @apidevtools/swagger-cli@4.0.4 validate docs/contracts/openapi.yaml
else
  echo "Warning: npx not available, skipping OpenAPI validation" >&2
fi

# Optional CLI scaffold validation
TEMP_DIR=$(mktemp -d)
if [ "${KEEP_TEMP_DIR:-}" = "1" ]; then
  echo "Keeping temp directory: $TEMP_DIR"
else
  trap 'rm -rf "$TEMP_DIR"' EXIT
fi

PX_PLUGIN_BIN="${PX_PLUGIN_BIN:-$ROOT_DIR/bin/px-plugin}"
if [ ! -x "$PX_PLUGIN_BIN" ]; then
  echo "Building px-plugin CLI at $PX_PLUGIN_BIN"
  go build -o "$PX_PLUGIN_BIN" ./tools/cli/cmd/px-plugin
fi

pushd "$TEMP_DIR" > /dev/null
"$PX_PLUGIN_BIN" init com.powerx.validation --force --module github.com/example/powerxvalidation > /dev/null
python3 -m json.tool com.powerx.validation/docs/contracts/manifest.json > /dev/null
python3 -m json.tool com.powerx.validation/docs/contracts/rbac.json > /dev/null
popd > /dev/null

echo "Contracts validation complete"
