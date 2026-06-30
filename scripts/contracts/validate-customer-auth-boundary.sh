#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
CUSTOMERFW_DIR="$ROOT_DIR/framework/backend/go/runtime/customerfw"

if [[ ! -d "$CUSTOMERFW_DIR" ]]; then
  echo "customerfw directory not found: $CUSTOMERFW_DIR" >&2
  exit 1
fi

forbidden_domain_terms='CustomerProfile|CustomerTag|CustomerOwner|FollowUp|Timeline|Lead|Opportunity|Sales|Guardian|Player|Learner|Patient|Fan|Entitlement|Benefit|GrowthLevel|GrowthReport'

if rg -n "$forbidden_domain_terms" "$CUSTOMERFW_DIR" -g '*.go' -g '!doc.go' -g '!*_test.go'; then
  echo "customerfw must not define SCRM or industry customer domain models" >&2
  exit 1
fi

if rg -n 'log\.(Info|Warn|Error|Debug)|slog\.|logrus\.' "$CUSTOMERFW_DIR" -g '*.go' -g '!*_test.go' | rg -n 'token|password|secret|refresh_token|access_token'; then
  echo "customerfw must not log raw token/password/secret fields" >&2
  exit 1
fi

echo "customer auth boundary check passed"
