# Root Makefile: include modular task definitions from make-files/

MAKEFILES_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
SKELETON_DIR := skeleton

.PHONY: help
help: ## Show available make targets
	@$(MAKE) -C $(SKELETON_DIR) help

# Proxy common targets to skeleton/Makefile to unify entrypoints.
.PHONY: test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend \
        build-px-plugin \
        install-px-plugin \
        migrate migrate-cmd seed setup-db reset-db \
        test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check
test test-smoke test-regression test-cli-devwatch ci-all ci-backend ci-frontend \
build-px-plugin \
install-px-plugin \
migrate migrate-cmd seed setup-db reset-db \
test-admin lint-admin build-admin test-coverage lint fmt mod-tidy test-all integration-smoke check:
	@$(MAKE) -C $(SKELETON_DIR) $@ BACKEND=$(BACKEND)
