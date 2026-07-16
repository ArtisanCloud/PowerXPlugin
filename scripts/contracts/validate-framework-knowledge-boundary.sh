#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PKG_DIR="${ROOT_DIR}/framework/backend/go/runtime/knowledge"

if [[ ! -d "${PKG_DIR}" ]]; then
  echo "framework knowledge package not found: ${PKG_DIR}" >&2
  exit 1
fi

if rg -n --glob '!**/*_test.go' 'course|patient|order|membership benefit|training plan|support ticket|customer profile' "${PKG_DIR}"; then
  echo "framework knowledge package must not define industry-specific business models" >&2
  exit 1
fi

echo "framework knowledge boundary OK"
