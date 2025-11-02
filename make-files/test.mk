# test.mk - Testing workflows

SMOKE_TIMEOUT ?= 300
REGRESSION_TIMEOUT ?= 3600

.PHONY: test-smoke test-regression

## Testing ------------------------------------------------------------------

test-smoke: ## Run smoke checks (Go/unit/contract/CLI) with timeout
	@echo "=== Smoke Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(SMOKE_TIMEOUT) ./scripts/testing/smoke.sh
	@echo "=== Smoke Tests Finished ==="

test-regression: ## Run full regression suite (smoke + Go + frontend + Playwright)
	@echo "=== Regression Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(REGRESSION_TIMEOUT) ./scripts/testing/regression.sh
	@echo "=== Regression Tests Finished ==="

