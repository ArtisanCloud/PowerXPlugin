# test.mk - Testing workflows

SMOKE_TIMEOUT ?= 300
REGRESSION_TIMEOUT ?= 3600

.PHONY: test-smoke test-regression ci-all ci-backend ci-frontend

## Testing ------------------------------------------------------------------

test-smoke: ## Run smoke checks (Go/unit/contract/CLI) with timeout
	@echo "=== Smoke Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(SMOKE_TIMEOUT) ./scripts/testing/smoke.sh
	@echo "=== Smoke Tests Finished ==="

test-regression: ## Run full regression suite (smoke + Go + frontend + Playwright)
	@echo "=== Regression Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(REGRESSION_TIMEOUT) ./scripts/testing/regression.sh
	@echo "=== Regression Tests Finished ==="

## CI Simulation -------------------------------------------------------------

ci-all: ## Run full GitHub CI workflow locally via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml

ci-backend: ## Run only backend job from CI workflow via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml -j backend

ci-frontend: ## Run only frontend job from CI workflow via act
	@if ! command -v act >/dev/null 2>&1; then \
	  echo "Error: act is not installed. See https://github.com/nektos/act" >&2; \
	  exit 1; \
	fi
	act -W .github/workflows/ci.yml -j frontend
