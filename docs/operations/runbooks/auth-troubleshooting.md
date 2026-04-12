# Runbook – Auth Troubleshooting

掌握插件在 Delegated / Local IAM 模式下的常见故障处理办法。

## 快速定位
| 现象 | 可能原因 | 排查步骤 |
|------|----------|----------|
| 登录页提示“宿主认证不可用” | `POWERX_CORE_ENDPOINT` 无法访问或 `POWERX_AUTH_TOKEN` 失效 | 查看插件日志 `auth_mode=delegated` 的错误，确认宿主 API 返回码；必要时在宿主上刷新服务 Token |
| 登录永远 401（Local） | 默认管理员凭证无效（未覆盖或密码输错） | 检查环境变量；若未设置将使用 `admin@local.test` / `S3cret!!`；运行 `go run ./cmd/database/main.go seed` 重新注入管理员；确认 bcrypt hash 长度 |
| 刷新 token 失败 | refresh token 过期或被删除 | 在 DB `iam_refresh_tokens` 表检查 `expires_at`；确保 `PLUGIN_IAM_REFRESH_TTL` 足够长 |
| `/auth/me/context` 返回空 | 请求缺少 `Authorization` header 或本地 JWT Secret 与配置不一致 | 确认前端 `localStorage` 存在 `access_token`；检查 `context.hmac_secret` |

## Delegated 模式检查清单
1. `IAMMode=delegated`（宿主标准场景通常同时 `POWERX_PROXY=1`）。
2. `POWERX_CORE_ENDPOINT` 可从插件容器访问（`curl $POWERX_CORE_ENDPOINT/_health`）。
3. `POWERX_AUTH_TOKEN` 在宿主侧仍有效（查看宿主日志 `service-token` 相关提示）。
4. 插件日志需出现 `IAM mode resolved` 且 `mode=delegated`。
5. 如需模拟宿主故障，可停止 Core，前端应提示 fail-closed。

## Local 模式检查清单
1. `IAMMode=local` 且 `PLUGIN_IAM_ADMIN_*`、`PLUGIN_IAM_TENANT_*` 均已设置。
2. 运行 `go run ./cmd/database/main.go setup` 以创建 `iam_*` 表和管理员账户。
3. 默认管理员：`PLUGIN_IAM_ADMIN_EMAIL` / `PLUGIN_IAM_ADMIN_PASSWORD`。
4. 本地 JWT 依赖 `context.hmac_secret`、`context.issuer`、`context.audience`；需要与前端/中间件一致。
5. `iam_refresh_tokens` 表记录 refresh token；若需强制登出，可删除对应 `token_hash`。
6. Playwright 验证：
   ```bash
   PLAYWRIGHT_LOCAL_IAM=1 \
   PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \
   PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \
   npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-local
   ```
7. 多 Tab 同步：任一浏览器 Tab 清除 token（storage event）后，其他 Tab 会自动跳转登入页，可在控制台执行 `localStorage.removeItem('access_token')` 验证。

## 指标
- `plugin_iam_mode{mode="delegated|local"}` – 当前模式。
- `plugin_auth_login_total{mode}` – 登录次数。
- `plugin_auth_refresh_total{mode}` – 刷新次数及成功率，可结合 `result` 标签区分成功/失败。
- `plugin_auth_delegate_errors_total{type}` – 委托模式错误（网络/401/其它）。
- `plugin_auth_logout_total{mode}` – 登出次数。

若指标缺失，确认 `monitoring.metrics.enabled` 为 true 并已将 `/api/v1/admin/runtime/metrics` 暴露给 Prometheus。
