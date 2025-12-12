.PHONY: capabilities-lint
capabilities-lint: ## Run capability manifest/schema validation (scripts/capabilities)
	npm --prefix scripts/capabilities run lint

.PHONY: capabilities-export
capabilities-export: ## Generate capability protocol assets (placeholder)
	npm --prefix scripts/capabilities run export
