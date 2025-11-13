.PHONY: validate validate-capabilities

VALIDATE_MANIFEST ?=

validate-capabilities: ## Validate capability contracts (set VALIDATE_MANIFEST=/path/to/plugin.yaml)
	@if [ -n "$(VALIDATE_MANIFEST)" ]; then \
		node scripts/capabilities/validate-capabilities.mjs --manifest "$(VALIDATE_MANIFEST)"; \
	else \
		echo "Skip capability validation (set VALIDATE_MANIFEST=/path/to/plugin.yaml)"; \
	fi

validate: validate-capabilities
	@echo "validate target completed"
