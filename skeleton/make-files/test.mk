# test.mk 汇总测试与代码质量相关目标

.PHONY: test
test: ## 执行 Go 单元测试
	@echo "运行测试..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "运行 Python 测试..."; \
		if [ -d "$(BACKEND_DIR)/tests" ]; then \
			cd $(BACKEND_DIR) && python -m pytest; \
		else \
			echo "未找到 tests 目录，跳过 Python 测试"; \
		fi; \
	else \
		cd $(BACKEND_DIR) && go test ./...; \
	fi

.PHONY: test-admin
test-admin: ## 运行 web-admin 测试
	@echo "运行 Web Admin 测试..."
	cd $(FRONTEND_DIR) && npm run test

.PHONY: lint-admin
lint-admin: ## 运行 web-admin Lint
	@echo "运行 Web Admin Lint..."
	cd $(FRONTEND_DIR) && npm run lint -- --max-warnings=0

.PHONY: build-admin
build-admin: ## 构建 web-admin 产物
	@echo "构建 Web Admin..."
	cd $(FRONTEND_DIR) && npm run build

.PHONY: test-coverage
test-coverage: ## 执行测试并生成覆盖率报告
	@echo "运行测试并生成覆盖率报告..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "Python 后端暂不生成覆盖率报告"; \
	else \
		cd $(BACKEND_DIR) && go test -coverprofile=coverage.out ./...; \
		cd $(BACKEND_DIR) && go tool cover -html=coverage.out -o coverage.html; \
	fi

.PHONY: lint
lint: ## 运行 golangci-lint
	@echo "运行代码检查..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "Python 后端暂不执行 golangci-lint"; \
	else \
		cd $(BACKEND_DIR) && golangci-lint run; \
	fi

.PHONY: fmt
fmt: ## 使用 go fmt 格式化代码
	@echo "格式化代码..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "Python 后端暂不执行 go fmt"; \
	else \
		cd $(BACKEND_DIR) && go fmt ./...; \
	fi

.PHONY: mod-tidy
mod-tidy: ## 整理 Go 模块依赖
	@echo "整理 Go 模块依赖..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "Python 后端暂不执行 go mod tidy"; \
	else \
		cd $(BACKEND_DIR) && go mod tidy; \
	fi

.PHONY: test-all
test-all: fmt lint lint-admin test test-admin build-admin ## 运行后端/前端全量验证
	@echo "所有测试与构建完成"

.PHONY: integration-smoke
integration-smoke: ## 运行集成回归演练（Webhook Replay + Nuxt 构建）
	@echo "运行 Webhook Replay Drill..."
	cd $(BACKEND_DIR) && go test ./internal/services/integration -run TestWebhookService_ReplayAttemptFlow -count=1
	$(MAKE) frontend-build

.PHONY: check
check: lint lint-admin test test-admin security-audit ## 运行 lint + test + security audit
