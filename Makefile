SMOKE_TIMEOUT ?= 300
REGRESSION_TIMEOUT ?= 3600

.PHONY: test-smoke test-regression

## Smoke workflow: wrapper around scripts/testing/smoke.sh with timeout/diagnostics
test-smoke:
	@echo "=== Smoke Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(SMOKE_TIMEOUT) ./scripts/testing/smoke.sh
	@echo "=== Smoke Tests Finished ==="

## Regression workflow wrapper (runs smoke + Playwright)
test-regression:
	@echo "=== Regression Tests Start ==="
	@python3 scripts/testing/run_with_timeout.py --timeout $(REGRESSION_TIMEOUT) ./scripts/testing/regression.sh
	@echo "=== Regression Tests Finished ==="
