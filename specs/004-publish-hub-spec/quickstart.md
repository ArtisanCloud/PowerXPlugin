# Quickstart — Publish Hub Feature

## 1. 环境准备
1. 安装 Go 1.24+、Node.js 18+、npm 9+、Playwright 1.48+。
2. 运行 `npm install && npm run bootstrap` 以确保 `tools/cli` 与 `sdk/workspace` 依赖同步。
3. 配置凭据：
   - `px auth configure` 产出 mTLS 证书到 `~/.powerx/cli`。
   - `PX_MARKETPLACE_API_URL`, `PX_MARKETPLACE_TOKEN`, `PX_ARTIFACT_STORE_*`、`PX_SIGNING_ENDPOINT` 写入环境变量或 `~/.powerx-plugin/config.json`。
   - 离线签名准备 `cert.pem` 或 KMS Key (`--kms-key-id`).
4. 确保 Feature Flags：`PX_PLUGIN_DEV_MODE`, `PX_PLUGIN_PUBLISH`, `PX_MARKET_PUBLISH_ENABLED`, `PX_MARKET_OFFLINE_UPLOAD`, `PX_PLUGIN_HUB_ENABLED` 均开启。

## 2. Dev 热加载链路
1. 在插件仓执行 `px-plugin dev --watch --tenant demo-tenant --entry ./dist`（实现参考 `tools/cli/src/commands/dev/watch.ts`）。
2. CLI 将读取 `dist/manifest.json` 并向 Dev API (`tools/cli/src/runtime/hotreload/session.ts`) 发送 register 请求，输出 `sessionId`, `reloadToken`, Admin 调试地址。
3. 修改代码并保存，CLI watcher 会在 250ms 去抖后聚合 diff，调用 Dev API reload；确认构建耗时 ≤2s，Admin SSE 获取 `reload` 日志。
4. 结束调试：运行 `px-plugin dev --stop`（或 `CTRL+C`），Dev API 删除 session 并由 `framework/backend/go/runtime/devapi/handlers/dev_plugins.go` 记录审计，日志保留 7 天。

## 3. 在线发布链路
1. 运行 `npm run lint && npm test && go test ./...`，确保预检输入准备就绪。
2. 执行 `px-plugin publish --channel stable --notes ./CHANGELOG.md`。
3. 检查 `publish-receipt.json`（含 `publishId`, `versionId`, `reviewQueueId`）。
4. 进入 Marketplace 审核队列查看状态；若失败，根据 CLI 提示修复。
5. 审核通过后确认租户在 30 分钟内收到通知。

## 4. 离线发布链路
1. 运行 `px-plugin dist --target offline --sign ./cert.pem`。
2. 验证输出：`.pxp`, `integrity.txt`, `manifest.signature`, `dist/report.json`, `dist/audit.log`。
3. 运维在 Marketplace 离线入口上传，并勾选目标租户白名单。
4. 审核通过后，租户管理员使用 `install/local` 导入；失败时 5 分钟内执行回滚。

## 5. Admin 安装 / 回滚验证
1. 在线安装：调用 `framework/backend/go/runtime/admin/handlers/plugins_install_url.go` 暴露的 API（或 Admin 界面）选择远程版本。观察响应 `status=installing` 并查看 `framework/backend/go/runtime/admin/services/plugin_deployer.go` 打印的 deploymentId。
2. 离线导入：使用 `plugins_install_local.go` 上传 `.pxp`，确保 CLI 输出的 manifest/签名/密钥信息被校验。
3. 回滚测试：人为构造失败（或调用 `PluginDeployer.Rollback`），确认 `framework/backend/go/runtime/admin/events/install_events.go` 记录 `tenant.install.rolled_back`，并在 5 分钟内恢复上一版本。
4. UI 校验：在 `examples/starter/web-admin/app/pages/plugins/manage.vue` 刷新版本列表，验证安装状态与回滚按钮可正常触发。

## 5. Telemetry & 验证
1. 使用 `npm run e2e:dev-hotload`、`npm run e2e:publish-online` 运行端到端验证（如已提供）。
2. 通过 Grafana / Workflow Metrics 检查以下指标：
   - `dev.hotload.cli_reload_duration_ms`
   - `plugin.publish.duration_ms`
   - `plugin.offline.approval.duration`
   - `plugin.install.success_rate`
3. Playwright 脚本验证 Admin 安装/回滚 UI。

## 6. 交付清单
- 更新 `docs/contracts/` 中的 OpenAPI/Schema，并生成新的版本记录。
- 在 `examples/` 中运行 `px-plugin init demo-plugin` 验证 CLI 输出。
- 将产生的 `.pxp`、`publish-receipt.json`、`dist/report.json` 上传到审查工单，附带 SLA 结果。
