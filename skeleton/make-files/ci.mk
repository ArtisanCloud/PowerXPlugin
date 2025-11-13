# ci.mk 汇总 CI 相关快捷目标

.PHONY: ci-integration
ci-integration: ## 运行集成特性 CI 流程
	scripts/ci/integration.sh

