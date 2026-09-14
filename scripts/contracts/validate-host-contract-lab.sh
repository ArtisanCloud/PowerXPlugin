#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HOST_LAB="${ROOT_DIR}/skeleton/web-admin/nuxt/app/pages/powerx/host-contract-lab.vue"
HOST_API="${ROOT_DIR}/skeleton/web-admin/nuxt/app/composables/api/useHostContractLab.ts"
ROUTES="${ROOT_DIR}/skeleton/backend/go-gin/internal/transport/http/admin/routes.go"

for file in "$HOST_LAB" "$HOST_API" "$ROUTES"; do
  [[ -f "$file" ]] || { echo "Host Contract status page file not found: $file" >&2; exit 1; }
done

rg -q 'onMounted\(refreshAll\)' "$HOST_LAB" || { echo "Host Contract page must automatically refresh status cards" >&2; exit 1; }
rg -Fq 'probe(item.module)' "$HOST_LAB" || { echo "Host Contract page must invoke module status checks" >&2; exit 1; }
rg -q 'operation: "status"' "$HOST_API" || { echo "Host Contract API must only invoke status" >&2; exit 1; }
rg -q 'adminhostcontract.RegisterRoutes' "$ROUTES" || { echo "Host Contract status route must be registered" >&2; exit 1; }

if rg -n 'UTextarea|manualInput|knowledgeWrite|document\.delete|index\.rebuild|skill\.invoke' "$HOST_LAB"; then
  echo "Host Contract page must not expose manual JSON or write-operation panels" >&2
  exit 1
fi

echo "host contract status page boundary check passed"
