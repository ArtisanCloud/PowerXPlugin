.PHONY: validate validate-capabilities validate-taskbus-contracts

VALIDATE_MANIFEST ?= ./skeleton/plugin.yaml
TASKBUS_CONTRACTS ?= ./specs/008-framework-task-bus/contracts/channel-events.yaml

validate-capabilities: ## Validate capability contracts (set VALIDATE_MANIFEST=/path/to/plugin.yaml)
	@if [ -f "$(VALIDATE_MANIFEST)" ]; then \
		echo "[validate] using manifest $(VALIDATE_MANIFEST)"; \
		node scripts/capabilities/validate-capabilities.mjs --manifest "$(VALIDATE_MANIFEST)"; \
	else \
		echo "Skip capability validation (manifest not found: $(VALIDATE_MANIFEST))"; \
	fi

validate-taskbus-contracts: ## Validate TaskBus event contracts (set TASKBUS_CONTRACTS=/path/to/contracts.yaml)
	@if [ -f "tools/contracts/validate-taskbus-contracts.go" ] && [ -f "$(TASKBUS_CONTRACTS)" ]; then \
		echo "[validate] taskbus contracts: $(TASKBUS_CONTRACTS)"; \
		mkdir -p tmp/go-build; \
		(cd tools/contracts && GOWORK=off GOCACHE="$(abspath ./tmp/go-build)" go run . "$(abspath $(TASKBUS_CONTRACTS))"); \
	else \
		echo "Skip TaskBus contracts validation (missing validator or contracts file)"; \
	fi

validate: validate-capabilities validate-taskbus-contracts
	@echo "validate target completed"
