#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")"/../.. && pwd)"
cd "$ROOT_DIR"

echo "=== FastAPI Regression Tests Start ==="
./scripts/testing/smoke-python.sh
# Placeholder for future FastAPI regression tests
echo "=== FastAPI Regression Tests Finished ==="
