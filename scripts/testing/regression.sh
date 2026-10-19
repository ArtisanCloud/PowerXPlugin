#!/usr/bin/env bash
set -euo pipefail
REGRESSION_REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$REGRESSION_REPO_ROOT"
mkdir -p artifacts/regression
(cd framework/backend/go && GOWORK=off go test -race ./runtime/regression/...)
(cd skeleton/backend/go-gin && REGRESSION_EVIDENCE_DIR="$REGRESSION_REPO_ROOT/artifacts/regression" go test ./internal/config ./cmd/database/migrate ./internal/services/admin/regression/... ./internal/transport/http/admin/regression ./tests/integration/regression -run Regression\|TemplateRepeated\|MetadataScenario -count=1)
(cd tools/cli && go test ./internal/regression ./cmd -run Regression\|Client -count=1)
(cd tools/cli && REGRESSION_GENERATED_TEST=1 go test ./internal/templates -run RegressionGenerated -count=1 -timeout=4m)
node --experimental-strip-types --test framework/frontend/nuxt/framework-client/__tests__/regression.test.ts
python3 -m unittest discover -s scripts/testing -p 'regression_live_check_test.py'
(cd skeleton/web-admin/nuxt && node node_modules/vitest/dist/cli.js run --dir tests/unit)
npm run sync:templates -- --check
if [[ "${REGRESSION_POSTGRES:-0}" == "1" ]]; then python3 scripts/testing/regression-postgres.py; fi
if [[ "${REGRESSION_BROWSER:-0}" == "1" ]]; then
  (cd skeleton/web-admin/nuxt && node node_modules/nuxi/bin/nuxi.mjs build)
  python3 scripts/testing/regression-browser.py
fi
if [[ "${REGRESSION_GENERATED_BROWSER:-0}" == "1" ]]; then
  python3 scripts/testing/regression-generated.py
fi
