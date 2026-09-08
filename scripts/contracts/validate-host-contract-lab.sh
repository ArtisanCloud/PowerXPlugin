#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
HANDLER="${ROOT_DIR}/skeleton/backend/go-gin/internal/transport/http/admin/host_contract/handler.go"
CAPABILITY_LAB="${ROOT_DIR}/skeleton/web-admin/nuxt/app/pages/powerx/capability-lab.vue"
HOST_LAB="${ROOT_DIR}/skeleton/web-admin/nuxt/app/pages/powerx/host-contract-lab.vue"
ZH_LOCALE="${ROOT_DIR}/skeleton/web-admin/nuxt/i18n/locales/zh.json"
EN_LOCALE="${ROOT_DIR}/skeleton/web-admin/nuxt/i18n/locales/en.json"

for file in "$HANDLER" "$CAPABILITY_LAB" "$HOST_LAB" "$ZH_LOCALE" "$EN_LOCALE"; do
  [[ -f "$file" ]] || { echo "Host Contract Lab file not found: $file" >&2; exit 1; }
done

# The formal handler must use typed Framework clients, never invoke Core URLs
# or retain the legacy Knowledge QA bridge as a substitute.
if rg -n '/api/v1/|http://|https://' "$HANDLER"; then
  echo "Host Contract handler must not construct a Core HTTP URL" >&2
  exit 1
fi
if rg -n 'KnowledgeQABridge|powerxknowledge' "$HANDLER"; then
  echo "Host Contract handler must not use the Knowledge QA bridge" >&2
  exit 1
fi
rg -q 'deps\.KnowledgeDirectory' "$HANDLER" || { echo "Host Contract handler must use KnowledgeDirectory" >&2; exit 1; }

# Capability Lab remains generic and only links to the typed lab for supported
# module identifiers. It must not recreate a direct Core transport.
rg -q "'/powerx/host-contract-lab'" "$CAPABILITY_LAB" || { echo "Capability Lab must link to Host Contract Lab" >&2; exit 1; }
rg -q 'hostContractModule' "$CAPABILITY_LAB" || { echo "Capability Lab must map capabilities to Host Contract modules" >&2; exit 1; }
if rg -n '/api/v1/tenant/' "$CAPABILITY_LAB"; then
  echo "Capability Lab must not construct tenant Core URLs" >&2
  exit 1
fi
rg -q 'requestedModule' "$HOST_LAB" || { echo "Host Contract Lab must honor module links" >&2; exit 1; }

node -e '
const fs = require("fs");
for (const path of process.argv.slice(1)) {
  const doc = JSON.parse(fs.readFileSync(path, "utf8"));
  if (!doc.capabilityLab?.openHostContract) throw new Error(`${path}: missing capabilityLab.openHostContract`);
  if (!doc.navigation?.knowledgeQALab) throw new Error(`${path}: missing navigation.knowledgeQALab`);
}
' "$ZH_LOCALE" "$EN_LOCALE"

echo "host contract lab boundary check passed"
