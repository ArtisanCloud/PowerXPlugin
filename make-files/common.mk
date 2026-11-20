# common.mk 用于定义 Makefile 公共变量与帮助信息

# 基础变量（自动从 manifest / go.mod 推导）
PLUGIN_MANIFEST ?= plugin.yaml
DEFAULT_VERSION := 0.4.0
_MANIFEST_VERSION := $(strip $(shell awk -F': *' '/^version:/ {print $$2; exit}' $(PLUGIN_MANIFEST) 2>/dev/null))
ifneq ($(strip $(DIST_VERSION)),)
VERSION ?= $(DIST_VERSION)
else ifeq ($(_MANIFEST_VERSION),)
VERSION ?= $(DEFAULT_VERSION)
else
VERSION ?= $(_MANIFEST_VERSION)
endif

_MANIFEST_ID := $(strip $(shell awk -F': *' '/^id:/ {print $$2; exit}' $(PLUGIN_MANIFEST) 2>/dev/null))
ifeq ($(_MANIFEST_ID),)
PLUGIN_ID ?= com.powerx.plugins.sample
else
PLUGIN_ID ?= $(_MANIFEST_ID)
endif

PLUGIN_SLUG := $(shell echo $(PLUGIN_ID) | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/-/g')
APP_NAME ?= $(PLUGIN_SLUG)

BACKEND ?= gin
ifeq ($(BACKEND),fastapi)
BACKEND_DIR := backend/python-fastapi
BACKEND_KIND := python
else
ifneq ($(wildcard backend/go.mod),)
BACKEND_DIR := backend
else
BACKEND_DIR := backend/go-gin
endif
BACKEND_KIND := gin
endif

ifeq ($(BACKEND_KIND),gin)
BACKEND_MODULE_FILE := $(BACKEND_DIR)/go.mod
_GO_MODULE := $(strip $(shell awk '/^module[[:space:]]+/ {print $$2; exit}' $(BACKEND_MODULE_FILE) 2>/dev/null))
ifeq ($(_GO_MODULE),)
GO_MODULE ?= github.com/ArtisanCloud/PowerXPlugin/skeleton/backend
else
GO_MODULE ?= $(_GO_MODULE)
endif
else
GO_MODULE ?=
endif

BUILD_DIR := $(BACKEND_DIR)/bin
ifeq ($(BACKEND_KIND),python)
MAIN_FILE := $(BACKEND_DIR)/app/main.py
else
MAIN_FILE := $(BACKEND_DIR)/cmd/plugin/main.go
endif

DOCKER_IMAGE := $(APP_NAME):$(VERSION)
DOCKER_REGISTRY ?=

DIST_ROOT := dist
DIST_DIR := $(DIST_ROOT)/$(VERSION)
DIST_BACKEND_BIN := $(DIST_DIR)/backend/bin
DIST_WEBADMIN_DIR := $(DIST_DIR)/web-admin
DIST_WEBADMIN_OUTPUT := $(DIST_WEBADMIN_DIR)/.output

ifneq ($(wildcard web-admin/nuxt/package.json),)
FRONTEND_DIR := web-admin/nuxt
else
FRONTEND_DIR := web-admin
endif
FRONTEND_OUTPUT := $(FRONTEND_DIR)/.output
FRONTEND_BUILD_CMD ?= npm --prefix $(FRONTEND_DIR) run build

_RAW_SCHEMA := $(shell echo $(PLUGIN_ID) | tr '[:upper:]' '[:lower:]' | sed 's/[^a-z0-9]/_/g')
POWERX_DB_SCHEMA ?= px_$(_RAW_SCHEMA)

RELEASE_ROOT := target
RELEASE_DIR := $(RELEASE_ROOT)/$(VERSION)
RELEASE_BACKEND_BIN := $(RELEASE_DIR)/backend/bin
RELEASE_WEBADMIN_DIR := $(RELEASE_DIR)/web-admin
RELEASE_WEBADMIN_OUTPUT := $(RELEASE_WEBADMIN_DIR)/.output

.DEFAULT_GOAL := help

.PHONY: help
help: ## 显示可用命令列表
	@echo "$(APP_NAME) Makefile"
	@echo ""
	@echo "可用的命令:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		sed 's|^[^:]*:||' | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf " %-18s %s\n", $$1, $$2}'
