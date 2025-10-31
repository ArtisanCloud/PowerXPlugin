# PowerXPlugin 测试执行手册

**版本**: v1.0  
**最后更新时间**: 2025-10-31  
**关联文档**: `docs/test/testing_strategy.md`

---

## 1. 目的

本手册指导贡献者如何按当前实现执行测试套件，覆盖 Go 后端、Nuxt 前端、契约文件与 CLI 脚手架四个维度。阅读本文前，建议先了解整体策略（参见 `testing_strategy.md`），再依照下列操作运行具体检查。

---

## 2. 前置环境

- Go 1.21+（确保 `go env GOWORK` 指向本仓库）
- Node.js 18+ 与 npm 9+
- Playwright 1.48+（首次运行会自动安装浏览器二进制）
- curl / bash / python3（契约校验脚本需要标准工具）
- 可选：Chrome/Chromium 若需调试非无头模式

快速检查：

```bash
go version
node --version
npm --version
python3 --version
npx playwright --version
```

示例输出（供对照）：

```
go version go1.23.1 darwin/arm64
v22.13.0
9.3.1
Python 3.12.3
Version 1.56.1
```

---

## 3. 快速冒烟流程（约 30 秒）

```bash
# 仓库根目录
go test ./framework/backend/go/bootstrap/... -v
go test ./skeleton/backend/internal/routes/... -v
python3 -m json.tool docs/contracts/manifest.json > /dev/null
python3 -m json.tool docs/contracts/rbac.json > /dev/null
go build -o /tmp/px-plugin ./tools/cli/cmd/px-plugin

# 完成 Phase 3 后，可改用脚本入口
./scripts/testing/smoke.sh
# make test-smoke
```

通过以上命令或脚本可初步确认后端逻辑、契约文件与 CLI 构建无异常（示例输出末尾会打印 `=== Smoke workflow complete in Ns ===`，可直接记录耗时）。任何一步失败请参考第 7 节排查。

---

## 4. 完整回归流程

### 4.1 后端测试

```bash
go test ./framework/... ./skeleton/backend/... -v -coverprofile=coverage.out
go tool cover -func=coverage.out
```

> 输出覆盖率后，可执行 `mkdir -p tmp && go tool cover -html=coverage.out -o tmp/coverage.html` 生成 HTML 报告。

### 4.2 前端 E2E

1. 安装依赖与浏览器

   ```bash
   cd skeleton/web-admin
   npm install
   npx playwright install
   ```

2. 启动后端（新终端）

   ```bash
   cd /path/to/PowerXPlugin
   go run ./skeleton/backend/cmd/plugin
   ```

3. 启动前端（再开一终端）

   ```bash
   cd /path/to/PowerXPlugin/skeleton/web-admin
   npm run dev
   ```

4. 运行测试

```bash
PLAYWRIGHT_BASE_URL=http://localhost:3031 npx playwright test

# 完成 Phase 4 后，可改用脚本入口
./scripts/testing/regression.sh
# make test-regression
```

5. 停止服务（Ctrl+C），若失败可在 `skeleton/web-admin/test-results/` 查看报告。脚本模式会输出 `=== Regression workflow complete in Ns ===` 并保留 `tmp/regression-backend.log` / `tmp/regression-frontend.log`。

> 稳定性建议：确保 dev server 输出无 `PXAdminLayout` 等组件解析警告，再执行 Playwright。
>
> 脚本版本会自动启动后端 `go run ./skeleton/backend/cmd/plugin` 与前端 `npx nuxi preview --hostname 127.0.0.1 --port ${REGRESSION_FRONTEND_PORT}`，若未指定则自动选择空闲端口；随后等待 `http://127.0.0.1:8078/healthz` 与 `PLAYWRIGHT_BASE_URL` 可访问；相关日志保存在 `tmp/regression-backend.log` 与 `tmp/regression-frontend.log`。

### 4.3 契约校验

```bash
./scripts/testing/validate-contracts.sh
```

如需手动执行，可依次运行：

```bash
python3 -m json.tool docs/contracts/manifest.json > /dev/null
python3 -m json.tool docs/contracts/rbac.json > /dev/null
npx --yes @apidevtools/swagger-cli@4.0.4 validate docs/contracts/openapi.yaml

TEMP_DIR=$(mktemp -d)
pushd "$TEMP_DIR" >/dev/null
/path/to/PowerXPlugin/bin/px-plugin init com.powerx.temp-test --force --module github.com/example/temp
python3 -m json.tool com.powerx.temp-test/docs/contracts/manifest.json > /dev/null
python3 -m json.tool com.powerx.temp-test/docs/contracts/rbac.json > /dev/null
popd >/dev/null
rm -rf "$TEMP_DIR"
```

> 提示：设置 `KEEP_TEMP_DIR=1` 可保留脚本生成的临时目录以便排查。

### 4.4 CLI 验证

```bash
go build -o bin/px-plugin ./tools/cli/cmd/px-plugin
TMP_DIR=$(mktemp -d)
pushd "$TMP_DIR" >/dev/null
/path/to/PowerXPlugin/bin/px-plugin init com.powerx.smoke --force
tree com.powerx.smoke
popd >/dev/null
rm -rf "$TMP_DIR"
```

确认生成目录包含 backend、web-admin、docs/contracts 等关键文件。

---

## 5. 模块化测试入口

| 场景 | 命令 | 说明 |
|------|------|------|
| 后端快速回归 | `go test ./framework/... ./skeleton/backend/...` | 含单元与集成测试 |
| Playwright 单用例 | `npx playwright test tests/e2e/starter.spec.ts` | 需先设定 `PLAYWRIGHT_BASE_URL` |
| 契约变更验证 | 参考 4.3 | 修改 `docs/contracts/**` 后必跑 |
| CLI 模块改动 | 参考 4.4 | 确保 `px-plugin init` 无回归 |
| 测试采纳率审计 | `./scripts/testing/audit-test-adoption.sh` | 统计最近提交是否新增测试 |

> 计划中的 `make`/`scripts` 聚合命令详见 `testing_strategy.md`，落实后可替换为单行入口。

---

## 6. 编写与扩展测试用例

新增代码时，请同步补充对应的测试覆盖。下表列出了常用目录：

| 类型 | 目录示例 | 说明 |
|------|----------|------|
| Go 单元测试 | `framework/backend/go/<pkg>/*_test.go`<br>`skeleton/backend/internal/<pkg>/*_test.go` | 与被测文件同目录，命名为 `*_test.go`，可被 `go test ./...` 自动发现 |
| Playwright E2E | `skeleton/web-admin/tests/e2e/*.spec.ts` | 放在 `tests/e2e/` 下，命名 `*.spec.ts`，可被 `npx playwright test` 扫描 |
| CLI 示例 | `tools/cli/cmd` + `scripts/testing/*.sh` | CLI 新增命令时需同步脚本调用与文档 |

下面列举各层常见场景与最小示例，方便快速复制扩展。

> **目录约定提醒**  
> - Go 测试文件与实现同目录，命名为 `*_test.go`，便于 `go test ./...` 自动发现。  
> - 前端 E2E 用例统一放在 `skeleton/web-admin/tests/e2e/`，后续若加入 component/unit 测试，可在该目录下增设子目录。  
> - CLI 相关验证建议放在 `tools/cli` 同仓或 `scripts/` 目录，确保逻辑与脚手架输出一一对应。

### 6.1 Go 单元测试（framework / skeleton）

1. 在目标目录添加 `*_test.go` 文件，例如 `framework/backend/go/router/router_test.go`
2. 使用 `testing` 包编写 `Test` 函数：

   ```go
   package router

   import (
     "net/http"
     "testing"

     "github.com/powerx-plugin/framework/backend/go/bootstrap"
   )

   func TestRegisterPluginRoutesAddsPrefix(t *testing.T) {
     stub := &stubRouter{}
     RegisterPluginRoutes(&bootstrap.App{Router: stub}, func(r bootstrap.Router) {
       r.Handle(http.MethodGet, "/demo", func(ctx bootstrap.Context) {})
     })

     if got := stub.paths[0]; got != "/api/v1/demo" {
       t.Fatalf("route = %s, want /api/v1/demo", got)
     }
   }

   type stubRouter struct {
     paths []string
   }

   func (s *stubRouter) Group(rel string) bootstrap.Router { return s }
   func (s *stubRouter) Handle(method, path string, _ bootstrap.Handler) {
     s.paths = append(s.paths, path)
   }
   func (s *stubRouter) Use(_ ...bootstrap.Middleware) {}
   ```

3. 运行指定包测试：

   ```bash
   go test ./framework/backend/go/router -v
   ```

### 6.2 后端特性级（feature）回归

针对业务 Handler/Service，可结合 `httptest` 自行构造上下文：

```go
package handler_test

import (
  "encoding/json"
  "net/http"
  "net/http/httptest"
  "strings"
  "testing"

  "github.com/powerx-plugin/framework/backend/go/bootstrap"
  "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/handler"
  "github.com/powerx-plugin/powerxplugin/skeleton/backend/internal/service"
)

func TestPingHandler_ReturnsOK(t *testing.T) {
  req := httptest.NewRequest(http.MethodGet, "/api/v1/ping", nil)
  w := httptest.NewRecorder()

  handler := handler.NewPingHandler(service.NewPingService())
  ctx := &stubContext{w: w}
  handler.Handle()(ctx)

  if ctx.status != http.StatusOK {
    t.Fatalf("status = %d, want 200", ctx.status)
  }
  if body := w.Body.String(); !strings.Contains(body, "\"status\":\"ok\"") {
    t.Fatalf("unexpected body: %s", body)
  }
}

// 模板提示：将上面的 `handler`/`service` 替换为实际业务包，即可作为新增测试的起点。

type stubContext struct {
  w      *httptest.ResponseRecorder
  status int
}

var _ bootstrap.Context = (*stubContext)(nil)

func (c *stubContext) Param(string) string                     { return "" }
func (c *stubContext) Query(string) string                     { return "" }
func (c *stubContext) BindJSON(any) error                      { return nil }
func (c *stubContext) JSON(code int, v any) {
  c.Status(code)
  _ = json.NewEncoder(c.w).Encode(v)
}
func (c *stubContext) Status(code int) {
  c.status = code
}
```

执行局部用例：

```bash
go test ./skeleton/backend/internal/... -run TestPingHandler_ReturnsOK -v
```

> 样例中的 `stubContext` 仅实现必要方法，实际项目可根据需要扩展。

### 6.3 前端 E2E 用例

1. 在 `skeleton/web-admin/tests/e2e/` 添加新的 `*.spec.ts`：

   ```ts
   import { test, expect } from '@playwright/test';

   test('设置页显示版本号', async ({ page }) => {
     await page.goto('/_p/com.powerx.sample/admin/settings');
     await expect(page.getByText('当前版本')).toBeVisible();
   });
   ```

   > 以上片段可作为新 spec 的模板，替换 `goto` 地址与断言内容即可。

2. 启动后端与前端 dev server 后，仅运行该文件：

   ```bash
   cd skeleton/web-admin
   PLAYWRIGHT_BASE_URL=http://localhost:3031 npx playwright test tests/e2e/settings.spec.ts
   ```

> 建议在 `test.beforeEach` 内准备登录态或初始化数据，保证测试可重复。

### 6.4 CLI 生成物验证

扩展 `px-plugin` 模板时，可通过临时目录验证输出：

```bash
go build -o bin/px-plugin ./tools/cli/cmd/px-plugin
TMP_DIR=$(mktemp -d)
pushd "$TMP_DIR" >/dev/null
/path/to/repo/bin/px-plugin init com.powerx.feature --force
test -f com.powerx.feature/backend/internal/feature/new.go
test -f com.powerx.feature/web-admin/app/pages/_p/com.powerx.feature/admin/index.vue
popd >/dev/null
rm -rf "$TMP_DIR"
```

> 若需长期维护，可将上述逻辑萃取为 shell/Go 脚本并纳入 CI。

---

## 7. CI 集成建议

- 在 GitHub Actions 中并行运行后端测试、契约校验与 CLI 验证；E2E 可设置为非阻断任务或加重试。
- 缓存 Playwright 浏览器目录：`~/.cache/ms-playwright`
- 使用环境变量 `PLAYWRIGHT_BASE_URL` 绑定 dev server，配合等待逻辑（见策略文档中的 `wait_for_service` 示例）。
- 在 PR 模板中加入核对清单：后端测试、契约校验、CLI 验证、（如适用）E2E。

---

## 8. 常见问题排查

| 问题 | 可能原因 | 解决方案 |
|------|----------|----------|
| Playwright 报 “Failed to resolve component: PXAdminLayout” | 未安装 `@powerx-plugin/framework-admin` 依赖或 dev server 启动前 node_modules 残缺 | 重新执行 `npm install`，删除 `node_modules`/`package-lock.json` 后再装 |
| E2E 测试访问超时 | 前端/后端端口未就绪 | 启动测试前手动访问 `http://localhost:3031/_p/...` 与 `http://localhost:8078/api/v1/ping`，或实现等待函数 |
| CLI 生成命令失败 | 未 `go build` px-plugin 或 GOPATH 权限问题 | 先在仓库根执行 `go build -o bin/px-plugin ./tools/cli/cmd/px-plugin` |
| 契约校验报语法错误 | JSON 文件格式化异常 | 使用 `python3 -m json.tool <file>` 定位具体报错行 |
| 覆盖率下降 | 新增代码无测试 | 参考 `docs/test/testing_strategy.md` 中的改进建议，补充相应测试用例 |
| `scripts/testing/*.sh` 提前退出 | 依赖未装或日志被忽略 | 查看脚本输出末尾的 failing command，确认 `go`/`node`/`npx` 版本满足要求，必要时设置 `KEEP_TEMP_DIR=1` 保留排查现场 |

---

## 9. 下一步建设

- 在仓库新增 `scripts/` 目录与 Makefile 落地聚合命令
- 将契约校验、CLI 生成和 Playwright 流程接入 CI/CD
- 引入性能基准测试、测试仪表板等增强工具

如需对测试体系提出新需求或报告问题，请在仓库创建 Issue 或直接更新 `testing_strategy.md` 与本手册。

---

**保持习惯**：每次合并前至少执行一次 4.1 和 4.4；涉及前端改动时务必跑 4.2。祝测试顺利！🚀
