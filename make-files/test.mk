# test.mk - Testing workflows

SMOKE_TIMEOUT ?= 300
REGRESSION_TIMEOUT ?= 3600
BACKEND ?= gin

SMOKE_SCRIPT := ./scripts/testing/smoke.sh
REGRESSION_SCRIPT := ./scripts/testing/regression.sh

ifneq (,$(filter $(BACKEND),fastapi python))
SMOKE_SCRIPT := ./scripts/testing/smoke-python.sh
REGRESSION_SCRIPT := ./scripts/testing/regression-python.sh
endif

.PHONY: test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend

## Testing ------------------------------------------------------------------

test: test-smoke ## Run default smoke test suite

test-smoke: ## Run smoke checks (Go/unit/contract/CLI) with timeout
	@echo "=== Smoke Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(SMOKE_TIMEOUT) $(SMOKE_SCRIPT)
	@echo "=== Smoke Tests Finished ==="

test-regression: ## Run full regression suite (smoke + Go + frontend + Playwright)
	@echo "=== Regression Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(REGRESSION_TIMEOUT) $(REGRESSION_SCRIPT)
	@echo "=== Regression Tests Finished ==="

test-cli-devwatch: ## Run CLI dev watch stack go tests (devwatch/devapi/watch)
	@echo "=== CLI Dev Watch Tests Start ==="
	@cd tools/cli && GOWORK=auto go test ./internal/devwatch ./internal/devapi ./internal/watch
	@echo "=== CLI Dev Watch Tests Finished ==="

## CI Simulation -------------------------------------------------------------

ci-all: ## Run full GitHub CI workflow locally via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml $(ACT_OPTS)

ci-backend: ## Run only backend job from CI workflow via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml -j backend $(ACT_OPTS)

ci-frontend: ## Run only frontend job from CI workflow via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml -j frontend $(ACT_OPTS)
