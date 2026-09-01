# Root Makefile: include modular task definitions from make-files/

MAKEFILES_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
SKELETON_DIR := skeleton
FORWARD_OPTIONAL_VARS = $(if $(filter undefined,$(origin VERSION)),,VERSION=$(VERSION)) $(if $(filter undefined,$(origin DIST_DIR)),,DIST_DIR=$(DIST_DIR)) $(if $(filter undefined,$(origin DIST_ROOT)),,DIST_ROOT=$(DIST_ROOT))

.PHONY: help
help: ## Show available make targets
	@echo "PowerXPlugin root shortcuts"
	@echo ""
	@echo "  skeleton-dist      Build skeleton dist package from repo root"
	@echo "  skeleton-install   Install skeleton dist to PowerX host (API_BASE/TOKEN required)"
	@echo "  skeleton-reinstall Disable -> force install -> switch version"
	@echo "  plugin-yaml-check  One-shot plugin.yaml checks (id/capabilities/events)"
	@echo "  plugin-permission-declaration-check  Check PowerX permissions/page/api bindings"
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
        plugin-id-check plugin-yaml-check plugin-permission-declaration-check manifest-align-fix manifest-align-check \
        dist dist-linux package pack package-pxp local-install local-install-run local-install-pxp release package-release \
        build-px-plugin \
        install-px-plugin \
        migrate migrate-cmd seed setup-db reset-db \
        test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check

skeleton-dist: ## Build skeleton dist package from repo root
	@$(MAKE) -C $(SKELETON_DIR) dist BACKEND=$(BACKEND) $(FORWARD_OPTIONAL_VARS)
	@dist_version="$(VERSION)"; \
	if [ -z "$$dist_version" ]; then dist_version="$$(awk -F': *' '/^version:/ {print $$2; exit}' $(SKELETON_DIR)/plugin.yaml)"; fi; \
	dist_src="$(DIST_DIR)"; \
	if [ -z "$$dist_src" ]; then dist_src="dist/$$dist_version"; fi; \
	case "$$dist_src" in /*) abs_src="$$dist_src" ;; *) abs_src="$(SKELETON_DIR)/$$dist_src" ;; esac; \
	alias_dir="$(SKELETON_DIR)/dist/mac"; \
	test -d "$$abs_src" || { echo "❌ dist alias sync failed: source not found: $$abs_src"; exit 1; }; \
	abs_src_real="$$(cd "$$abs_src" && pwd -P)"; \
	mkdir -p "$(SKELETON_DIR)/dist"; \
	abs_alias_real="$$(cd "$(SKELETON_DIR)/dist" && pwd -P)/mac"; \
	if [ "$$abs_src_real" = "$$abs_alias_real" ]; then \
	  echo "==> install upload alias already at: $$alias_dir"; \
	else \
	  rm -rf "$$alias_dir"; \
	  cp -R "$$abs_src" "$$alias_dir"; \
	  echo "==> synced install upload alias: $$alias_dir"; \
	fi

skeleton-install: ## Install skeleton dist to PowerX host (requires API_BASE/TOKEN)
	@$(MAKE) -C $(SKELETON_DIR) local-install BACKEND=$(BACKEND) $(if $(API_BASE),API_BASE=$(API_BASE),) $(if $(TOKEN),TOKEN=$(TOKEN),) $(if $(ENABLE),ENABLE=$(ENABLE),) $(if $(FORCE),FORCE=$(FORCE),)

skeleton-reinstall: ## Disable -> force install -> switch version(enable=true)
	@$(MAKE) -C $(SKELETON_DIR) local-reinstall BACKEND=$(BACKEND) $(if $(API_BASE),API_BASE=$(API_BASE),) $(if $(TOKEN),TOKEN=$(TOKEN),) $(if $(VERSION),VERSION=$(VERSION),)

plugin-id-check: ## Check plugin id naming & legacy prefix residues
	@$(MAKE) -C $(SKELETON_DIR) plugin-id-check BACKEND=$(BACKEND)

plugin-yaml-check: ## One-shot plugin.yaml checks (id/capabilities/events)
	@$(MAKE) -C $(SKELETON_DIR) plugin-yaml-check BACKEND=$(BACKEND)

plugin-permission-declaration-check: ## Check PowerX permissions/page/api bindings
	@$(MAKE) -C $(SKELETON_DIR) plugin-permission-declaration-check BACKEND=$(BACKEND)

manifest-align-fix: ## Auto sync plugin.d catalogs and verify capability mapping
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix

manifest-align-check: ## Strict drift/mapping check for CI gate
	@node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs

test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend \
dist dist-linux package pack package-pxp local-install local-install-run local-install-pxp release package-release \
build-px-plugin \
install-px-plugin \
migrate migrate-cmd seed setup-db reset-db \
test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check:
	@$(MAKE) -C $(SKELETON_DIR) $@ BACKEND=$(BACKEND) $(FORWARD_OPTIONAL_VARS)
	@if [ "$@" = "dist" ]; then \
	  dist_version="$(VERSION)"; \
	  if [ -z "$$dist_version" ]; then dist_version="$$(awk -F': *' '/^version:/ {print $$2; exit}' $(SKELETON_DIR)/plugin.yaml)"; fi; \
	  dist_src="$(DIST_DIR)"; \
	  if [ -z "$$dist_src" ]; then dist_src="dist/$$dist_version"; fi; \
	  case "$$dist_src" in /*) abs_src="$$dist_src" ;; *) abs_src="$(SKELETON_DIR)/$$dist_src" ;; esac; \
	  alias_dir="$(SKELETON_DIR)/dist/mac"; \
	  test -d "$$abs_src" || { echo "❌ dist alias sync failed: source not found: $$abs_src"; exit 1; }; \
	  abs_src_real="$$(cd "$$abs_src" && pwd -P)"; \
	  mkdir -p "$(SKELETON_DIR)/dist"; \
	  abs_alias_real="$$(cd "$(SKELETON_DIR)/dist" && pwd -P)/mac"; \
	  if [ "$$abs_src_real" = "$$abs_alias_real" ]; then \
	    echo "==> install upload alias already at: $$alias_dir"; \
	  else \
	    rm -rf "$$alias_dir"; \
	    cp -R "$$abs_src" "$$alias_dir"; \
	    echo "==> synced install upload alias: $$alias_dir"; \
	  fi; \
	fi
