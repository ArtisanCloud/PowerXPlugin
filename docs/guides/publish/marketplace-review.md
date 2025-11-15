# PowerX Publish Hub · Marketplace 审核指南

本指南详细说明 Marketplace 审核员如何处理在线发布和离线上传审核，包括审核流程、SLA 指标、告警处理和操作规范。

**适用对象**: Marketplace 审核员 (marketplace_reviewer) + 运维团队 (platform_ops)
**预计耗时**: 5-10 分钟（每个包审核）+ 4 小时（在线 SLA）/ 1 个工作日（离线 SLA）
**相关 SLA**: 在线审核 ≤ 4 小时，离线审核 ≤ 1 个工作日

---

## 审核员职责

### 核心职责

1. **安全审核**: 验证包完整性、签名有效性和安全扫描结果
2. **合规检查**: 确保插件符合平台政策和权限规范
3. **质量控制**: 评估插件质量、用户体验和兼容性
4. **SLA 管理**: 在规定时间内完成审核，避免超时告警
5. **事件记录**: 记录审核过程、决策和反馈

### 权限要求

审核员需要以下权限：
- `marketplace:review` - 审核 Marketplace 提交
- `marketplace:offline` - 处理离线包
- `publish:approve` - 批准发布
- `publish:reject` - 拒绝发布
- `publish:view` - 查看发布状态
- `system:view_logs` - 查看系统日志

---

## 访问审核系统

### 1. 登录 Admin 控制台

访问 Marketplace 审核界面：
- **在线审核**: `https://admin.powerx.dev/marketplace/review`
- **离线审核**: `https://admin.powerx.dev/marketplace/offline-review`

### 2. 验证权限

```bash
# 检查当前用户权限
curl -H "X-Powerx-User-Id: $USER_ID" \
     -H "X-Powerx-Role: marketplace_reviewer" \
     $PX_MARKETPLACE_API_URL/admin/metrics
```

### 3. 界面布局

审核界面包含以下区域：
- **队列列表**: 显示待审核的发布/包
- **详情面板**: 显示发布/包详细信息
- **操作区**: 批准、拒绝、要求修改按钮
- **SLA 计时器**: 显示剩余审核时间
- **日志区**: 显示审核过程日志

---

## 在线发布审核流程

### 步骤 1: 查看审核队列

在在线审核页面中，查看待审核的发布：

| 字段 | 说明 |
|------|------|
| `publishId` | 发布请求唯一标识 |
| `pluginId` | 插件标识 |
| `version` | 版本号 |
| `channel` | 发布渠道（stable/beta） |
| `submitter` | 提交者 |
| `submittedAt` | 提交时间 |
| `slaDeadline` | SLA 截止时间（提交时间 + 4 小时） |
| `scanStatus` | 自动化扫描状态 |
| `riskLevel` | 风险等级（低/中/高） |

> **Plan/Deploy 上下文（px-plugin publish create/deploy）**  
> 004-publish-hub-spec 引入的 Plan 机制会在 CLI 阶段生成 `planId`、`deploymentId` 与 `rollbackToken`。审核员在 Admin `/_p/<tenant>/publish/pipelines` 页面可查看：  
> - `planId`：对应 CLI `publish create` 输出，需与审核记录关联。  
> - `deploymentId` + 批次：`publish deploy` 触发，可用于确认灰度与回滚策略。  
> - `rollbackToken`：若审核后触发回滚，请确保记录该 token 并在审批备注中引用。  
> 若发现队列中的版本缺少 Plan 信息，应退回开发者补充，避免绕过灰度/回滚守护。

**状态说明**:
- `pending_review` - 等待审核
- `in_review` - 正在审核
- `approved` - 审核通过
- `rejected` - 审核拒绝
- `needs_changes` - 需要修改

### 步骤 2: 查看发布详情

点击 `publishId` 查看详细信息：

#### 2.1 基础信息

```json
{
  "publishId": "d6b3c1c0-5f07-4bfa-b083-6f8cc9a4b9de",
  "pluginId": "demo-plugin",
  "version": "1.4.0",
  "channel": "stable",
  "submitter": "dev1",
  "submittedAt": "2025-11-07T08:42:12.000Z",
  "slaDeadline": "2025-11-07T12:42:12.000Z",
  "notes": "详见 CHANGELOG.md"
}
```

#### 2.2 自动化扫描结果

**文件位置**: `framework/backend/go/runtime/marketplace/services/scanner.go`

扫描项目：
- [ ] **安全扫描**: 恶意代码、漏洞检测
- [ ] **合规检查**: 权限使用、数据访问
- [ ] **依赖分析**: 第三方库安全
- [ ] **性能评估**: 资源使用、启动时间
- [ ] **兼容性测试**: 多租户环境

扫描结果示例：
```json
{
  "scanId": "scan-123",
  "startedAt": "2025-11-07T08:42:15.000Z",
  "completedAt": "2025-11-07T08:43:20.000Z",
  "status": "completed",
  "issues": [
    {
      "severity": "low",
      "type": "dependency_vulnerability",
      "description": "发现已知漏洞依赖",
      "component": "lodash@4.17.20",
      "recommendation": "升级到 lodash@4.17.21"
    }
  ],
  "score": 85,
  "passed": true
}
```

#### 2.3 发布说明

查看 `CHANGELOG.md` 或发布说明：
- 新功能列表
- 修复问题列表
- Breaking Changes
- 升级指南
- 已知问题

### 步骤 3: 执行审核检查

#### 3.1 完整性检查

- [ ] `manifest.json` 格式正确
- [ ] 版本号符合语义化规范
- [ ] 权限声明合理且必要
- [ ] 依赖关系完整
- [ ] 包大小 < 300MB

#### 3.2 安全检查

- [ ] 自动化扫描通过（无高危问题）
- [ ] 无已知安全漏洞
- [ ] 无恶意代码
- [ ] 权限使用合理
- [ ] 无敏感信息泄露

#### 3.3 合规检查

- [ ] 符合平台政策
- [ ] 无违规内容
- [ ] 数据使用合规
- [ ] 隐私政策符合要求
- [ ] 第三方服务授权完整

#### 3.4 质量检查

- [ ] 代码质量可接受
- [ ] 文档完整
- [ ] 错误处理完善
- [ ] 性能表现良好
- [ ] 用户体验友好

#### 3.5 兼容性检查

- [ ] 多租户环境兼容
- [ ] 资源隔离正确
- [ ] 无命名冲突
- [ ] API 版本兼容
- [ ] 升级/回滚方案完整

### 步骤 4: 查看租户影响

#### 4.1 灰度发布计划

如果配置了灰度发布，查看批次计划：
```yaml
rollout:
  batches:
    - percentage: 10
      startAt: "2025-11-07T00:00:00Z"
      tenants:
        - "tenant-001"
        - "tenant-002"
    - percentage: 100
      startAt: "2025-11-08T00:00:00Z"
```

#### 4.2 回滚方案

检查回滚计划：
- [ ] 上一版本已备份
- [ ] 回滚脚本可执行
- [ ] 数据迁移方案完整
- [ ] 用户通知已准备

### 步骤 5: 执行审核操作

#### 5.1 批准发布

```bash
# API 方式
curl -X POST "https://api.powerx.dev/publish/$PUBLISH_ID/approve" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "审核通过",
    "notes": "所有检查项通过，建议灰度发布",
    "autoRollout": true
  }'
```

**批准后系统自动执行**:
1. 更新发布状态为 `approved`
2. 广播 `plugin.publish.approved` 事件
3. 通知订阅租户
4. 触发灰度发布（如配置）
5. 记录审核日志

#### 5.2 拒绝发布

```bash
# API 方式
curl -X POST "https://api.powerx.dev/publish/$PUBLISH_ID/reject" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "安全扫描发现高危问题",
    "details": "检测到已知漏洞依赖 lodash@4.17.20，建议升级到 4.17.21",
    "requiredChanges": [
      "升级 lodash 依赖",
      "重新运行安全扫描"
    ]
  }'
```

**拒绝后系统自动执行**:
1. 更新发布状态为 `rejected`
2. 广播 `plugin.publish.rejected` 事件
3. 通知提交者
4. 提供修改建议
5. 记录审核日志

#### 5.3 要求修改

```bash
# API 方式
curl -X POST "https://api.powerx.dev/publish/$PUBLISH_ID/request-changes" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "需要补充文档",
    "details": "缺少升级指南和已知问题说明",
    "requiredChanges": [
      "添加升级指南章节",
      "添加已知问题说明",
      "完善错误处理文档"
    ]
  }'
```

**要求修改后**:
1. 发布状态变为 `needs_changes`
2. 开发者需修改后重新提交
3. SLA 计时器重置

### 步骤 6: 监控发布结果

#### 6.1 查看租户通知

```bash
# 查看通知状态
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/$PUBLISH_ID/notifications
```

#### 6.2 监控安装状态

```bash
# 查看安装统计
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/$PUBLISH_ID/installations
```

#### 6.3 检查回滚准备

- 确认上一版本快照已创建
- 验证回滚脚本可执行
- 检查数据备份完整性

---

## 离线包审核流程

### 步骤 1: 查看离线审核队列

在离线审核页面中，查看待审核的包：

| 字段 | 说明 |
|------|------|
| `reviewQueueId` | 审核队列 ID |
| `packageId` | 离线包 ID |
| `pluginId` | 插件标识 |
| `version` | 版本号 |
| `submitter` | 提交者 |
| `submittedAt` | 提交时间 |
| `slaDeadline` | SLA 截止时间（提交时间 + 1 个工作日） |
| `validationStatus` | 验证状态 |
| `whitelistCount` | 白名单租户数量 |

### 步骤 2: 验证离线包

#### 2.1 检查上传文件

必需文件：
- [ ] `demo-plugin-1.4.0.pxp` - 离线包（加密）
- [ ] `integrity.txt` - 完整性校验列表
- [ ] `manifest.signature` - manifest 签名
- [ ] `report.json` - 打包报告
- [ ] `audit.log` - 审计日志（可选）

#### 2.2 验证 .pxp 包

**文件位置**: `framework/backend/go/runtime/marketplace/services/offline_validator.go`

验证流程：
1. **解封对称密钥** - 使用 Marketplace 私钥解封
2. **解密包内容** - 使用对称密钥解密 `.pxp`
3. **验证签名** - 验证 RSA-PSS 签名
4. **检查完整性** - 验证 `integrity.txt` 中每个文件的哈希
5. **验证证书** - 验证证书链和有效期

#### 2.3 查看验证报告

```json
{
  "packageId": "demo-plugin-1.4.0.pxp",
  "validationStatus": "passed",
  "startedAt": "2025-11-07T08:42:20.000Z",
  "completedAt": "2025-11-07T08:42:45.000Z",
  "checks": {
    "keyUnwrap": {
      "status": "passed",
      "duration_ms": 120
    },
    "packageDecrypt": {
      "status": "passed",
      "duration_ms": 350
    },
    "signatureVerify": {
      "status": "passed",
      "algorithm": "rsa-pss-sha256",
      "cert_fingerprint": "SHA256:..."
    },
    "integrityCheck": {
      "status": "passed",
      "total_files": 142,
      "verified_files": 142,
      "failed_files": 0
    },
    "certValidation": {
      "status": "passed",
      "chain_valid": true,
      "not_expired": true
    }
  },
  "issues": []
}
```

### 步骤 3: 审核包内容

#### 3.1 完整性检查

- [ ] `integrity.txt` 存在且格式正确
- [ ] 所有文件哈希验证通过
- [ ] 无缺失或多余文件
- [ ] 文件大小符合预期
- [ ] 压缩格式正确

#### 3.2 签名验证

- [ ] `manifest.signature` 格式正确
- [ ] RSA-PSS 签名有效
- [ ] 证书链验证通过
- [ ] 证书未过期
- [ ] 证书指纹匹配

#### 3.3 包内容检查

- [ ] `manifest.json` 格式正确
- [ ] 版本号符合语义化规范
- [ ] 权限声明合理
- [ ] 无敏感信息泄露
- [ ] 依赖关系完整

#### 3.4 安全检查

- [ ] 无已知安全漏洞
- [ ] 无恶意代码
- [ ] 加密实现正确
- [ ] 密钥封装安全
- [ ] 审计日志完整

### 步骤 4: 检查租户白名单

#### 4.1 验证租户列表

在 "目标租户" 字段中查看：
- 租户 ID 列表
- 租户存在性
- 租户状态
- 授权有效性

```bash
# 验证租户存在
for tenant in $TENANT_LIST; do
  curl -H "X-Powerx-User-Id: $USER_ID" \
       $PX_MARKETPLACE_API_URL/tenants/$tenant/status
done
```

#### 4.2 检查授权

- [ ] 所有目标租户已正确配置
- [ ] 租户存在且状态正常
- [ ] 租户有权限使用此插件
- [ ] 租户数量符合预期

### 步骤 5: 执行审核操作

#### 5.1 批准离线包

```bash
# API 方式
curl -X POST "https://api.powerx.dev/marketplace/offline/$REVIEW_QUEUE_ID/approve" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "审核通过",
    "notes": "所有检查项通过，签名验证成功",
    "whitelist": [
      "offline-tenant-001",
      "offline-tenant-002"
    ]
  }'
```

**批准后系统自动执行**:
1. 更新审核状态为 `approved`
2. 广播 `plugin.offline.approved` 事件
3. 通知白名单租户管理员
4. 租户可在 Admin 中看到新版本
5. 记录审核日志

#### 5.2 拒绝离线包

```bash
# API 方式
curl -X POST "https://api.powerx.dev/marketplace/offline/$REVIEW_QUEUE_ID/reject" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "签名验证失败",
    "details": "manifest.signature 与 manifest.json 不匹配",
    "validation_report": "详见附件验证报告"
  }'
```

**拒绝后系统自动执行**:
1. 更新审核状态为 `rejected`
2. 广播 `plugin.offline.rejected` 事件
3. 通知提交者
4. 提供验证报告和修改建议
5. 记录审核日志

#### 5.3 请求重新上传

```bash
# API 方式
curl -X POST "https://api.powerx.dev/marketplace/offline/$REVIEW_QUEUE_ID/request-reupload" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer" \
  -d '{
    "reason": "完整性验证失败",
    "details": "integrity.txt 中部分文件哈希不匹配",
    "requiredActions": [
      "重新生成 integrity.txt",
      "验证源文件完整性",
      "重新打包"
    ]
  }'
```

---

## SLA 监控与告警

### 目标指标

| 指标 | 目标值 | 监控方式 |
|------|--------|----------|
| 在线审核时间 | ≤ 4 小时 | `plugin_publish_review_duration_ms` |
| 离线审核时间 | ≤ 1 个工作日（1440 分钟） | `plugin_offline_approval_duration_minutes` |
| 审核通过率 | ≥ 90% | `plugin_review_approval_rate` |
| 首次审核通过率 | ≥ 70% | `plugin_review_first_time_approval_rate` |

### 查看 SLA 仪表板

**Grafana URL**: https://grafana.powerx.dev/d/publish-hub-sla

**面板内容**:
1. 在线审核时延（95th percentile）
2. 离线审核时延（95th percentile）
3. 审核通过率趋势
4. 审核队列长度
5. 告警状态总览
6. 审核员工作量统计
7. SLA 违规统计

### 告警处理

#### 告警 1: PublishOnlineSLAExceeded

**触发条件**:
```
plugin_publish_review_duration_ms > 4h for 15m
```

**级别**: Warning

**处理流程**:
1. 立即响应（0-5 分钟）
   - 登录 Grafana 查看告警详情
   - 定位超时的发布
   - 检查过去 1 小时的发布数量趋势

2. 调查分析（5-15 分钟）
   - 检查审核队列积压情况
   - 查看审核员工作负载
   - 确认自动化扫描是否卡住

3. 执行修复（15-30 分钟）
   - 分配更多审核员处理
   - 重启卡住的扫描服务
   - 清理积压队列

4. 后续跟进（30-60 分钟）
   - 监控告警状态
   - 通知受影响开发者
   - 记录根因分析

**详细步骤**: 参见 `docs/operations/publish-hub-sla.md` 第 17-123 行

#### 告警 2: PublishOfflineSLAExceeded

**触发条件**:
```
plugin_offline_approval_duration_minutes > 1440m for 1h
```

**级别**: Warning

**处理流程**:
1. 立即响应（0-15 分钟）
   - 检查离线审核队列
   - 验证 .pxp 包下载状态
   - 确认白名单配置

2. 调查分析（15-30 分钟）
   - 检查包完整性
   - 验证签名和加密
   - 检查审核员指派

3. 执行修复（30-60 分钟）
   - 重新触发密钥解封
   - 通知审核员
   - 必要时回退上传

**详细步骤**: 参见 `docs/operations/publish-hub-sla.md` 第 133-223 行

### 通知渠道

- **Slack**: `#powerx-publish-hub`
- **PagerDuty**: On-call 工程师
- **Email**: `ops@powerx.dev`
- **Admin 控制台**: 实时状态更新

---

## 故障排除

### 在线审核问题

#### 问题 1: 自动化扫描失败

**症状**:
```
ERROR: automated security scan failed: high_severity_issues
```

**原因**:
- 扫描服务异常
- 网络连接问题
- 发现高危安全问题

**解决方案**:
```bash
# 1. 查看扫描日志
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/$PUBLISH_ID/scan-logs

# 2. 重新触发扫描
curl -X POST "https://api.powerx.dev/publish/$PUBLISH_ID/rescan" \
  -H "X-Powerx-User-Id: $USER_ID" \
  -H "X-Powerx-Role: marketplace_reviewer"

# 3. 查看扫描服务状态
kubectl get pods -n powerx-marketplace | grep scanner
```

#### 问题 2: 权限验证失败

**症状**:
```
ERROR: permission validation failed
```

**原因**:
- 插件权限声明不当
- 权限范围过大
- 缺少必要权限

**解决方案**:
1. 查看权限声明
2. 与开发者沟通调整
3. 重新提交审核

#### 问题 3: 依赖冲突

**症状**:
```
ERROR: dependency conflict detected
```

**原因**:
- 与其他插件冲突
- 依赖版本不兼容
- 命名空间冲突

**解决方案**:
1. 检查依赖列表
2. 验证兼容性
3. 要求开发者修复

#### 问题 4: 灰度发布配置错误

**症状**:
```
ERROR: rollout configuration invalid
```

**原因**:
- 租户列表格式错误
- 百分比计算错误
- 时间配置无效

**解决方案**:
```bash
# 验证灰度配置
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/publish/$PUBLISH_ID/rollout-config
```

### 离线审核问题

#### 问题 5: 密钥解封失败

**症状**:
```
ERROR: failed to unwrap symmetric key
```

**原因**:
- Marketplace 私钥错误
- 密钥封装数据损坏
- 算法不匹配

**解决方案**:
1. 验证 Marketplace 私钥
2. 重新上传包
3. 联系开发团队

#### 问题 6: 完整性验证失败

**症状**:
```
ERROR: integrity check failed: hash mismatch
```

**原因**:
- 文件在打包后被修改
- integrity.txt 错误
- 压缩算法不一致

**解决方案**:
1. 查看验证报告
2. 要求重新打包
3. 验证源文件完整性

#### 问题 7: 签名验证失败

**症状**:
```
ERROR: signature verification failed
```

**原因**:
- 私钥不匹配
- 签名数据损坏
- 证书过期

**解决方案**:
1. 验证证书链
2. 检查证书有效期
3. 要求重新签名

#### 问题 8: 租户白名单无效

**症状**:
```
ERROR: tenant whitelist validation failed
```

**原因**:
- 租户 ID 不存在
- 租户状态异常
- 格式错误

**解决方案**:
```bash
# 验证租户存在
curl -H "X-Powerx-User-Id: $USER_ID" \
     $PX_MARKETPLACE_API_URL/tenants/$TENANT_ID/status
```

---

## 审核最佳实践

### 1. 审核前准备

- [ ] 了解插件功能和背景
- [ ] 查看历史审核记录
- [ ] 确认有足够的审核时间
- [ ] 检查 SLA 截止时间

### 2. 审核过程

- [ ] 按检查清单逐项验证
- [ ] 记录发现的问题
- [ ] 截图保存重要信息
- [ ] 及时更新审核状态

### 3. 决策制定

- [ ] 权衡安全与效率
- [ ] 参考历史类似案例
- [ ] 咨询团队专家
- [ ] 记录决策依据

### 4. 沟通反馈

- [ ] 提供详细反馈
- [ ] 给出修改建议
- [ ] 保持专业态度
- [ ] 及时响应问题

### 5. 持续改进

- [ ] 定期回顾审核案例
- [ ] 总结常见问题
- [ ] 更新审核标准
- [ ] 分享经验教训

---

## 审核检查清单

### 在线发布审核清单

- [ ] **基础信息**
  - [ ] manifest.json 格式正确
  - [ ] 版本号递增且符合规范
  - [ ] 插件 ID 唯一
  - [ ] 描述完整准确

- [ ] **权限检查**
  - [ ] 权限声明合理
  - [ ] 权限范围最小化
  - [ ] 无过度授权
  - [ ] 权限用途明确

- [ ] **安全检查**
  - [ ] 自动化扫描通过
  - [ ] 无高危漏洞
  - [ ] 无恶意代码
  - [ ] 依赖安全

- [ ] **质量检查**
  - [ ] 代码质量可接受
  - [ ] 文档完整
  - [ ] 错误处理完善
  - [ ] 性能表现良好

- [ ] **合规检查**
  - [ ] 符合平台政策
  - [ ] 无违规内容
  - [ ] 隐私合规
  - [ ] 第三方授权完整

- [ ] **发布配置**
  - [ ] 灰度策略合理
  - [ ] 回滚方案完整
  - [ ] 租户列表正确
  - [ ] 通知设置正确

- [ ] **发布说明**
  - [ ] CHANGELOG 完整
  - [ ] 新功能说明
  - [ ] 修复问题列表
  - [ ] 升级指南

### 离线包审核清单

- [ ] **文件完整性**
  - [ ] .pxp 包存在
  - [ ] integrity.txt 完整
  - [ ] manifest.signature 存在
  - [ ] report.json 有效
  - [ ] audit.log 记录完整

- [ ] **加密与签名**
  - [ ] 密钥解封成功
  - [ ] 包解密成功
  - [ ] RSA-PSS 签名有效
  - [ ] 证书链验证通过
  - [ ] 证书未过期

- [ ] **完整性验证**
  - [ ] 所有文件哈希匹配
  - [ ] 无缺失文件
  - [ ] 无多余文件
  - [ ] 压缩格式正确

- [ ] **包内容检查**
  - [ ] manifest.json 正确
  - [ ] 版本号规范
  - [ ] 权限声明合理
  - [ ] 依赖关系完整

- [ ] **安全检查**
  - [ ] 无安全漏洞
  - [ ] 无恶意代码
  - [ ] 加密实现正确
  - [ ] 密钥封装安全

- [ ] **租户白名单**
  - [ ] 租户列表完整
  - [ ] 租户存在且正常
  - [ ] 授权有效
  - [ ] 数量符合预期

---

## 常见问题解答

### Q1: 如何处理边界情况？

**A**: 对于复杂边界情况：
1. 记录详细信息
2. 咨询团队专家
3. 参考历史案例
4. 制定临时方案
5. 更新审核标准

### Q2: 如何平衡安全与效率？

**A**: 建议做法：
1. 优先处理高风险发布
2. 使用自动化工具加速
3. 建立快速通道
4. 定期优化流程

### Q3: 遇到不确定的问题怎么办？

**A**: 处理流程：
1. 记录问题详情
2. 咨询有经验的同事
3. 查看相关文档
4. 必要时升级处理
5. 记录解决方案

### Q4: 如何处理开发者申诉？

**A**: 处理步骤：
1. 详细了解申诉内容
2. 重新审核相关问题
3. 与申诉者沟通
4. 如有误解则纠正
5. 更新审核记录

### Q5: 审核超时怎么办？

**A**: 应急措施：
1. 立即通知团队
2. 优先处理超时发布
3. 分配更多资源
4. 更新 SLA 状态
5. 事后分析改进

---

## 相关资源

### 文档
- [Quickstart 指南](../../004-publish-hub-spec/quickstart.md)
- [在线发布指南](./online.md)
- [离线发布指南](./offline.md)
- [SLA 运行手册](../../operations/publish-hub-sla.md)
- [安全文档](../../security/publish-hub.md)
- [RBAC 指南](../../operations/publish-hub-rbac.md)

### 代码位置
- 审核处理器: `framework/backend/go/runtime/marketplace/handlers/publish.go`
- 自动化扫描: `framework/backend/go/runtime/marketplace/services/scanner.go`
- 离线验证: `framework/backend/go/runtime/marketplace/services/offline_validator.go`
- 离线上传: `framework/backend/go/runtime/marketplace/handlers/offline_upload.go`
- 在线 SLA: `framework/backend/go/runtime/marketplace/services/online_sla_tracker.go`
- 离线 SLA: `framework/backend/go/runtime/marketplace/services/offline_sla_tracker.go`
- 审核事件: `framework/backend/go/runtime/marketplace/events/publish_events.go`

### 监控
- Grafana 仪表板: https://grafana.powerx.dev/d/publish-hub-sla
- Prometheus: https://prometheus.powerx.dev
- 告警管理: https://alertmanager.powerx.dev

### 界面
- 在线审核: https://admin.powerx.dev/marketplace/review
- 离线审核: https://admin.powerx.dev/marketplace/offline-review

### 支持
- Slack: `#powerx-publish-hub`
- 邮箱: `ops@powerx.dev`
- 文档反馈: https://github.com/ArtisanCloud/PowerXPlugin/issues

### 培训
- 审核员培训手册: `docs/training/marketplace-reviewer.md`
- 安全审核指南: `docs/security/review-guidelines.md`
- 平台政策文档: `docs/policies/platform-policies.md`

---

## 调试/沙箱诊断协同

Phase 11 引入的宿主模拟器与沙箱验证链路能协助审核员更快确认问题：

1. **宿主模拟器日志**  
   开发者运行 `px-plugin host start --mock --plugin <id>`（API：`POST /internal/dev/hosts/sessions`）后会获得 `sessionId` 与日志端点。审核员如需实时日志，可访问 `GET /internal/dev/hosts/sessions/{sessionId}/logs` 或在 Admin Dev Console 的 SSE 面板订阅 `plugin.debug.hot_reload`，确认热更新与断点同步情况。

2. **沙箱验证记录**  
   `px-plugin sandbox deploy --host-session <id>`（API：`POST /internal/dev/sandbox/deploy`）会输出 `validationId` 与数据集/测试执行结果。审核过程中可要求开发者在提交材料时附带该 ID 及 CLI 输出，快速了解覆盖率、性能与脱敏策略。

3. **调试报告**  
   `px-plugin debug report --session <id>` 触发 `POST /internal/dev/debug/report`，生成带 `debug.report.generate_ms` 指标的脱敏报告并同步工单系统。审核员可直接查看结构化报告，减少重复复现成本。

遇到需要联合诊断的场景，请确保目标环境开启 `debug-observability-v2` Feature Flag，并通过 Slack `#powerx-dev-hotload` 或 Admin 控制台的调试视图实时跟进。

## 发布计划视图（Admin）

新版 Admin 页面 `/_p/<tenant>/publish/pipelines`（参见 `examples/starter/web-admin/app/pages/publish/pipelines.vue`）提供以下能力，方便审核与运维协作：

1. **Plan 列表**：列出 `px-plugin publish create` 生成的计划，展示 `planId/publishId/channel/status`。
2. **策略编辑**：支持调整 rollout strategy、批次比例与等待时间，提交后会调用 `POST /internal/publish/create`。
3. **部署进度**：界面轮询 `GET /internal/publish/plans`（或 CLI 输出的部署状态），同步 `px-plugin publish deploy --strategy canary` 的批次进展，失败时提供回滚按钮。
4. **SLA 指标**：集成 `publish.local.iteration_cycle_time`、`publish.gray.error_rate`、`marketplace.listing.sla_hours`，当 Grafana 触发 `PublishOnlineSLAExceeded`/`PublishGrayErrorRate` 告警时在 UI 左侧显示警示条。

审核员在审批前可快速查看计划窗口、灰度批次与回滚 token，并与开发者对齐策略后再执行最终批准。
