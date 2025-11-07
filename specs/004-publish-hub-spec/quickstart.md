# Quickstart — Publish Hub Feature

## 1. 环境准备
1. 安装 Go 1.24+、Node.js 18+、npm 9+、Playwright 1.48+。
2. 运行 `npm install && npm run bootstrap` 以确保 `tools/cli` 与 `sdk/workspace` 依赖同步。
3. 配置凭据：
   - mTLS 证书配置（重要！）：
     - 运行 `px auth configure` 产出 mTLS 证书到 `~/.powerx/cli`，包括：
       - `client.crt` (客户端证书)
       - `client.key` (客户端私钥)
       - `ca.crt` (CA 根证书)
     - 验证证书：`openssl x509 -in ~/.powerx/cli/client.crt -text -noout`
     - 确保证书有效期 > 30 天，否则需要更新证书
   - 配置环境变量：
     - `PX_MARKETPLACE_API_URL`, `PX_MARKETPLACE_TOKEN`, `PX_ARTIFACT_STORE_*`、`PX_SIGNING_ENDPOINT` 写入环境变量或 `~/.powerx-plugin/config.json`
     - mTLS 相关：`PX_MTLS_CERT_PATH=~/.powerx/cli/client.crt`, `PX_MTLS_KEY_PATH=~/.powerx/cli/client.key`, `PX_MTLS_CA_PATH=~/.powerx/cli/ca.crt`
   - 离线签名准备 `cert.pem` 或 KMS Key (`--kms-key-id`)。
4. 确保 Feature Flags：`PX_PLUGIN_DEV_MODE`, `PX_PLUGIN_PUBLISH`, `PX_MARKET_PUBLISH_ENABLED`, `PX_MARKET_OFFLINE_UPLOAD`, `PX_PLUGIN_HUB_ENABLED` 均开启。

## 2. Dev 热加载链路
1. 在插件仓执行 `px-plugin dev --watch --tenant demo-tenant --entry ./dist`（实现参考 `tools/cli/src/commands/dev/watch.ts`）。
2. CLI 将读取 `dist/manifest.json` 并向 Dev API (`tools/cli/src/runtime/hotreload/session.ts`) 发送 register 请求，输出 `sessionId`, `reloadToken`, Admin 调试地址。
   - **mTLS 验证**：CLI 会自动加载 mTLS 证书并建立双向 TLS 连接
   - 如果 mTLS 握手失败，会自动重试 3 次（指数退避：1s, 2s, 4s）
3. 修改代码并保存，CLI watcher 会在 250ms 去抖后聚合 diff，调用 Dev API reload；确认构建耗时 ≤2s，Admin SSE 获取 `reload` 日志。
4. 结束调试：运行 `px-plugin dev --stop`（或 `CTRL+C`），Dev API 删除 session 并由 `framework/backend/go/runtime/devapi/handlers/dev_plugins.go` 记录审计，日志保留 7 天。

## 3. 在线发布链路
1. 运行 `npm run lint && npm test && go test ./...`，确保预检输入准备就绪。
2. 执行 `px-plugin publish --channel stable --notes ./CHANGELOG.md`。
3. 检查 `publish-receipt.json`（含 `publishId`, `versionId`, `reviewQueueId`）。
4. 进入 Marketplace 审核队列查看状态；若失败，根据 CLI 提示修复。
5. 审核通过后确认租户在 30 分钟内收到通知。
6. **SLA 监控**：在线发布审核应在 4 小时内完成，可在 Grafana 中查看 `plugin_publish_pipeline_duration_ms` 指标。

## 4. 离线发布链路
1. 运行 `px-plugin dist --target offline --sign ./cert.pem`。
2. 验证输出：`.pxp`, `integrity.txt`, `manifest.signature`, `dist/report.json`, `dist/audit.log`。
3. 运维在 Marketplace 离线入口上传，并勾选目标租户白名单。
4. 审核通过后，租户管理员使用 `install/local` 导入；失败时 5 分钟内执行回滚。
5. **离线签名验证**：系统会验证 RSA-PSS 签名和密钥封装，确保包完整性
6. **SLA 监控**：离线审核应在 1 个工作日内完成，可在 Grafana 中查看 `plugin_offline_approval_duration_minutes` 指标。

## 5. Admin 安装 / 回滚验证
1. 在线安装：调用 `framework/backend/go/runtime/admin/handlers/plugins_install_url.go` 暴露的 API（或 Admin 界面）选择远程版本。观察响应 `status=installing` 并查看 `framework/backend/go/runtime/admin/services/plugin_deployer.go` 打印的 deploymentId。
2. 离线导入：使用 `plugins_install_local.go` 上传 `.pxp`，确保 CLI 输出的 manifest/签名/密钥信息被校验。
3. **回滚验证（重点）**：
   - **自动回滚测试**：
     - 人为构造安装失败（模拟网络错误或返回 500）
     - 观察日志：`tenant.install.failed` 事件被记录
     - 确认 5 分钟（300 秒）后自动触发回滚
     - 验证回滚状态：使用 `GET /admin/deployments/{deploymentId}` 检查 `status=rolled_back`
     - 检查指标：`plugin_install_rollback_latency_seconds` 应记录回滚耗时
   - **手动回滚测试**：
     - 成功安装后，调用 `PluginDeployer.Rollback(deploymentId)` 或使用 Admin UI
     - 确认回滚在 1 分钟内完成
     - 验证 `tenant.install.rolled_back` 事件被记录，标记 `autoTriggered=false`
   - **回滚后验证**：
     - 检查 `framework/backend/go/runtime/admin/events/install_events.go` 记录完整事件链
     - 验证上一版本恢复，所有功能正常
4. UI 校验：在 `examples/starter/web-admin/app/pages/plugins/manage.vue` 刷新版本列表，验证安装状态与回滚按钮可正常触发。
5. **SLA 监控**：所有回滚操作应在 5 分钟内完成，可在 Grafana 中查看 `plugin_install_rollback_latency_seconds` 指标。

## 5. Telemetry & 验证
1. 使用 `npm run e2e:dev-hotload`、`npm run e2e:publish-online` 运行端到端验证（如已提供）。
2. 通过 Grafana / Workflow Metrics 检查以下指标：
   - `dev.hotload.cli_reload_duration_ms` (目标：≤2s)
   - `plugin_publish_pipeline_duration_ms` (目标：95th percentile ≤4h)
   - `plugin_offline_approval_duration_minutes` (目标：95th percentile ≤1d)
   - `plugin_install_rollback_latency_seconds` (目标：≤300s)
   - `plugin_deployments_total` (成功/失败率统计)
3. **mTLS 验证**：
   - 查看 Dev API 日志，确认 mTLS 握手成功
   - 检查证书验证日志：`client certificate verification succeeded`
   - 监控证书过期提醒
4. Playwright 脚本验证 Admin 安装/回滚 UI。

## 6. 交付清单
- 更新 `docs/contracts/` 中的 OpenAPI/Schema，并生成新的版本记录。
- 在 `examples/` 中运行 `px-plugin init demo-plugin` 验证 CLI 输出。
- 将产生的 `.pxp`、`publish-receipt.json`、`dist/report.json` 上传到审查工单，附带 SLA 结果。
- **安全交付**：
  - 确认 mTLS 证书配置正确
  - 导出 CA 公钥供运维部署
  - 提交证书轮换计划文档

## 7. 最终 Checklist（建议截图保留）
- `px-plugin publish/dist/dev` 命令均执行一次并保存 CLI 输出截图。
- Marketplace 在线/离线审核界面（`review.vue` / `offline-review.vue`）截图 + 审核结果记录。
- Admin 安装/回滚页面截图，确认 5 分钟内回退及 Telemetry 事件写入。
- Grafana/SLA 仪表板截图，展示各指标在目标范围内。
- `scripts/perf/publish-hub-bench.sh` 执行结果与主要指标对照表附在交付文档。
- **mTLS 验证截图**：
  - 证书信息：`openssl x509 -in ~/.powerx/cli/client.crt -text -noout`
  - Dev API mTLS 握手日志
  - CLI 重试机制工作日志
- **回滚验证截图**：
  - 自动回滚触发日志（5 分钟计时器）
  - 手动回滚操作日志
  - Grafana 回滚延迟指标图表
  - 部署状态变更序列（installing → success/failed → rolling_back → rolled_back）
