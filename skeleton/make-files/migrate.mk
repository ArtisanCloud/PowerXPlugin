# migrate.mk 管理数据库迁移与数据初始化任务

.PHONY: migrate
migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		cd $(BACKEND_DIR) && alembic upgrade head; \
	else \
		cd $(BACKEND_DIR) && go run ./cmd/database/main.go migrate; \
	fi

.PHONY: migrate-cmd
migrate-cmd: migrate ## 兼容旧命令，内部调用 migrate 目标
	@:

.PHONY: seed
seed: ## 运行数据种子脚本
	@echo "运行数据种子..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		echo "Python 后端暂不提供 seed 命令"; \
	else \
		cd $(BACKEND_DIR) && go run ./cmd/database/main.go seed; \
	fi

.PHONY: setup-db
setup-db: ## 执行迁移并填充初始数据
	@echo "运行迁移并填充初始数据..."
	@if [ "$(BACKEND)" = "fastapi" ]; then \
		cd $(BACKEND_DIR) && alembic upgrade head; \
	else \
		cd $(BACKEND_DIR) && go run ./cmd/database/main.go setup; \
	fi

.PHONY: reset-db
reset-db: ## 重置数据库（危险操作）
	@echo "警告: 这将删除所有数据！"
	@read -p "确定要继续吗？[y/N] " confirm; \
	if [ "$$confirm" = "y" ] || [ "$$confirm" = "Y" ]; then \
		echo "重置数据库..."; \
		if [ "$(BACKEND)" = "fastapi" ]; then \
			cd $(BACKEND_DIR) && python scripts/reset_db.py; \
		else \
			cd $(BACKEND_DIR) && go run ./cmd/database/main.go refresh; \
		fi; \
	else \
		echo "操作已取消"; \
	fi
