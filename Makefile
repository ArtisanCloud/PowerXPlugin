# Root Makefile: include modular task definitions from make-files/

MAKEFILES_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
SKELETON_DIR := skeleton

.PHONY: help
help: ## Show available make targets
	@echo "PowerXPlugin root shortcuts"
	@echo ""
	@echo "  skeleton-dist      Build skeleton dist package from repo root"
	@echo "  skeleton-install   Install skeleton dist to PowerX host (API_BASE/TOKEN required)"
	@echo "  skeleton-reinstall Disable -> force install -> switch version"
	@echo "  plugin-yaml-check  One-shot plugin.yaml checks (id/capabilities/events)"
	@echo "  manifest-align-fix Auto sync plugin.d catalogs from capabilities and verify mapping"
	@echo "  manifest-align-check Strict check catalog drift/mapping for CI gate"
	@echo "  dist               Alias of 'make -C skeleton dist'"
	@echo ""
	@echo "Delegated skeleton targets:"
	@$(MAKE) -C $(SKELETON_DIR) help

.PHONY: ci-agent-assets
ci-agent-assets: ## Check .codex/.specify path normalization
	@npm run sync:templates -- --check
	@npm run check:agent-paths

# Proxy common targets to skeleton/Makefile to unify entrypoints.
.PHONY: test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend \
        skeleton-dist skeleton-install skeleton-reinstall \
        plugin-id-check plugin-yaml-check manifest-align-fix manifest-align-check \
        dist package pack package-pxp local-install local-install-run local-install-pxp release package-release \
        build-px-plugin \
        install-px-plugin \
        migrate migrate-cmd seed setup-db reset-db \
        test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check

skeleton-dist: ## Build skeleton dist package from repo root
	@$(MAKE) -C $(SKELETON_DIR) dist BACKEND=$(BACKEND)

skeleton-install: ## Install skeleton dist to PowerX host (requires API_BASE/TOKEN)
	@$(MAKE) -C $(SKELETON_DIR) local-install BACKEND=$(BACKEND) $(if $(API_BASE),API_BASE=$(API_BASE),) $(if $(TOKEN),TOKEN=$(TOKEN),) $(if $(ENABLE),ENABLE=$(ENABLE),) $(if $(FORCE),FORCE=$(FORCE),)

skeleton-reinstall: ## Disable -> force install -> switch version(enable=true)
	@$(MAKE) -C $(SKELETON_DIR) local-reinstall BACKEND=$(BACKEND) $(if $(API_BASE),API_BASE=$(API_BASE),) $(if $(TOKEN),TOKEN=$(TOKEN),) $(if $(VERSION),VERSION=$(VERSION),)

plugin-id-check: ## Check plugin id naming & legacy prefix residues
	@$(MAKE) -C $(SKELETON_DIR) plugin-id-check BACKEND=$(BACKEND)

plugin-yaml-check: ## One-shot plugin.yaml checks (id/capabilities/events)
	@$(MAKE) -C $(SKELETON_DIR) plugin-yaml-check BACKEND=$(BACKEND)

manifest-align-fix: ## Auto sync plugin.d catalogs and verify capability mapping
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix

manifest-align-check: ## Strict drift/mapping check for CI gate
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs

test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend \
dist package pack package-pxp local-install local-install-run local-install-pxp release package-release \
build-px-plugin \
install-px-plugin \
migrate migrate-cmd seed setup-db reset-db \
test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check:
	@$(MAKE) -C $(SKELETON_DIR) $@ BACKEND=$(BACKEND)
