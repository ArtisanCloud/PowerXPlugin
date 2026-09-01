#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
IAM_DIRECTORY_CONTRACT_DIR="${ROOT_DIR}/framework/backend/go/iam/contracts"
ADAPTER_DIR="${ROOT_DIR}/skeleton/backend/go-gin/internal/services/iam/adapters"

if [[ ! -d "${IAM_DIRECTORY_CONTRACT_DIR}" || ! -d "${ADAPTER_DIR}" ]]; then
  echo "IAM framework directories not found" >&2
  exit 1
fi

# UUIDs are machine identifiers. They must never be assigned to a field that
# is rendered as a human display name by the framework or adapter layer.
if rg -n --glob '*.go' --glob '!*_test.go' 'DisplayName\s*:\s*[^,}]*MemberUUID|"display_name"\s*:\s*[^,}]*member_uuid' "${IAM_DIRECTORY_CONTRACT_DIR}" "${ADAPTER_DIR}"; then
  echo "IAM directory must not use member_uuid as display_name" >&2
  exit 1
fi

# The IAM framework is a typed boundary; raw dynamic Gateway payloads belong
# outside this package and must not leak back into business callers.
if rg -n --glob '*.go' --glob '!*_test.go' 'map\[string\]any|map\[string\]interface\{\}' "${IAM_DIRECTORY_CONTRACT_DIR}" "${ADAPTER_DIR}"; then
  echo "IAM framework must not expose raw dynamic Gateway payloads" >&2
  exit 1
fi

echo "IAM directory boundary check passed"
