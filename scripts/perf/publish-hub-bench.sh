#!/usr/bin/env bash
set -euo pipefail

ROOT=$(git rev-parse --show-toplevel)
CLI=$ROOT/tools/cli

if ! command -v ts-node >/dev/null; then
  echo "ts-node is required (npm install ts-node)" >&2
fi

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "[bench] measuring px-plugin publish pipeline"
START=$(date +%s%3N)
node --loader ts-node/esm $CLI/src/commands/publish.ts || true
END=$(date +%s%3N)
echo "publish duration: $((END-START)) ms"

echo "[bench] measuring offline packager"
node --loader ts-node/esm $CLI/src/commands/dist.ts || true

echo "[bench] reminder: run Playwright to validate Admin install flow"
echo "npx playwright test tests/admin/install_flow.spec.ts"
