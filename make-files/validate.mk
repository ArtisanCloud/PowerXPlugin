.PHONY: validate validate-capabilities

VALIDATE_MANIFEST ?= ./skeleton/plugin.yaml

validate-capabilities: ## Validate capability contracts (set VALIDATE_MANIFEST=/path/to/plugin.yaml)
	@if [ -f "$(VALIDATE_MANIFEST)" ]; then \
		echo "[validate] using manifest $(VALIDATE_MANIFEST)"; \
		node scripts/capabilities/validate-capabilities.mjs --manifest "$(VALIDATE_MANIFEST)"; \
	else \
		echo "Skip capability validation (manifest not found: $(VALIDATE_MANIFEST))"; \
	fi

validate: validate-capabilities
	@echo "validate target completed"
