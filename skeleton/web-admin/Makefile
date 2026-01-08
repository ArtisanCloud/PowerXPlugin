# web-admin Makefile：提供前端本地/CI 常用目标

.PHONY: ci-frontend clean install verify-lightningcss lint test build

NPM ?= npm
NPM_CACHE ?= $(abspath ../tmp/npm-cache-web-admin)

export npm_config_cache := $(NPM_CACHE)

clean: ## 清理构建产物与依赖
	rm -rf node_modules .nuxt .output

install: ## 安装依赖（CI 友好）
	$(NPM) ci --include=optional

verify-lightningcss: ## 校验 optional 依赖是否可用（避免某些平台缺失导致构建失败）
	node -e "require('lightningcss'); console.log('lightningcss loaded')"

lint: ## 运行 Lint
	$(NPM) run lint

test: ## 运行测试
	$(NPM) test

build: ## 构建
	$(NPM) run build

ci-frontend: install verify-lightningcss test lint build ## 前端 CI：安装/校验/测试/检查/构建
