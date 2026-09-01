# =========================
# build.mk （完整版本）
# 支持：
#  - Host（被 PowerX 反代运行）：POWERX_PROXY=1, baseURL=/_p/<pluginId>/admin/
#  - Standalone（独立部署/本地预览）：POWERX_PROXY=0, baseURL=/
#  - 前后端构建、运行、打包、检查
# =========================

# ===== 基础信息 =====
PLUGIN_ID           ?= com.powerx.plugins.base
PLATFORM            ?= host
TARGET_ARCH         ?= amd64

# ===== 目录结构（可按项目调整）=====
# 后端代码在仓库根；如你的 cmd/plugin 在 repo/cmd/plugin，请保持 BACKEND_DIR = .
BACKEND_DIR         ?= .
BUILD_DIR           ?= $(BACKEND_DIR)/bin
ABS_BACKEND_DIR     := $(abspath $(BACKEND_DIR))
ABS_BUILD_DIR       := $(abspath $(BUILD_DIR))
GO_BUILD_CACHE     ?= $(abspath $(BACKEND_DIR)/.cache/go-build)

FRONTEND_DIR        ?= web-admin
FRONTEND_OUTPUT     ?= $(FRONTEND_DIR)/.output

CLI_ROOT_DIR        ?= ..
CLI_BUILD_CACHE     ?= $(abspath $(CLI_ROOT_DIR)/.cache/go-build)
PX_PLUGIN_CLI_VERSION ?=
GO_BIN_DIR          ?= $(shell sh -c 'if [ -n "$$GOBIN" ]; then printf "%s" "$$GOBIN"; else GOPATH="$$(go env GOPATH 2>/dev/null)"; printf "%s/bin" "$$GOPATH"; fi')

# Dist（install/local 用）
DIST_ROOT           ?= dist
DIST_DIR            ?= $(DIST_ROOT)/$(VERSION)
DIST_BACKEND_BIN    ?= $(DIST_DIR)/bin
DIST_WEBADMIN_DIR   ?= $(DIST_DIR)/web-admin
DIST_WEBADMIN_OUTPUT?= $(DIST_WEBADMIN_DIR)/.output
DIST_VERIFY         ?= 1

# Release（完整发布包）
RELEASE_ROOT        ?= target
RELEASE_DIR         ?= $(RELEASE_ROOT)/$(VERSION)
RELEASE_BACKEND_BIN ?= $(RELEASE_DIR)/bin
RELEASE_WEBADMIN_DIR?= $(RELEASE_DIR)/web-admin
RELEASE_WEBADMIN_OUTPUT ?= $(RELEASE_WEBADMIN_DIR)/.output

# ===== URL / 端口 =====
# Host 构建时写入到前端 baseURL
POWERX_ADMIN_BASE   ?= /_p/$(PLUGIN_ID)/admin/
HOST_PORT           ?= 4100                       # 运行 Host 产物时的本地端口
STANDALONE_PORT     ?= 4200                       # 运行 Standalone 产物时的本地端口
CHECK_PORT          ?= 4999                       # 临时检查端口（不要和上面冲突）

# ===== Go 构建（如不需要可删）=====
.PHONY: build
build: ## 构建后端（本机平台）
	@echo "==> 构建后端二进制（本机平台）..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
	  echo "跳过 Go 构建（python backend）"; \
	else \
	  mkdir -p $(ABS_BUILD_DIR); \
	  mkdir -p $(GO_BUILD_CACHE); \
	  rm -f $(ABS_BUILD_DIR)/plugin $(ABS_BUILD_DIR)/migrate; \
	  GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/plugin ./cmd/plugin; \
	  if [ -d "$(ABS_BACKEND_DIR)/cmd/database" ]; then \
	    echo "   构建 migrate（如存在）..."; \
	    GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/migrate ./cmd/database; \
	  else \
	    echo "   跳过 migrate（未找到 cmd/database）"; \
	  fi; \
	fi

.PHONY: build-px-plugin
build-px-plugin: ## 在仓库根目录构建 bin/px-plugin
	@echo "==> 构建 px-plugin CLI -> $(CLI_ROOT_DIR)/bin/px-plugin"
	@mkdir -p $(CLI_ROOT_DIR)/bin
	@mkdir -p $(CLI_BUILD_CACHE)
	@if [ -n "$(PX_PLUGIN_CLI_VERSION)" ]; then \
	  echo "   使用版本号: $(PX_PLUGIN_CLI_VERSION)"; \
	  GOCACHE=$(CLI_BUILD_CACHE) go build -C $(CLI_ROOT_DIR) -ldflags "-X main.version=$(PX_PLUGIN_CLI_VERSION)" -o ./bin/px-plugin ./tools/cli/cmd/px-plugin; \
	else \
	  GOCACHE=$(CLI_BUILD_CACHE) go build -C $(CLI_ROOT_DIR) -o ./bin/px-plugin ./tools/cli/cmd/px-plugin; \
	fi
	@$(CLI_ROOT_DIR)/bin/px-plugin --version

.PHONY: install-px-plugin
install-px-plugin: build-px-plugin ## 构建并安装到 GOBIN/GOPATH/bin，保持 `px-plugin` 直接可用
	@echo "==> 安装 px-plugin 到 $(GO_BIN_DIR)/px-plugin"
	@mkdir -p $(GO_BIN_DIR)
	@cp $(CLI_ROOT_DIR)/bin/px-plugin $(GO_BIN_DIR)/px-plugin
	@echo "已安装: $(GO_BIN_DIR)/px-plugin"
	@$(GO_BIN_DIR)/px-plugin --version || { \
	  echo "⚠️ 已安装，但无法直接执行 $(GO_BIN_DIR)/px-plugin --version（可能被系统策略拦截或终端缓存影响）"; \
	  echo "   请手动执行: hash -r && px-plugin --version"; \
	}

.PHONY: build-linux
build-linux: ## 构建后端（Linux amd64）
	@echo "==> 构建后端二进制（Linux/$(TARGET_ARCH)）..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
	  echo "跳过 Go 构建（python backend）"; \
	else \
	  mkdir -p $(ABS_BUILD_DIR); \
	  mkdir -p $(GO_BUILD_CACHE); \
	  rm -f $(ABS_BUILD_DIR)/plugin $(ABS_BUILD_DIR)/migrate; \
	  GOOS=linux GOARCH=$(TARGET_ARCH) GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/plugin ./cmd/plugin; \
	  if [ -d "$(ABS_BACKEND_DIR)/cmd/database" ]; then \
	    echo "   构建 migrate（Linux/$(TARGET_ARCH)）..."; \
	    GOOS=linux GOARCH=$(TARGET_ARCH) GOCACHE=$(GO_BUILD_CACHE) go build -C $(ABS_BACKEND_DIR) -o $(ABS_BUILD_DIR)/migrate ./cmd/database; \
	  else \
	    echo "   跳过 migrate（未找到 cmd/database）"; \
	  fi; \
	fi

.PHONY: dist-backend
dist-backend:
	@if [ "$(PLATFORM)" = "linux" ]; then \
	  $(MAKE) build-linux BACKEND=$(BACKEND) BUILD_DIR="$(BUILD_DIR)" TARGET_ARCH="$(TARGET_ARCH)" GO_BUILD_CACHE="$(GO_BUILD_CACHE)"; \
	else \
	  $(MAKE) build BACKEND=$(BACKEND) BUILD_DIR="$(BUILD_DIR)" GO_BUILD_CACHE="$(GO_BUILD_CACHE)"; \
	fi
	@if [ "$(BACKEND)" != "fastapi" ]; then \
	  if [ ! -s "$(BUILD_DIR)/plugin" ]; then \
	    echo "❌ dist-backend 失败：未生成有效的后端二进制 $(BUILD_DIR)/plugin"; \
	    exit 1; \
	  fi; \
	fi

# ===== 前端构建（Host / 被 PowerX 反代）=====
.PHONY: frontend-build
frontend-build: ## 构建 Host 包（POWERX_PROXY=1, baseURL=$(POWERX_ADMIN_BASE)）
	@echo "==> 构建 web-admin（Host 包） POWERX_PROXY=1 baseURL=$(POWERX_ADMIN_BASE)"
	cd $(FRONTEND_DIR) && \
	  POWERX_PROXY=1 \
	  NUXT_PUBLIC_INSIDE_POWERX=1 \
	  POWERX_PLUGIN_ID="$(PLUGIN_ID)" \
	  POWERX_PLUGIN_VERSION="$(VERSION)" \
	  NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" \
	  NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" \
	  NUXT_PUBLIC_API_BASE= \
	  NUXT_PUBLIC_API_PREFIX= \
	  POWERX_ADMIN_BASE="$(POWERX_ADMIN_BASE)" \
	  NODE_ENV=production \
	  npm run build

# ===== 前端构建（Standalone / 独立部署）=====
.PHONY: frontend-build-standalone
frontend-build-standalone: ## 构建 Standalone 包（POWERX_PROXY=0, baseURL=/）
	@echo "==> 构建 web-admin（Standalone 包） POWERX_PROXY=0 baseURL=/"
	cd $(FRONTEND_DIR) && \
	  POWERX_PROXY=0 \
	  NUXT_PUBLIC_INSIDE_POWERX=0 \
	  POWERX_PLUGIN_ID="$(PLUGIN_ID)" \
	  POWERX_PLUGIN_VERSION="$(VERSION)" \
	  NUXT_PUBLIC_POWERX_PLUGIN_ID="$(PLUGIN_ID)" \
	  NUXT_PUBLIC_POWERX_PLUGIN_VERSION="$(VERSION)" \
	  NODE_ENV=production \
	  npm run build

# ===== 运行已编译的前端产物（Host）=====
.PHONY: run-frontend
run-frontend: ## 启动 Host 产物（默认端口 $(HOST_PORT)）
	@if [ ! -f "$(FRONTEND_OUTPUT)/server/index.mjs" ]; then \
	  echo "❌ 未找到 $(FRONTEND_OUTPUT)/server/index.mjs"; \
	  echo "   请先执行: make frontend-build"; \
	  exit 1; \
	fi
	@if [ -z "$(HOST_PORT)" ]; then echo "❌ HOST_PORT 为空"; exit 1; fi
	@echo "==> 启动 Host 产物： http://127.0.0.1:$(HOST_PORT)$(POWERX_ADMIN_BASE)"
	cd $(FRONTEND_OUTPUT) && PORT=$(HOST_PORT) NODE_ENV=production node server/index.mjs

# ===== 运行已编译的前端产物（Standalone）=====
.PHONY: run-frontend-standalone
run-frontend-standalone: ## 启动 Standalone 产物（默认端口 $(STANDALONE_PORT)）
	@if [ ! -f "$(FRONTEND_OUTPUT)/server/index.mjs" ]; then \
	  echo "❌ 未找到 $(FRONTEND_OUTPUT)/server/index.mjs"; \
	  echo "   请先执行: make frontend-build-standalone"; \
	  exit 1; \
	fi
	@if [ -z "$(STANDALONE_PORT)" ]; then echo "❌ STANDALONE_PORT 为空"; exit 1; fi
	@echo "==> 启动 Standalone 产物： http://127.0.0.1:$(STANDALONE_PORT)/"
	cd $(FRONTEND_OUTPUT) && PORT=$(STANDALONE_PORT) NODE_ENV=production node server/index.mjs

# ===== 校验产物中的 baseURL（Host 构建）=====
.PHONY: check-base-host
check-base-host: frontend-build ## 构建后临时起 Nitro，抓首页里的 app.baseURL
	@echo "==> 检查 Host 产物 baseURL..."
	@if [ ! -f "$(FRONTEND_OUTPUT)/server/index.mjs" ]; then \
	  echo "❌ 未找到 $(FRONTEND_OUTPUT)/server/index.mjs"; exit 1; fi
	cd $(FRONTEND_OUTPUT) && \
	  PORT=$(CHECK_PORT) NODE_ENV=production node server/index.mjs & echo $$! > .nuxt_pid; \
	  for i in `seq 1 40`; do \
	    sleep 0.25; curl -fsS "http://127.0.0.1:$(CHECK_PORT)$(POWERX_ADMIN_BASE)" >/dev/null 2>&1 && break; \
	    if [ $$i -eq 40 ]; then echo "Nitro 未就绪"; kill $$(cat .nuxt_pid) 2>/dev/null || true; exit 1; fi; \
	  done; \
	  echo -n "HTML 中的 "; \
	  curl -s "http://127.0.0.1:$(CHECK_PORT)$(POWERX_ADMIN_BASE)" | grep -o 'app:{baseURL:"[^"]*"}' | head -1 || true; \
	  kill $$(cat .nuxt_pid) 2>/dev/null || true; rm -f .nuxt_pid

# ===== 校验产物中的 baseURL（Standalone 构建）=====
.PHONY: check-base-standalone
check-base-standalone: frontend-build-standalone
	@echo "==> 检查 Standalone 产物 baseURL..."
	@if [ ! -f "$(FRONTEND_OUTPUT)/server/index.mjs" ]; then \
	  echo "❌ 未找到 $(FRONTEND_OUTPUT)/server/index.mjs"; exit 1; fi
	cd $(FRONTEND_OUTPUT) && \
	  PORT=$(CHECK_PORT) NODE_ENV=production node server/index.mjs & echo $$! > .nuxt_pid; \
	  for i in `seq 1 40`; do \
	    sleep 0.25; curl -fsS "http://127.0.0.1:$(CHECK_PORT)/" >/dev/null 2>&1 && break; \
	    if [ $$i -eq 40 ]; then echo "Nitro 未就绪"; kill $$(cat .nuxt_pid) 2>/dev/null || true; exit 1; fi; \
	  done; \
	  echo -n "HTML 中的 "; \
	  curl -s "http://127.0.0.1:$(CHECK_PORT)/" | grep -o 'app:{baseURL:"[^"]*"}' | head -1 || true; \
	  kill $$(cat .nuxt_pid) 2>/dev/null || true; rm -f .nuxt_pid

# ===== 生成 dist（目录安装包，给 PowerX 的 install/local 用）=====
.PHONY: dist
dist: plugin-yaml-check plugin-permission-declaration-check dist-backend frontend-build
	@echo "==> 生成 dist 安装包目录：$(DIST_DIR)"
	@rm -rf $(DIST_DIR)
	@mkdir -p $(DIST_BACKEND_BIN) $(DIST_WEBADMIN_OUTPUT)
	@echo "写入插件清单 -> $(DIST_DIR)/plugin.yaml (version=$(VERSION))"
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(DIST_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then \
	  mkdir -p $(DIST_DIR)/plugin.d; \
	  cp -R plugin.d/. $(DIST_DIR)/plugin.d/; \
	fi
	@if [ -f "config/event_fabric.yaml" ]; then \
	  mkdir -p $(DIST_DIR)/config; \
	  cp config/event_fabric.yaml $(DIST_DIR)/config/event_fabric.yaml; \
	fi
	@if [ -d "skills" ]; then \
	  mkdir -p $(DIST_DIR)/skills; \
	  cp -R skills/. $(DIST_DIR)/skills/; \
	fi
	@cp $(BUILD_DIR)/plugin $(DIST_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(DIST_BACKEND_BIN)/; fi
	@if [ -d "$(FRONTEND_OUTPUT)" ] && [ -n "$$(ls -A $(FRONTEND_OUTPUT) 2>/dev/null)" ]; then \
	  echo "复制前端构建产物 -> $(DIST_WEBADMIN_OUTPUT)"; \
	  cp -R $(FRONTEND_OUTPUT)/. $(DIST_WEBADMIN_OUTPUT)/; \
	else \
	  echo "⚠️  未找到前端构建产物：$(FRONTEND_OUTPUT)"; \
	fi
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then \
	  mkdir -p $(DIST_WEBADMIN_DIR)/i18n; \
	  cp -R $(FRONTEND_DIR)/i18n/. $(DIST_WEBADMIN_DIR)/i18n/; \
	fi
	@if [ -f README.md ]; then cp README.md $(DIST_DIR)/; fi
	@if [ "$(DIST_VERIFY)" = "1" ]; then \
	  echo "==> dist 验证（内建）"; \
	  rg -q 'POWERX_BIND_ADDR:[[:space:]]*":__POWERX_DYNAMIC_PORT__"' "$(DIST_DIR)/plugin.yaml" || { echo "❌ dist 验证失败：plugin.yaml 必须使用 POWERX_BIND_ADDR 动态端口占位符"; exit 1; }; \
	  ! rg -q 'port:[[:space:]]*(8078|8086)[[:space:]]*$$' "$(DIST_DIR)/plugin.yaml" || { echo "❌ dist 验证失败：plugin.yaml 不应固化旧 backend port"; exit 1; }; \
	  echo "manifest runtime port check: ok"; \
	  if [ -d "skills" ]; then \
	    find "$(DIST_DIR)/skills" -mindepth 2 -maxdepth 2 -name SKILL.md -print -quit | rg -q . || { echo "❌ dist 验证失败：skills 目录没有标准 SKILL.md 包"; exit 1; }; \
	    echo "skills package check: ok"; \
	  fi; \
	  if [ "$(BACKEND)" != "fastapi" ]; then \
	    test -s "$(DIST_BACKEND_BIN)/plugin" || { echo "❌ dist 验证失败：缺少后端二进制 $(DIST_BACKEND_BIN)/plugin"; exit 1; }; \
	    strings "$(DIST_BACKEND_BIN)/plugin" | rg -q "/ws-bus/test-flow" || { echo "❌ dist 验证失败：binary 缺少 /ws-bus/test-flow"; exit 1; }; \
	    strings "$(DIST_BACKEND_BIN)/plugin" | rg -q "/admin/runtime/ws-bus/grant|/ws-bus/grant" || { echo "❌ dist 验证失败：binary 缺少 ws-bus grant 路由片段"; exit 1; }; \
	    strings "$(DIST_BACKEND_BIN)/plugin" | rg -q "/admin/runtime/ws-bus/publish|/ws-bus/publish" || { echo "❌ dist 验证失败：binary 缺少 ws-bus publish 路由片段"; exit 1; }; \
	    echo "binary check: ok"; \
	  fi; \
	  if [ -f "$(DIST_DIR)/plugin.d/rbac.yaml" ]; then \
	    rg -q "/admin/runtime/ws-bus/test-flow" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 test-flow"; exit 1; }; \
	    rg -q "/admin/runtime/ws-bus/grant" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 grant"; exit 1; }; \
	    rg -q "/admin/runtime/ws-bus/publish" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 publish"; exit 1; }; \
	    rg -q "/admin/runtime/event-fabric/topics" "$(DIST_DIR)/plugin.d/rbac.yaml" || { echo "❌ dist 验证失败：rbac 缺少 event-fabric/topics"; exit 1; }; \
	    echo "rbac check: ok"; \
	  else \
	    echo "❌ dist 验证失败：缺少 $(DIST_DIR)/plugin.d/rbac.yaml"; exit 1; \
	  fi; \
	fi

.PHONY: dist-linux
dist-linux: ## 兼容别名：等价于 `make dist PLATFORM=linux`
	@$(MAKE) dist PLATFORM=linux DIST_DIR="$(DIST_DIR)" BACKEND="$(BACKEND)" BUILD_DIR="$(BUILD_DIR)" TARGET_ARCH="$(TARGET_ARCH)"

# ===== 生成 release（完整发布包）=====
.PHONY: release
release: build frontend-build
	@echo "==> 生成 release 发布目录：$(RELEASE_DIR)"
	@rm -rf $(RELEASE_DIR)
	@mkdir -p $(RELEASE_BACKEND_BIN) $(RELEASE_WEBADMIN_OUTPUT)
	@echo "写入插件清单 -> $(RELEASE_DIR)/plugin.yaml (version=$(VERSION))"
	@awk -v ver="$(VERSION)" 'BEGIN{patched=0} /^[[:space:]]*version:[[:space:]]*/ && !patched {print "version: " ver; patched=1; next} {print} END{if(!patched) print "version: " ver}' plugin.yaml > $(RELEASE_DIR)/plugin.yaml
	@if [ -d "plugin.d" ]; then \
	  mkdir -p $(RELEASE_DIR)/plugin.d; \
	  cp -R plugin.d/. $(RELEASE_DIR)/plugin.d/; \
	fi
	@if [ -f "config/event_fabric.yaml" ]; then \
	  mkdir -p $(RELEASE_DIR)/config; \
	  cp config/event_fabric.yaml $(RELEASE_DIR)/config/event_fabric.yaml; \
	fi
	@if [ -d "skills" ]; then \
	  mkdir -p $(RELEASE_DIR)/skills; \
	  cp -R skills/. $(RELEASE_DIR)/skills/; \
	fi
	@cp $(BUILD_DIR)/plugin $(RELEASE_BACKEND_BIN)/
	@if [ -f "$(BUILD_DIR)/migrate" ]; then cp $(BUILD_DIR)/migrate $(RELEASE_BACKEND_BIN)/; fi
	@cp -R $(FRONTEND_OUTPUT)/. $(RELEASE_WEBADMIN_OUTPUT)/
	@if [ -d "$(FRONTEND_DIR)/i18n" ]; then \
	  mkdir -p $(RELEASE_WEBADMIN_DIR)/i18n; \
	  cp -R $(FRONTEND_DIR)/i18n/. $(RELEASE_WEBADMIN_DIR)/i18n/; \
	fi
	@if [ -f README.md ]; then cp README.md $(RELEASE_DIR)/; fi
	@rg -q 'POWERX_BIND_ADDR:[[:space:]]*":__POWERX_DYNAMIC_PORT__"' "$(RELEASE_DIR)/plugin.yaml" || { echo "❌ release 验证失败：plugin.yaml 必须使用 POWERX_BIND_ADDR 动态端口占位符"; exit 1; }
	@! rg -q 'port:[[:space:]]*(8078|8086)[[:space:]]*$$' "$(RELEASE_DIR)/plugin.yaml" || { echo "❌ release 验证失败：plugin.yaml 不应固化旧 backend port"; exit 1; }
	@echo "release manifest runtime port check: ok"

# ===== 打包 zip =====
.PHONY: package
package: dist
	@echo "==> 打包 dist 为 zip（install/local 用）..."
	@rm -f $(PLUGIN_ID)-$(VERSION).zip
	@cd $(DIST_ROOT) && zip -r ../$(PLUGIN_ID)-$(VERSION).zip $(VERSION)
	@echo "✅ 输出：$(PLUGIN_ID)-$(VERSION).zip"

.PHONY: package-release
package-release: release
	@echo "==> 打包 release 为 zip（发布包）..."
	@rm -f $(PLUGIN_ID)-$(VERSION)-release.zip
	@cd $(RELEASE_ROOT) && zip -r ../$(PLUGIN_ID)-$(VERSION)-release.zip $(VERSION)
	@echo "✅ 输出：$(PLUGIN_ID)-$(VERSION)-release.zip"

# ===== 清理 =====
.PHONY: clean
clean:
	@echo "==> 清理 build 产物..."
	@rm -rf $(BUILD_DIR)

.PHONY: dist-clean
dist-clean:
	@echo "==> 清理 dist..."
	@rm -rf $(DIST_ROOT)

.PHONY: release-clean
release-clean:
	@echo "==> 清理 target..."
	@rm -rf $(RELEASE_ROOT)
