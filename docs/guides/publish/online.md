# PowerX Publish Hub · 在线发布指南

本指南详细说明如何使用 `px-plugin publish` 将插件版本推送到 Publish Hub，通过 Marketplace 在线审核链路分发到租户。若需要跳过 Marketplace、直接在 PowerX Admin API 上执行本地安装，请结合《[本地安装指南](./local-install.md)》。

> 📌 在执行下述步骤前，请优先按《[能力注册与暴露指南](./capabilities.md)》完成 capability 定义与提交，确保 `.px-plugin/capabilities.json` 处于 approved 状态，否则 `px-plugin dist/publish` 会被自动阻断。

**适用对象**: 插件开发者 (plugin_developer)
**预计耗时**: 10 分钟（预检+入队）+ 4 小时（审核 SLA）
**相关 SLA**: 在线发布审核 ≤ 4 小时

---

## 快速命令一览（CLI 已支持 package/publish）

1. **本地构建 artefacts**
   ```bash
   px-plugin package --entry .
   ```
   产物写入 `<plugin>/.px-plugin/build/<timestamp>/package.tar.gz`、`metadata.json`、`payload/**`，可通过 `--skip-frontend/--skip-backend` 控制构建范围。
2. **配置 Publish API** – 任一方式均可：
   - 在 `~/.px-plugin/config.json` 增加：
     ```json
     {
       "publishApi": {
         "baseUrl": "http://127.0.0.1:8077/api/v1",
         "apiKey": "<registry-token>"
       }
     }
     ```
   - 或设置环境变量 `PX_PUBLISH_API_BASE`、`PX_PUBLISH_API_TOKEN`（CI/CD 友好）。
3. **上传到当前 PowerX Registry**
   ```bash
   px-plugin publish \
     --entry . \
     --tenant <tenant-uuid> \
     --channel beta \
     --notes "feat: 新审批流程"
   ```
   - `--tenant` 为必填，传租户 UUID（例如本地 dev tenant）。
   - 命令会读取最近一次 package 产物并调用 `POST {publishApi.baseUrl}/internal/plugins/releases`。成功输出 `publishId` 后，即可在 PowerX Marketplace/插件管理后台进入审核阶段。

若命令提示缺少配置、证书或 artefact，按提示修复即可；不需要再手动上传 tar 包或改写宿主 `plugins/installed`。

---

## 前置准备

### 1. 环境要求

| 组件 | 版本要求 | 验证命令 |
|------|----------|----------|
| Node.js | ≥ 18.0 | `node --version` |
| npm | ≥ 9.0 | `npm --version` |
| Go | ≥ 1.24 | `go version` |
| Playwright | ≥ 1.48 | `npx playwright --version` |

### 2. 依赖安装

```bash
# 根目录依赖
npm install

# CLI 依赖
cd tools/cli
npm install

# 返回根目录
cd ../..
```

### 3. 权限验证

确保您的账号具备 `plugin_developer` 角色权限：

- `publish:submit` - 提交发布
- `publish:view` - 查看发布状态
- `plugin:view` - 查看插件信息
- `system:view_logs` - 查看系统日志

**验证方法**:
```bash
# 检查 RBAC 权限
curl -H "X-Powerx-User-Id: $USER_ID" \
     -H "X-Powerx-Role: plugin_developer" \
     $PX_MARKETPLACE_API_URL/admin/metrics
```

### 4. mTLS 证书配置（重要！）

#### 4.1 生成 mTLS 证书

```bash
# 配置认证并生成证书到 ~/.powerx/cli
px auth configure
```

生成的文件：
- `~/.powerx/cli/client.crt` - 客户端证书（有效期 90 天）
- `~/.powerx/cli/client.key` - 客户端私钥
- `~/.powerx/cli/ca.crt` - CA 根证书（有效期 3 年）

#### 4.2 验证证书

```bash
# 检查证书有效期
openssl x509 -in ~/.powerx/cli/client.crt -text -noout | grep "Not After"

# 验证证书链
openssl verify -CAfile ~/.powerx/cli/ca.crt ~/.powerx/cli/client.crt
```

**重要**: 确保证书有效期 > 30 天，否则需要更新证书。

### 5. 环境变量配置

在 `~/.px-plugin/config.json` 或环境变量中配置：

```bash
# Dev / Publish API 配置
export PX_DEV_API_BASE="http://127.0.0.1:8077/api/v1"
export PX_DEV_API_TOKEN="dev-api-token"
export PX_PUBLISH_API_BASE="http://127.0.0.1:8077/api/v1"
export PX_PUBLISH_API_TOKEN="registry-token"

# 签名相关
export PX_MARKETPLACE_PUBLIC_KEY="$(cat /path/to/marketplace-public-key.pem)"
export PX_SIGNING_ENDPOINT="https://api.powerx.dev/signing"

# mTLS 证书路径
export PX_MTLS_CERT_PATH="~/.powerx/cli/client.crt"
export PX_MTLS_KEY_PATH="~/.powerx/cli/client.key"
export PX_MTLS_CA_PATH="~/.powerx/cli/ca.crt"
```

### 6. 配置文件检查

#### 6.1 manifest.json / plugin.yaml

确保 `manifest.json` 包含：
- `id` - 插件唯一标识符
- `version` - 语义化版本号（递增）
- `permissions` - 权限声明数组
- `channel` - 发布渠道（stable/beta）

#### 6.2 publish.yml

```yaml
channels:
  - name: stable
    rollout:
      batches:
        - percentage: 10
          startAt: "2025-11-07T00:00:00Z"
        - percentage: 100
          startAt: "2025-11-08T00:00:00Z"

tenantFilters:
  enabled: true
  tenants:
    - "tenant-001"
    - "tenant-002"

autoUpgrade: false
rollbackPlan:
  enabled: true
  previousVersion: "1.3.0"
```

#### 6.3 CHANGELOG.md 与 CLI 产物

Stable 渠道发布必须包含发布说明（新功能 / 修复 / Breaking Changes / 升级指南），建议同步写入 `CHANGELOG.md`。在运行 `px-plugin package` 后，可使用以下命令检查 artefact：

```bash
ls -R .px-plugin/build/latest
jq '.' .px-plugin/build/latest/metadata.json
```

---

## 执行流程

### 步骤 1: 预检查

在发布前执行完整的代码检查：

```bash
# 代码检查
npm run lint

# 单元测试
npm test

# Go 测试
go test ./...

# 类型检查
tsc --noEmit
```

**验证点**:
- ✅ 所有 lint 检查通过
- ✅ 测试覆盖率 ≥ 80%
- ✅ 无类型错误
- ✅ Go 测试全部通过

### 步骤 2: 创建发布计划（px-plugin publish create）

004-publish-hub-spec 启用后，所有在线发布都应通过计划（Plan）驱动，以便串联灰度、回滚和 SLA 监控。

```bash
px-plugin publish create \
  --manifest ./dist/manifest.json \
  --channel stable \
  --notes ./CHANGELOG.md \
  --rollout-strategy canary \
  --batches '[{"percentage":20,"wait":"10m"},{"percentage":80}]' \
  --window-start "2025-11-07T10:00:00Z" \
  --window-end "2025-11-07T12:00:00Z"
```

**参数说明**:
- `--manifest` - manifest 路径，Plan 会将其内联保存，供审批比对。
- `--channel` - stable/beta，写入 `plan.channel`。
- `--notes` - 一般映射到 `CHANGELOG.md`，Plan 审核页会显示。
- `--rollout-strategy` - `canary`（默认）或 `direct`。
- `--batches` - JSON 数组，定义每个批次的百分比与等待时间（ISO-8601 或 `10m`/`2h`）。
- `--window-*` - 控制审批窗口，便于 QA/发布经理锁定时间。
- `--auto-rollback/--dry-run` - 是否自动触发回滚、是否只演练流程。

CLI 会调用 `POST /internal/publish/create` 并返回 Plan 元数据，Plan 会实时同步到 Admin `/_p/<tenant>/publish/pipelines` 页面。

### 步骤 3: 部署计划并管理灰度（px-plugin publish deploy）

计划获批后，执行部署命令触发灰度批次与回滚令牌生成：

```bash
px-plugin publish deploy \
  --plan $PLAN_ID \
  --strategy canary \
  --batches '[{"percentage":20,"wait":"15m"},{"percentage":80}]' \
  --notes "stable rollout" \
  --commit $(git rev-parse HEAD)
```

**要点**:
- `--plan` 对应步骤 2 输出的 `planId`。
- `--strategy` 支持 `canary`、`blue-green`、`direct`；当批次数组为空时默认为 100%。
- `--batches` 可与 `publish create` 阶段配置不同，用于紧急调整。
- `--commit` 便于回溯构建来源；`--dry-run` 可用于演练。
- 响应包含 `deploymentId`、`rollbackToken`，后续灰度失败时 5 分钟内可回滚。

### 步骤 4: 兼容旧版流水线（px-plugin publish，可选）

如需与旧版 Marketplace 审核流程对齐或回归测试，可继续使用单步命令：

```bash
px-plugin publish \
  --manifest dist/manifest.json \
  --channel stable \
  --notes ./CHANGELOG.md \
  --receipt ./artifacts/publish-receipt.json
```

此命令仍会执行预检、签名、上传，并写入 `publish-receipt.json` 供审计，但不再承载灰度批次与 Plan 元数据。官方推荐优先使用步骤 2-3 的新链路。

### 步骤 5: 保存回执与计划记录

#### 5.1 发布计划响应

```json
{
  "planId": "plan-1730978931123",
  "publishId": "publish-1730978931567",
  "channel": "stable",
  "status": "draft",
  "window": {
    "start": "2025-11-07T10:00:00Z",
    "end": "2025-11-07T12:00:00Z"
  },
  "autoRollback": true,
  "dryRun": false
}
```

- `planId`/`publishId`：Plan 与 Marketplace 审核之间的桥梁，需写入变更记录。
- `window`：审批窗口，Admin 会根据该信息提示 SLA。

#### 5.2 部署响应

```json
{
  "deploymentId": "deploy-1730979000444",
  "planId": "plan-1730978931123",
  "state": "running",
  "batches": [
    { "percentage": 20, "status": "scheduled" },
    { "percentage": 80, "status": "scheduled" }
  ],
  "rollbackToken": "rollback-1730979000501"
}
```

- `deploymentId`：用于查询灰度状态、生成安装/回滚日志。
- `rollbackToken`：回滚 API/Admin 使用的凭证，必须安全保存。

#### 5.3 兼容模式回执（publish-receipt.json）

```json
{
  "publishId": "d6b3c1c0-5f07-4bfa-b083-6f8cc9a4b9de",
  "versionId": "demo-plugin-1.4.0",
  "reviewQueueId": "b8b47ad1-5d90-45e3-b3b3-a870e63f9d00",
  "uploadUrl": "https://upload.marketplace.powerx.local/plugins/demo-plugin/uploads/d6b3c1c0-5f07-4bfa-b083-6f8cc9a4b9de",
  "channel": "stable",
  "submittedAt": "2025-11-07T08:42:12.000Z",
  "sla": {
    "expectedReviewDuration": "4h",
    "maxDuration": "4h"
  }
}
```

仍可用来对接历史工单或自动化脚本；若使用 Plan，则以 `planId`/`deploymentId` 作为主键。

### 步骤 6: 监控发布状态

#### 6.1 追踪发布计划/灰度

- CLI：
  ```bash
  curl -H "X-Powerx-User-Id: $USER_ID" \
       -H "X-Powerx-Role: plugin_developer" \
       "$PX_MARKETPLACE_API_URL/internal/publish/plans/$PLAN_ID"
  ```
- Admin：访问 `/_p/<tenant>/publish/pipelines` 查看 Plan、批次进展与回滚按钮。
- 指标：`framework/backend/go/observability/publish_metrics.go` 会把 Plan → Deploy 耗时写入 `plugin_publish_pipeline_duration_ms` 与 `publish_local_iteration_cycle_time`。

#### 6.2 查看审核队列

访问 Marketplace 审核界面：
- URL: `https://admin.powerx.dev/marketplace/review`
- 确认 Plan 关联的 `publishId` 已入队，状态应显示为 `pending_review`

#### 6.3 监控 Telemetry

Plan/Deploy 会新增以下事件：
- `plugin.publish.plan.created` - Plan 创建成功
- `plugin.publish.plan.deployed` - 灰度启动
- `plugin.publish.deploy.rollback_queued` - 自动回滚触发
- `plugin.publish.submitted` / `plugin.publish.pipeline.*` - 兼容旧版流水线
- `plugin.publish.approved` / `plugin.publish.rejected` - 审核结果

---

## 内部执行流程（开发者参考）

### 1. 预检阶段 (Precheck)

**文件位置**: `tools/cli/src/lib/publish/precheck.ts`

执行检查：
- 验证 manifest 字段格式
- 检查语义化版本号递增
- 验证权限声明格式（必须是数组）
- 检查测试覆盖率
- 验证签名材料存在
- 确认 Stable 渠道包含发布说明

### 2. 发布计划编排 (Release Pipeline Orchestrator)

**文件位置**: `framework/backend/go/runtime/publish/pipeline_handler.go`

- `POST /internal/publish/create`：持久化 `planId/publishId`、rollout 配置与审批窗口，CLI `publish create` 直接调用。
- `POST /internal/publish/deploy`：根据计划生成 `deploymentId`、批次状态与 `rollbackToken`，并把耗时写入 `plugin_publish_pipeline_duration_ms`/`publish_local_iteration_cycle_time`。
- `planStore`（内存实现）负责追踪 Plan/Deploy 状态，Admin Pipelines 视图通过它拉取数据。
- 失败时返回结构化错误（`PLAN_NOT_FOUND`、`INVALID_*`），CLI 将错误传播到终端，方便开发者修复。

### 3. 旧版流水线阶段 (Pipeline)

**文件位置**: `tools/cli/src/lib/publish/pipeline.ts`

执行流程：
- 生成 `publishId` 和 `versionId`
- 生成 `reviewQueueId`
- 拼接 changelog
- 生成上传地址
- 创建发布请求记录

### 4. 签名阶段

使用 Marketplace 公钥对 artefact 进行签名：
- RSA-PSS 签名算法
- SHA-256 哈希
- 签名存储在 `manifest.signature`

### 5. 上传阶段

- 上传 artefact 到对象存储
- 生成临时访问 URL
- 记录上传状态

### 6. 事件广播

**文件位置**: `tools/cli/src/lib/telemetry/emitter.ts`

广播 `plugin.publish.submitted` 事件，包含：
- `publishId`
- `pluginId`
- `version`
- `channel`
- `submitter`
- `submittedAt`

---

## SLA 监控

### 目标指标

| 指标 | 目标值 | 监控方式 |
|------|--------|----------|
| 发布流水线执行时间 | ≤ 10 分钟 | `plugin_publish_pipeline_duration_ms` |
| 在线审核时间 | ≤ 4 小时 | `plugin_publish_review_duration_ms` |
| 本地迭代周期 | ≤ 15 分钟 | `publish_local_iteration_cycle_time` |
| 租户通知时间 | ≤ 30 分钟 | `plugin_notification_delay_seconds` |
| 灰度失败率 | < 5% | `publish_gray_error_rate` |

### 查看 SLA 仪表板

**Grafana URL**: https://grafana.powerx.dev/d/publish-hub-sla

查看面板：
1. 在线审核时延（95th percentile）
2. 发布流水线耗时趋势
3. 发布成功率统计
4. 告警状态总览

### 告警处理

如果 SLA 超时，会触发以下告警：

**告警名称**: `PublishOnlineSLAExceeded`
- **触发条件**: `plugin_publish_pipeline_duration_ms > 4h for 15m`
- **级别**: Warning
- **处理**: 参见 `docs/operations/publish-hub-sla.md` 第 17-123 行

**通知渠道**:
- Slack: `#powerx-publish-hub`
- PagerDuty: On-call 工程师
- Email: `ops@powerx.dev`

---

## 故障排除

### 常见错误与解决方案

#### 错误 1: `manifest permissions must be an array`

**症状**:
```
ERROR: manifest permissions must be an array
```

**原因**: `plugin.yaml` 或 `manifest.json` 中 `permissions` 字段格式错误

**解决方案**:
```yaml
# 错误格式
permissions: "read:users"

# 正确格式
permissions:
  - "read:users"
  - "write:data"
```

#### 错误 2: `stable releases must include release notes`

**症状**:
```
ERROR: stable releases must include release notes
```

**原因**: Stable 渠道发布未提供 `--notes` 参数

**解决方案**:
```bash
# 提供发布说明
px-plugin publish \
  --channel stable \
  --notes ./CHANGELOG.md
```

#### 错误 3: Changelog 读取失败

**症状**:
```
ERROR: Failed to read changelog file
```

**原因**: `--notes` 路径不存在或权限不足

**解决方案**:
```bash
# 确认文件存在
ls -l ./CHANGELOG.md

# 确认有读取权限
cat ./CHANGELOG.md
```

#### 错误 4: 版本号未递增

**症状**:
```
ERROR: version must be greater than previous version
```

**原因**: 当前版本号 ≤ 已发布版本号

**解决方案**:
```bash
# 查看已发布版本
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/plugins/$PLUGIN_ID/versions

# 更新版本号
# manifest.json
{
  "version": "1.4.1"  // 确保 > 1.4.0
}
```

#### 错误 5: mTLS 握手失败

**症状**:
```
ERROR: mTLS handshake failed
```

**原因**:
- 证书过期
- 证书链验证失败
- 证书与私钥不匹配

**解决方案**:
```bash
# 1. 检查证书有效期
openssl x509 -in ~/.powerx/cli/client.crt -noout -dates

# 2. 重新配置证书
px auth configure

# 3. 验证证书链
openssl verify -CAfile ~/.powerx/cli/ca.crt ~/.powerx/cli/client.crt
```

#### 错误 6: RBAC 权限不足

**症状**:
```
ERROR: insufficient permissions: missing publish:submit
```

**原因**: 用户角色或权限配置错误

**解决方案**:
1. 确认用户角色为 `plugin_developer`
2. 确认具备 `publish:submit` 权限
3. 联系管理员更新权限

```bash
# 验证权限
curl -H "X-Powerx-User-Id: $USER_ID" \
     -H "X-Powerx-Role: plugin_developer" \
     $PX_MARKETPLACE_API_URL/admin/metrics
```

#### 错误 7: 自动化扫描失败

**症状**:
```
ERROR: automated security scan failed: high_severity_issues
```

**原因**: 安全扫描发现高危问题

**解决方案**:
1. 查看扫描报告
2. 修复安全问题
3. 重新提交发布

```bash
# 查看扫描报告
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/$PUBLISH_ID/scan-report
```

#### 错误 8: 上传超时

**症状**:
```
ERROR: upload timeout after 300s
```

**原因**:
- Artefact 体积过大（默认 <300MB）
- 网络连接问题
- 对象存储服务异常

**解决方案**:
```bash
# 1. 检查 artefact 大小
du -h dist/bundle.zip

# 2. 压缩 artefact
px-plugin dist --target online --compress

# 3. 分片上传（如果支持）
# 参见 tools/cli/src/lib/upload/chunked_upload.ts
```

#### 错误 9: 重复提交

**症状**:
```
ERROR: duplicate publish request for version
```

**原因**: 相同版本号已提交

**解决方案**:
```bash
# 查看已提交版本
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/history/$PLUGIN_ID

# 递增版本号后重试
```

#### 错误 10: Telemetry 发送失败

**症状**:
```
WARN: failed to emit telemetry event
```

**影响**: 不影响发布，但影响监控

**解决方案**:
- 检查网络连接
- 验证 Telemetry 端点
- 查看详细日志

---

## 后续步骤

### 1. 等待审核结果

- **SLA**: 4 小时内完成审核
- **通知**: 审核完成后收到邮件/通知
- **状态查询**: 通过 `publishId` 查询状态

### 2. 审核通过

审核通过后：
1. 系统广播 `plugin.publish.approved` 事件
2. 订阅租户在 30 分钟内收到通知
3. 租户管理员可在 Admin 中查看新版本

### 3. 审核拒绝

如审核被拒绝：
1. 查看拒绝原因和扫描报告
2. 修复问题
3. 重新提交发布

### 4. 灰度发布

如配置了灰度策略：
1. 初始 10% 租户自动升级
2. 监控系统指标
3. 24-48 小时后推广到 100%

### 5. 后续操作

- **查看安装统计**: `https://admin.powerx.dev/marketplace/installations`
- **监控告警**: `https://grafana.powerx.dev/d/publish-hub-sla`
- **查看文档**: `docs/guides/publish/marketplace-review.md`
- **故障排除**: `docs/operations/publish-hub-sla.md`

---

## 最佳实践

### 1. 发布前准备

- [ ] 代码已完成 lint、test、type check
- [ ] 版本号已递增
- [ ] CHANGELOG 已更新
- [ ] 权限声明正确
- [ ] 测试覆盖率 ≥ 80%
- [ ] 安全扫描通过
- [ ] mTLS 证书有效（> 30 天）

### 2. 发布过程

- [ ] 使用 `--notes` 提供详细发布说明
- [ ] 监控 `publish-receipt.json` 生成
- [ ] 记录 `publishId` 供后续查询
- [ ] 关注审核队列状态

### 3. 发布后跟踪

- [ ] 监控 SLA 指标
- [ ] 关注租户反馈
- [ ] 准备回滚方案
- [ ] 收集使用统计

### 4. 安全注意事项

- 定期轮换 mTLS 证书（建议 30 天前预警）
- 不要在公开仓库提交 `publish-receipt.json`
- 保持签名私钥安全
- 监控异常发布活动

### 5. 性能优化

- Artefact 体积 < 300MB
- 启用增量构建
- 使用 CDN 加速上传
- 合理配置重试机制

---

## 相关资源

### 文档
- [Quickstart 指南](../../004-publish-hub-spec/quickstart.md)
- [Marketplace 审核指南](./marketplace-review.md)
- [离线发布指南](./offline.md)
- [SLA 运行手册](../../operations/publish-hub-sla.md)
- [安全文档](../../security/publish-hub.md)
- [RBAC 指南](../../operations/publish-hub-rbac.md)

### 代码位置
- CLI 发布命令: `tools/cli/src/commands/publish.ts`
- 预检逻辑: `tools/cli/src/lib/publish/precheck.ts`
- 流水线逻辑: `tools/cli/src/lib/publish/pipeline.ts`
- 签名验证: `tools/cli/src/lib/security/keyEnvelope.ts`
- 事件广播: `tools/cli/src/lib/telemetry/emitter.ts`

### 监控
- Grafana 仪表板: https://grafana.powerx.dev/d/publish-hub-sla
- Prometheus: https://prometheus.powerx.dev
- 告警管理: https://alertmanager.powerx.dev

### 支持
- Slack: `#powerx-publish-hub`
- 邮箱: `ops@powerx.dev`
- 文档反馈: https://github.com/ArtisanCloud/PowerXPlugin/issues
