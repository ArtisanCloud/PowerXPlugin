# Testing Scripts

This directory hosts automation used by the PowerXPlugin testing strategy.

## Available Scripts

- `smoke.sh`: orchestrates minimal Go/unit/contract/CLI checks and produces coverage artifacts.
- `regression.sh`: runs the full regression workflow (includes smoke, full Go suite, Playwright E2E).
- `validate-contracts.sh`: shared validator for manifest/RBAC/OpenAPI consistency。
- `audit-test-adoption.sh`: reserved for Phase 6，用于统计测试采纳率（即将实现）。

See `specs/002-testing-strategy/plan.md` 与 `docs/test/testing_usage.md` 获取最新流程说明。

## Hooking New Tests

- **Go/CLI**：新增 `*_test.go` 文件后，确保 `smoke.sh` / `regression.sh` 涵盖对应包；必要时在脚本中追加命令。
- **Playwright**：将新 spec 放入 `skeleton/web-admin/tests/e2e/`，脚本会自动检测；如需额外环境变量，在运行 `make test-regression` 前导出即可。
- **Artifacts**：所有脚本产物统一存储在 `tmp/` 与 `skeleton/web-admin/test-results/`，并由 CI 上传。

## Environment Variables

- `SMOKE_TIMEOUT`：`make test-smoke` 的超时时间（默认 300 秒）。
- `REGRESSION_TIMEOUT`：`make test-regression` 的超时时间（默认 3600 秒）。
- `PLAYWRIGHT_BASE_URL`：Playwright 访问的前端地址，默认 `http://127.0.0.1:3000`。
- `KEEP_TEMP_DIR`：设为 `1` 时保留脚本生成的临时目录用于排查。
- `PX_PLUGIN_BIN`：自定义 `px-plugin` 可执行路径，默认 `bin/px-plugin`。
