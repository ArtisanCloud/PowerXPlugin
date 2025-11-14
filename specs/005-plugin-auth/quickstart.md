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
export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
go run ./cmd/database/main.go setup
POWERX_RUN_MIGRATE=true go run ./cmd/plugin
```
- 登录凭证：即 `PLUGIN_IAM_ADMIN_*` 值。
- 可通过 `go test ./...` + `npm run test` 验证基本逻辑。
- Playwright: `npm run test:e2e -- auth`（需要在 `web-admin` 中配置）。

## 5. Template Sync & CLI Check
```bash
npm run sync:templates
cd tools/cli && go run ./cmd/px-plugin init dev.plugin.test
```
- 新建的插件工程应自动包含 `useAuth`、auth pages、中间件以及后台 Auth routes。
- 运行 `go run ./cmd/plugin` + `npm run dev` 验证 CLI 产物的登录流程。

## 6. Observability Checklist
- 确认 `plugin_iam_mode`, `plugin_auth_login_total`, `plugin_iam_delegate_errors_total` 指标在 `/metrics` 暴露。
- `request_trace` 日志需含 `auth_mode`, `tenant_id`, `user_id`, `trace_id`。
- 打开/关闭 Local 模式应各自写一条 Info 日志。
