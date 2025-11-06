# Testing Strategy Quickstart

## 1. 环境准备
- Go 1.24+
- Node.js 18+ / npm 9+
- Playwright 1.48+ (`npx playwright install`)
- Python 3（用于 JSON 校验）

## 2. 安装依赖
```bash
npm install --prefix skeleton/web-admin
npx playwright install --prefix skeleton/web-admin
```

## 3. 运行冒烟测试
```bash
./scripts/testing/smoke.sh
# 或使用 Makefile
make test-smoke
```

> 实测（macOS / Go 1.24 / Node 22）：约 3 秒完成，脚本末尾会打印 `=== Smoke workflow complete in Ns ===`，可作为 SC-001 的佐证。

## 4. 执行全量回归
```bash
./scripts/testing/regression.sh
# 或
make test-regression
```

> 实测同环境：约 12 秒完成，Playwright 报告与后端/前端日志可在 `tmp/` 与 `skeleton/web-admin/test-results/` 中查看。

## 5. 查看产物
- 覆盖率：`tmp/coverage.html`
- Playwright 报告：`skeleton/web-admin/test-results/`
- 临时 CLI 项目：`/tmp/powerx-*`（脚本默认清理）

## 6. 添加新测试
- Go：在对应目录创建 `*_test.go` 并使用 `go test ./path/to/pkg`
- Playwright：在 `skeleton/web-admin/tests/e2e/` 增加 `*.spec.ts`
- CLI：扩展 `scripts/testing` 中的脚本并更新文档

更多细节请阅读 `docs/test/testing_strategy.md` 与 `docs/test/testing_usage.md`。
