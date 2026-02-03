# Contract: Testing Workflow Interfaces

This document standardises the entry points and expected behaviours for the smoke and regression automation introduced by the PowerXPlugin testing strategy.

## Smoke Workflow (`scripts/testing/smoke.sh` / `make test-smoke`)
- **Inputs**: None (runs from repository root).
- **Prerequisites**:
  - Go 1.24+ installed and available in `PATH`.
  - Node.js 18+ / npm 9+ installed.
  - `PX_PLUGIN_BIN` optional override; otherwise script builds `bin/px-plugin`.
- **Steps (must execute sequentially)**:
  1. `go test ./framework/backend/go/bootstrap/...` with `-coverprofile=tmp/coverage.out`.
  2. `go test ./skeleton/backend/go-gin/internal/routes/...`.
  3. `python3 -m json.tool docs/contracts/manifest.json` and `docs/contracts/rbac.json`.
  4. Build CLI `go build -o bin/px-plugin ./tools/cli/cmd/px-plugin`.
  5. Scaffold temp project and verify `plugin.yaml` presence.
- **Outputs**:
  - `tmp/coverage.out`, `tmp/coverage.html` (generated via `go tool cover`).
  - CLI scaffold temp directory under `/tmp/powerx-smoke-*` (auto-cleaned).
- **Exit Codes**:
  - `0` on success.
  - Non-zero with descriptive stderr if any step fails.

## Regression Workflow (`scripts/testing/regression.sh` / `make test-regression`)
- **Inputs**: Optional environment variables:
  - `PLAYWRIGHT_BASE_URL` (defaults to `http://localhost:3000`).
  - `KEEP_TEMP_DIR=1` to retain generated CLI project for inspection.
- **Prerequisites**: All smoke workflow prerequisites plus Playwright browsers installed (`npx playwright install`).
- **Steps** (in addition to smoke tasks):
  1. `go test ./framework/... ./skeleton/backend/go-gin/... -coverprofile=tmp/coverage.out`.
  2. `npx playwright test` from `skeleton/web-admin/nuxt` (with service wait loop).
  3. Archive Playwright report to `tmp/playwright-report/` if `npx` emits HTML report.
- **Outputs**:
  - Updated `tmp/coverage.out` and `tmp/coverage.html`.
  - Playwright artifacts under `skeleton/web-admin/nuxt/test-results/` and optionally `tmp/playwright-report/`.
- **Exit Codes**:
  - `0` only if all layers pass.
  - Non-zero if any layer fails; script should print failing command.

## Validation Script (`scripts/testing/validate-contracts.sh`)
- **Purpose**: reusable contract check invoked by both workflows and pre-commit hooks.
- **Behaviour**:
  - Validates JSON files via `python3 -m json.tool`.
  - Runs `npx --yes @apidevtools/swagger-cli@4.0.4 validate docs/contracts/openapi.yaml` when `npx` present; otherwise warns.
  - Optionally scaffolds temp project to ensure generated contracts match.
- **Outputs**: Diagnostic logs on stdout/stderr; temporary project cleaned unless `KEEP_TEMP_DIR` set.

All scripts MUST be idempotent and safe for repeated executions. Any new workflow must extend this contract or define a new document under `specs/002-testing-strategy/contracts/`.
