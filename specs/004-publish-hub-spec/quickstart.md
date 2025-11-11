# Quickstart — Publish Hub Feature

## 0. 脚手架与合规准备
1. 在任意工作目录执行 `px-plugin init <plugin-id> --template fullstack-go-nuxt --org <team>`，生成标准骨架并写入 `publish.yml`、`reports/sbom.json`。
2. 运行 `px-plugin doctor --fix`，确保 Node/Go 版本、Feature Flag 以及 `backend/go.mod`、`web-admin/node_modules` 状态满足要求，查看 `.doctor/report.json`。
3. 如果需要导入第三方源码（客户模板、历史仓库），执行 `px-plugin import --source ./vendor.tar.gz --license MIT`，CLI 会依据 `config/compliance/external_source_policy.yaml` 生成 `./.compliance/import-report.json`，并将信息发送到 `plugin-import-audit` Webhook。

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

## 2. Dev 热加载链路（TypeScript CLI）

1. 在插件仓执行 `px-plugin dev --watch --tenant demo-tenant --entry ./dist`（实现参考 `tools/cli/src/commands/dev/watch.ts`）。
2. CLI 将读取 `dist/manifest.json` 并向 Dev API (`tools/cli/src/runtime/hotreload/session.ts`) 发送 register 请求，输出 `sessionId`, `reloadToken`, Admin 调试地址。
   - **mTLS 验证**：CLI 会自动加载 mTLS 证书并建立双向 TLS 连接
   - 如果 mTLS 握手失败，会自动重试 3 次（指数退避：1s, 2s, 4s）
3. 修改代码并保存，CLI watcher 会在 250ms 去抖后聚合 diff，调用 Dev API reload；确认构建耗时 ≤2s，Admin SSE 获取 `reload` 日志。
4. 结束调试：运行 `px-plugin dev --stop`（或 `CTRL+C`），Dev API 删除 session 并由 `framework/backend/go/runtime/devapi/handlers/dev_plugins.go` 记录审计，日志保留 7 天。

## 2.1 Dev 热加载链路（Go CLI - 新实现）

1. **构建 Go CLI**（首次使用）：
   ```bash
   cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tools/cli
   go build -o px-plugin ./cmd/px-plugin
   ```
   或使用 `make build-cli`（如提供）。

2. **配置 mTLS 证书**：
   ```bash
   # 证书位置：~/.px-plugin/certs/
   mkdir -p ~/.px-plugin/certs
   # 放置 client.crt, client.key, ca.crt
   ```

3. **运行 dev watch 模式**：
   ```bash
   # 基本用法
   ./px-plugin dev --watch --entry ./my-plugin

   # 完整参数
   ./px-plugin dev --watch \
     --entry ./my-plugin \
     --tenant demo-tenant \
     --ignore "**/*.log" \
     --dev-api https://dev-api.powerx.local
   ```

4. **核心流程**：
   - CLI 解析 `--entry` 参数并加载 `plugin.yaml` manifest
   - 向 Dev API 发送 `POST /internal/dev/plugins/register` 建立会话，返回 `sessionId` + `reloadToken`
   - 启动 `fsnotify` 文件监听器，递归监听插件目录（忽略 `.git`, `node_modules`, `dist/**`）
   - 文件变更时生成 SHA256 哈希，聚合到 250ms 去抖窗口
   - 调用 `POST /internal/dev/plugins/reload`（带幂等 `x-reload-id`），成功后在 stdout 输出日志
   - 结束调试时调用 `DELETE /internal/dev/plugins/register/{sessionId}` 清理会话

5. **会话管理**：
   ```bash
   # 查看活跃会话
   ./px-plugin dev list-sessions

   # 恢复会话
   ./px-plugin dev resume sess-123

   # 停止会话
   ./px-plugin dev stop sess-123

   # 查看会话日志
   ./px-plugin dev logs sess-123
   ```

6. **会话持久化位置**：
   - 会话数据：`~/.px-plugin/sessions/{sessionId}.json`
   - 审计日志：`~/.px-plugin/logs/audit.log`
   - 配置文件：`~/.px-plugin/config.json`

7. **性能指标**（Go CLI 优势）：
   - 冷启动（register）：≤ 1s
   - 增量 reload：P95 ≤ 2s
   - 文件变更到 API 调用：≤ 250ms
   - CLI 内存占用：≤ 100MB
   - 空闲时 CPU：≤ 1%，监听时 CPU：≤ 10%

8. **错误处理**：
   - 网络错误：自动重试（指数退避：1s, 2s, 4s, 8s, 30s）
   - 构建失败：回滚到上一个版本，显示错误日志
   - API 错误：检查 Dev API 健康状态，建议运行 `./px-plugin doctor`
   - 认证错误：引导用户运行 `./px-plugin auth configure`

9. **mTLS 验证**：
   ```bash
   # 检查证书
   openssl x509 -in ~/.px-plugin/certs/client.crt -text -noout

   # CLI 会自动加载并验证证书
   # 日志输出："mTLS handshake succeeded"
   ```

10. **对比 TypeScript 版**：
    - ✅ 行为 100% 一致
    - ✅ 性能更优（Go 并发优势）
    - ✅ 内存占用更低
    - ✅ 冷启动更快
    - ✅ CPU 使用率更低

## 3. 在线发布链路
1. 运行 `npm run lint && npm test && go test ./...`，确保预检输入准备就绪。
2. 使用新的 create 命令编排窗口：
   ```bash
   px-plugin publish create \
     --manifest ./dist/manifest.json \
     --channel stable \
     --notes ./CHANGELOG.md
   ```
   记录返回的 `planId` 与 `publishId`，并在 Admin `/publish/pipelines` 页面查看。
3. 部署至测试/灰度租户：
   ```bash
   px-plugin publish deploy \
     --plan <planId> \
     --strategy canary \
     --batches '[{"percentage":20,"wait":"10m"},{"percentage":80}]'
   ```
   CLI 将输出 `deploymentId` 与 `rollbackToken`，方便后续回退。
4. 可选：运行 `px-plugin publish --channel stable --notes ./CHANGELOG.md` 触发旧版流水线，确保兼容。
5. 检查 `publish-receipt.json`（含 `publishId`, `versionId`, `reviewQueueId`），进入 Marketplace 审核队列查看状态；若失败，根据 CLI 提示修复。
6. 审核通过后确认租户在 30 分钟内收到通知。
7. **SLA 监控**：在线发布审核应在 4 小时内完成，可在 Grafana 中查看 `plugin_publish_pipeline_duration_ms` 与 `publish_local_iteration_cycle_time` 指标。

## 4. 离线发布链路
1. 运行 `px-plugin pack --manifest ./dist/manifest.json --artefact ./dist --channel offline --notes "隔离租户发版" --sign ./cert.pem`，命令会复用 `.pxp` 打包逻辑并输出 `release.manifest.json`。
2. 执行 `px-plugin import --offline --pkg dist/<plugin>.pxp --integrity dist/integrity.txt --signature dist/manifest.signature --whitelist tenant-a,tenant-b`，CLI 会调用 `POST /internal/marketplace/offline/import` 并返回审核队列 ID。
3. 运维在 Marketplace 离线入口确认队列信息，并勾选目标租户白名单。
4. 审核通过后，租户管理员使用 `install/local` 导入；失败时 5 分钟内执行回滚。
5. **离线签名验证**：系统会验证 RSA-PSS 签名和密钥封装，确保包完整性。
6. **SLA 监控**：离线审核应在 1 个工作日内完成，可在 Grafana 中查看 `plugin_offline_approval_duration_minutes` 与 `marketplace_listing_sla_hours` 指标。

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
2. **Go CLI 性能测试**：
   ```bash
   # 构建 Go CLI
   go build -o px-plugin ./tools/cli/cmd/px-plugin

   # 运行性能测试
   ./px-plugin dev --watch --entry ./my-plugin &
   # 修改文件多次，观察：
   # - reload 耗时（应 ≤ 2s）
   # - 内存占用（应 ≤ 100MB）
   # - CPU 使用（空闲时 ≤ 1%，监听时 ≤ 10%）
   ```

3. 通过 Grafana / `scripts/qa/workflow-metrics.mjs` 检查以下指标：
   - `dev.hotload.cli_reload_duration_ms` (目标：TypeScript ≤2s, Go ≤2s)
   - `dev.hotload.go_cli_register_ms` (Go CLI 冷启动，目标：≤1s)
   - `dev.hotload.go_cli_memory_bytes` (Go CLI 内存，目标：≤100MB)
   - `plugin_publish_pipeline_duration_ms` (目标：95th percentile ≤4h)
   - `plugin_offline_approval_duration_minutes` (目标：95th percentile ≤1d)
   - `plugin_install_rollback_latency_seconds` (目标：≤300s)
   - `publish_local_iteration_cycle_time` (目标：≤15m)
   - `publish_gray_error_rate` (目标：<5%)
   - `marketplace_listing_sla_hours` (目标：≤72h)
   - `plugin_deployments_total` (成功/失败率统计)
4. **mTLS 验证**：
   - 查看 Dev API 日志，确认 mTLS 握手成功
   - 检查证书验证日志：`client certificate verification succeeded`
   - 监控证书过期提醒
5. **Go CLI vs TypeScript CLI 对比**：
   - 性能基准测试
   - 行为一致性验证
   - API 契约对齐检查
6. Playwright 脚本验证 Admin 安装/回滚 UI。

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
- **Go CLI 验证**：
  - [ ] 构建 Go CLI：`go build -o px-plugin ./tools/cli/cmd/px-plugin`
  - [ ] 运行 `./px-plugin dev --watch` 并修改文件
  - [ ] 验证 reload 耗时 ≤ 2s
  - [ ] 检查会话持久化：`~/.px-plugin/sessions/`
  - [ ] 检查审计日志：`~/.px-plugin/logs/audit.log`
  - [ ] 对比 TypeScript 版行为一致性
- Marketplace 在线/离线审核界面（`review.vue` / `offline-review.vue`）截图 + 审核结果记录。
- Admin 安装/回滚页面截图，确认 5 分钟内回退及 Telemetry 事件写入。
- Grafana/SLA 仪表板截图，展示各指标在目标范围内。
- `scripts/perf/publish-hub-bench.sh` 执行结果与主要指标对照表附在交付文档。
- **mTLS 验证截图**：
  - 证书信息：`openssl x509 -in ~/.px-plugin/certs/client.crt -text -noout`
  - Dev API mTLS 握手日志
  - CLI 重试机制工作日志
- **回滚验证截图**：
  - 自动回滚触发日志（5 分钟计时器）
  - 手动回滚操作日志
  - Grafana 回滚延迟指标图表
  - 部署状态变更序列（installing → success/failed → rolling_back → rolled_back）
