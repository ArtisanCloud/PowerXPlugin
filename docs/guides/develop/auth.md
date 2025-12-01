# 开发指南：插件 Auth 集成

面向插件开发者，说明如何在 PowerX Skeleton 中实现和验证 Delegated / Local IAM、Token 生命周期、指标与调试流程。

## 1. 模式切换速查
| 模式 | 关键环境变量 | 说明 |
|------|--------------|------|
| Delegated (宿主) | `POWERX_PROXY=1` 或 `POWERX_RBAC_DELEGATE=true`<br>`POWERX_CORE_ENDPOINT`<br>`POWERX_AUTH_TOKEN` | 所有 `/api/v1/auth` 请求代理到宿主 `/admin/user/auth/*`，Token/组织信息完全复用 PowerX Core。宿主故障时 fail-closed。|
| Local (Standalone) | `POWERX_PROXY=0`<br>`PLUGIN_IAM_TENANT_*`<br>`PLUGIN_IAM_ADMIN_*` | 插件自持 IAM 表（`iam_*`），`go run ./cmd/database/main.go setup` 会创建默认租户/管理员，前端通过同一 `/users/login` 页面登录。|

> **提示**：`models.InitSchemaFrom` 会根据配置清空 schema 前缀，SQLite/内存模式无需额外设置；PostgreSQL 场景可设置 `POWERX_DB_SCHEMA=px_com_powerx_plugins_base` 避免冲突。若未显式设置 `PLUGIN_IAM_ADMIN_EMAIL/PASSWORD`，seeder 会默认注入 `admin@local.test` / `S3cret!!`（仅用于本地调试，生产环境务必覆盖）。本地模式默认同样强制校验 Authorization Header，如需临时跳过，可设置 `POWERX_AUTH_OPTIONAL=true`。

## 2. 前端 Token 生命周期
- `useAuth` 将 `access_token`、`refresh_token`、`expires_at` 保存在 localStorage + `token` Cookie，刷新失败或宿主 503 时会调用 `failClosed()`，清空状态并在登录页展示“宿主认证不可用”提示。
- `storage` 事件会同步跨 Tab 的登录/登出：任一 Tab 清理 Token，其它 Tab 将强制跳回 `/users/login`。
- `auth.global.ts` 中间件会在首屏调用 `ensureFreshToken()`，若 token 丢失则将原目标地址写入 `redirect` query。

## 3. 后端组件
| 文件 | 责任 |
|------|------|
| `internal/services/iam/local_store.go` | Local 模式的 `IAMDirectory` 实现，支持 Login/Refresh/Logout、JWT 签发、RefreshToken 持久化。|
| `internal/services/authproxy/delegated_client.go` | Delegated 模式 HTTP 代理，负责附带 `POWERX_AUTH_TOKEN` 调宿主 `/admin/user/auth/*`。|
| `internal/transport/http/public/auth_handler.go` | `/api/v1/auth/login|refresh|logout|me/context`；根据 `IAMMode` 决定走 Proxy 或 Local，实现 fail-closed 和指标打点。|
| `internal/observability/auth/metrics.go` | 输出 `plugin_auth_login_total`、`plugin_auth_refresh_total`、`plugin_auth_logout_total`、`plugin_iam_mode`、`plugin_iam_delegate_errors_total`，Prometheus 入口 `/api/v1/admin/runtime/metrics`。|
| `internal/transport/http/middleware/request_trace.go` | 新增 `iam_mode`, `tenant_uuid`, `user_id`, `trace_id` 字段，定位跨模式问题。|

## 4. 指标与日志
- **核心指标**（全部可在 `/api/v1/admin/runtime/metrics` 查看）：
  - `plugin_auth_login_total{mode,result}`
  - `plugin_auth_refresh_total{mode,result}`
  - `plugin_auth_logout_total{mode}`
  - `plugin_iam_delegate_errors_total{type}`
  - `plugin_iam_mode{mode="delegated|local"}`
- **日志**：`POWERX_DEBUG_TRAFFIC=1` 时可看到 `[PLUGIN-REQ-TRACE]` 日志（auth_mode/tenant/user/trace/ip/UA），用于追踪宿主联调流量。

## 5. 验证矩阵
| 步骤 | 命令 |
|------|------|
| Go 单测（包含 Local IAM store、Auth Handler、观测指标） | `cd skeleton/backend && go test ./...` |
| 前端单测（`useAuth`逻辑） | `npm --prefix skeleton/web-admin run test:unit` |
| Lint/Snapshot | `npm run lint` |
| 模板同步 | `npm run sync:templates` |
| Delegated E2E（需先启动后端/前端，见 5.1） | `npm --prefix skeleton/web-admin run test:e2e -- auth-delegated` |
| Local E2E（需先启动后端/前端，见 5.2） | `PLAYWRIGHT_LOCAL_IAM=1 npm --prefix skeleton/web-admin run test:e2e -- auth-local` |

### 5.1 Delegated Playwright 步骤
1. **启动后端（宿主代理）**
   ```bash
   cd skeleton/backend
   POWERX_PROXY=1 POWERX_RBAC_DELEGATE=true \
   POWERX_CORE_ENDPOINT="http://localhost:8077" \
   POWERX_AUTH_TOKEN="dev-token" \
   go run ./cmd/plugin
   ```
   替换为实际宿主 Endpoint/Token；若依赖 PowerX Core，请确认 Core Dev 环境已启动。
2. **启动前端开发服务器**
   ```bash
   cd skeleton/web-admin
   npm install
   npm run dev   # 默认 http://localhost:3031
   ```
3. **运行 Playwright**（另开终端）
   ```bash
   npm --prefix skeleton/web-admin run test:e2e -- auth-delegated
   ```

### 5.2 Local Playwright 步骤
1. **迁移 + 启动 Local IAM**
   ```bash
   cd skeleton/backend
   export POWERX_PROXY=0
   export POWERX_RBAC_DELEGATE=false
   export PLUGIN_IAM_TENANT_KEY=px_local
   export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
   export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
   go run ./cmd/database/main.go setup
   go run ./cmd/plugin
   ```
   > 看到 `sqlite 环境仅迁移 IAM + 插件核心表，跳过 30 张业务表`、`migrate ok`、`seed ok` 即表示初始化成功。接下来保持此终端运行 `go run ./cmd/plugin` 作为本地 IAM API 服务。
   > ⚠️ 默认 SQLite 驱动只会迁移 IAM + 插件核心表（`plugin_credentials` / `plugin_tenant_ext` / `template`），若需要 marketplace/ops 等全量表结构，请改用 Postgres。
2. **启动前端 dev server**（同上 `npm run dev`）。
3. **运行 Playwright**
   ```bash
   PLAYWRIGHT_LOCAL_IAM=1 \
   PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \
   PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \
   npm --prefix skeleton/web-admin run test:e2e -- auth-local
   ```

## 6. 性能与耗时
| 指标 | 测量方法 | 结果（Apple M3 Pro，本地 Chromium Headless） |
|------|-----------|-----------------------------------------------|
| Delegated 登录 p90 | 在运行 `npm --prefix skeleton/web-admin run dev` 后执行 `time PLAYWRIGHT_TEST_BASE_URL=http://localhost:3031 npm --prefix skeleton/web-admin run test:e2e -- auth-delegated`，读取 `Test finished` 时长 | ~1.8s/次（3 轮平均） |
| Token 轮询成功率 | 观察 `plugin_auth_refresh_total{mode,result}` 与 `plugin_auth_delegate_errors_total` 的 ratio | 3 次刷新全部成功（0 错误） |
| Local `go run ./cmd/database/main.go setup` | 针对 Postgres 15：`time POWERX_DB_DSN=postgres://... go run ./cmd/database/main.go setup` | ~4.6s（migrate+seed，本地容器） |

> **说明**：SQLite 驱动不支持 `uuid DEFAULT gen_random_uuid()`，若需在 SQLite 中演练，请先把相关列类型改为 `text` 并去掉默认值，或直接切换到 Postgres。强烈建议在 Postgres 中验证最终 Schema。

## 7. 故障排查
详见 `docs/operations/runbooks/auth-troubleshooting.md`，涵盖：
- Delegated 模式 fail-closed、服务 Token 失效。
- Local 模式管理员注入、RefreshToken 强制失效。
- 多 Tab storage 事件与提示。
- 指标缺失或 Prometheus 采集异常。

## 8. 常见 FAQ
1. **为什么本地登录突然跳回登录页？** 另一浏览器 Tab 清除了 token，storage 事件触发 `clearAuth` 并携带 `redirect` 参数；查看登录页的提示文本。  
2. **如何判断宿主返回 503？** 查看前端 Alert，或查询 `plugin_iam_delegate_errors_total{type="unavailable"}` 是否递增。  
3. **Local 模式可以自定义角色/权限吗？** 可以，向 `iam_permissions` 插入新行并更新 `iam_role_permissions`；`CheckPermission` 默认只允许 `system.admin` 通过。  
4. **同步模板后 CLI 生成器是否立即可用？** 是的，`npm run sync:templates` 会把 Skeleton → `scaffold/templates` → `tools/cli/internal/templates` 的改动全部同步，之后执行 `px-plugin init` 即可生成包含 Auth 的脚手架。
