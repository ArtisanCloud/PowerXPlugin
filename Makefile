# Root Makefile: include modular task definitions from make-files/

MAKEFILES_DIR := $(dir $(lastword $(MAKEFILE_LIST)))

include make-files/test.mk
include make-files/validate.mk
include make-files/capabilities.mk

.PHONY: help
help: ## Show available make targets
	@echo "Available targets:"
	@grep -hE '^[[:alnum:]_.-]+:.*##' make-files/*.mk | awk 'BEGIN {FS = ":.*##"}; {printf "  %-24s %s\n", $$1, $$2}'
