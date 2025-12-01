# Quickstart – Plugin Auth Integration

This guide shows how to run the Skeleton in both Delegated (PowerX host) and Local IAM modes.

## 1. Prerequisites
- Go 1.24, Node.js 18+, npm 9+
- PowerX Core dev instance (for Delegated mode)
- SQLite or PostgreSQL accessible locally (default: `skeleton/.cache/powerxplugin.db`)

## 2. Environment Variables
| Variable | Purpose | Delegated Example | Local Example |
|----------|---------|------------------|---------------|
| `POWERX_PROXY` | 注入来源，宿主=1 | `1` | `0`
| `POWERX_RBAC_DELEGATE` | 强制委托 | `true` | unset / `false`
| `POWERX_CORE_ENDPOINT` | 宿主 API | `http://powerx-core:8077` | optional
| `POWERX_AUTH_TOKEN` | 插件→宿主鉴权 | `eyJ...` | optional
| `POWERX_TENANT_ID` | 当前租户 | `tenant_123` | optional
| `PLUGIN_IAM_ADMIN_EMAIL` | Local 管理员 | optional | `admin@local.test`
| `PLUGIN_IAM_ADMIN_PASSWORD` | Local 管理员口令 | optional | `S3cret!!`
| `POWERX_RUN_MIGRATE` | 启动自动 migrate | `true` | `true`

## 3. Delegated Mode (PowerX Host)
```bash
# 后端
cd skeleton/backend
POWERX_PROXY=1 POWERX_RBAC_DELEGATE=true \
POWERX_CORE_ENDPOINT="http://localhost:8077" \
POWERX_AUTH_TOKEN="dev-token" \
go run ./cmd/plugin

# 前端
cd ../web-admin
npm install
npm run dev
```
- 登录入口：`http://localhost:3031/users/login`
- 成功后访问受保护页面（如 `/intro`），请求应携带宿主 Token。
- 断开 PowerX Core 以验证 fail-closed：登录会提示“宿主认证不可用”。

## 4. Local Mode (Standalone)
```bash
cd skeleton/backend
export POWERX_PROXY=0
export POWERX_RBAC_DELEGATE=false
export PLUGIN_IAM_TENANT_KEY=px_local
export PLUGIN_IAM_TENANT_NAME="Local Tenant"
export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
go run ./cmd/database/main.go setup
POWERX_RUN_MIGRATE=true go run ./cmd/plugin
```
- 登录凭证：`PLUGIN_IAM_ADMIN_EMAIL` / `PLUGIN_IAM_ADMIN_PASSWORD`。
- 默认会生成 `iam_departments` 与 `iam_permissions` 示例数据，并把管理员绑定到 `system.admin` 角色。
- 运行 `cd skeleton/web-admin && npm install && npm run dev` 后，即可在 `/users/login` 输入本地管理员完成登录。
- 自动化校验：
  - `cd skeleton/backend && go test ./...`（覆盖 Local IAM store 行为）。
  - `cd skeleton/web-admin && npm run test:unit`（`useAuth` fallback 行为）。
  - Playwright Local 测试：
    ```bash
    cd skeleton/web-admin
    PLAYWRIGHT_LOCAL_IAM=1 \
    PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \
    PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \
    npm run test:e2e -- auth-local
    ```
    需要先启动本地后端。

## 5. Template Sync & CLI Check
```bash
npm run sync:templates
cd tools/cli && go run ./cmd/px-plugin init dev.plugin.test
```
- 新建的插件工程应自动包含 `useAuth`、auth pages、中间件以及后台 Auth routes。
- 运行 `go run ./cmd/plugin` + `npm run dev` 验证 CLI 产物的登录流程。

## 6. Observability Checklist
- 确认 `plugin_iam_mode`, `plugin_auth_login_total`, `plugin_auth_refresh_total`, `plugin_auth_logout_total`, `plugin_iam_delegate_errors_total` 指标在 `/metrics` 暴露。
- `request_trace` 日志需含 `iam_mode`, `auth`, `tenant_uuid`, `user_id`, `trace_id`。
- 打开/关闭 Local 模式应各自写一条 Info 日志。

## 7. 验收与性能
- 验证命令：
  ```bash
  npm run lint
  cd skeleton/backend && go test ./...
  npm --prefix skeleton/web-admin run test:unit
  npm run sync:templates -- --check
  ```
- Delegated E2E（需另启 `npm --prefix skeleton/web-admin run dev`）：`npm --prefix skeleton/web-admin run test:e2e -- auth-delegated`
- Local E2E：`PLAYWRIGHT_LOCAL_IAM=1 npm --prefix skeleton/web-admin run test:e2e -- auth-local`
- 性能参考：在 Apple M3 Pro + Chromium Headless 上，Delegated 登录单次 ~1.8s；Postgres 15 中执行 `go run ./cmd/database/main.go setup` 约 4.6s（包含 IAM migrate+seed）。详细说明见 `docs/guides/develop/auth.md#6-性能与耗时`。

## 8. CLI Package / Publish 快速演练

完成 IAM 场景后，可用同一仓库演练发布链路：

1. 生成 artefact：`px-plugin package --entry <plugin>`（必要时使用 `--skip-frontend/--skip-backend`）。
2. 在 `~/.px-plugin/config.json` 写入：
   ```json
   {
     "publishApi": {
       "baseUrl": "http://127.0.0.1:8077/api/v1",
       "apiKey": "dev-registry-token"
     }
   }
   ```
   或通过 `PX_PUBLISH_API_BASE` / `PX_PUBLISH_API_TOKEN` 覆盖。
3. 上传：`px-plugin publish --entry <plugin> --channel dev --notes "feat: auth"`。命令会输出 `publishId` 与审核链接，随后即可在 PowerX Marketplace / 插件管理后台看到「待审核版本」。
4. 在宿主后台安装/启用该版本后，即可回到 `px-plugin dev --watch` 热加载流程，形成「安装一次 → 热载调试 → package/publish」闭环。
