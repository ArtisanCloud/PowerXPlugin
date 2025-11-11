# Publish Hub SLA Runbook

## 概述

本文档包含 Publish Hub 系统所有 SLA 指标的监控、告警和应急响应程序。运维团队应按照此文档进行日常监控和事件处理。

## SLA 目标

| 指标 | 目标值 | 监控指标 | 告警级别 |
|------|--------|----------|----------|
| 在线发布审核 | ≤ 4 小时 | `plugin_publish_pipeline_duration_ms` | Warning |
| 离线发布审核 | ≤ 1 个工作日 | `plugin_offline_approval_duration_minutes` | Warning |
| 插件回滚延迟 | ≤ 5 分钟 | `plugin_install_rollback_latency_seconds` | Critical |
| Dev 热加载 | ≤ 2 秒 | `dev.hotload.cli_reload_duration_ms` | Warning |
| 部署成功率 | ≥ 99% | `plugin_deployments_total` | Critical |

## 在线审核超时 (Online Review)

### 监控指标
- **Prometheus 指标**: `plugin_publish_pipeline_duration_ms`
- **查询语句**: `histogram_quantile(0.95, rate(plugin_publish_pipeline_duration_ms_bucket[5m]))`
- **阈值**: 4 小时 (14400 秒)
- **告警名称**: `PublishOnlineSLAExceeded`

### 告警触发条件
```
plugin_publish_pipeline_duration_ms > 4h for 15m
```

### 应急响应流程

#### 1. 立即响应 (0-5 分钟)
1. **确认告警**：
   - 登录 Grafana：https://grafana.powerx.dev/d/publish-hub-sla
   - 定位超时的发布：查看 `publishId` 和 `pluginId`
   - 检查过去 1 小时的发布数量趋势

2. **初步诊断**：
   ```bash
   # 查看在线审核队列状态
   curl -X GET "https://api.powerx.dev/marketplace/pending-reviews" \
     -H "Authorization: Bearer $TOKEN"

   # 检查 reviewer 工作负载
   ps aux | grep marketplace-review
   ```

3. **查看相关日志**：
   ```bash
   # 查看 Marketplace 审核日志
   tail -f /var/log/powerx/marketplace-review.log | \
     grep -E "(timeout|slow|queue)"

   # 查看自动化扫描日志
   tail -f /var/log/powerx/scanner.log | \
     grep -E "(failed|error|stalled)"
   ```

#### 2. 调查分析 (5-15 分钟)
1. **检查队列积压**：
   - 登录 Admin 控制台：https://admin.powerx.dev/marketplace/review
   - 统计未审核的发布数量
   - 查看每个发布待审核时长

2. **审查员状态检查**：
   ```bash
   # 检查活跃的 reviewer
   kubectl get pods -n powerx-marketplace | \
     grep -E "reviewer|worker"

   # 查看 reviewer CPU/内存使用
   kubectl top pods -n powerx-marketplace
   ```

3. **自动化扫描状态**：
   - 检查 `scanner` 服务状态
   - 查看扫描任务队列长度
   - 验证第三方安全扫描 API 连接

#### 3. 执行修复 (15-30 分钟)
1. **扩容 reviewer 资源**：
   ```bash
   # 扩容 marketplace-review 部署
   kubectl scale deployment marketplace-review \
     -n powerx-marketplace --replicas=5

   # 验证扩容状态
   kubectl get pods -n powerx-marketplace \
     -l app=marketplace-review
   ```

2. **重启卡住的服务**：
   ```bash
   # 重启 scanner 服务
   kubectl rollout restart deployment scanner \
     -n powerx-marketplace

   # 重启 review worker
   kubectl rollout restart deployment review-worker \
     -n powerx-marketplace
   ```

3. **清理积压队列**：
   - 手动将部分发布标记为 "需人工审核"
   - 优先处理超时最长的发布
   - 通知相关发布者当前进度

#### 4. 后续跟进 (30-60 分钟)
1. **监控恢复**：
   - 观察告警状态变为 "Resolved"
   - 确认新发布的审核时间回归正常
   - 记录恢复时间

2. **通知利益相关方**：
   - 发送状态更新邮件给受影响的发布者
   - 更新内部状态页：https://status.powerx.dev
   - 在 #powerx-publish-hub Slack 频道发布总结

3. **根因分析**：
   - 整理事件时间线
   - 分析根本原因
   - 制定改进措施

### 常见原因和解决方案

| 原因 | 症状 | 解决方案 |
|------|------|----------|
| Reviewer 资源不足 | CPU/内存使用率 > 80% | 扩容或优化 reviewer 性能 |
| 自动化扫描卡住 | Scanner 队列积压 > 100 | 重启 scanner 或检查 API 连接 |
| 数据库连接池耗尽 | Review 请求超时 | 增加连接池大小或优化查询 |
| 第三方 API 限流 | 安全扫描响应慢 | 与第三方协商提升限额或缓存结果 |

## 离线审核超时 (Offline Review)

### 监控指标
- **Prometheus 指标**: `plugin_offline_approval_duration_minutes`
- **查询语句**: `histogram_quantile(0.95, rate(plugin_offline_approval_duration_minutes_bucket[15m]))`
- **阈值**: 1 个工作日 (1440 分钟)
- **告警名称**: `PublishOfflineSLAExceeded`

### 告警触发条件
```
plugin_offline_approval_duration_minutes > 1440m for 1h
```

### 应急响应流程

#### 1. 立即响应 (0-15 分钟)
1. **检查离线上传状态**：
   ```bash
   # 查看离线审核队列
   curl -X GET "https://api.powerx.dev/marketplace/offline-queue" \
     -H "Authorization: Bearer $TOKEN"

   # 检查 .pxp 包下载状态
   ls -lh /data/offline-packages/*.pxp | head -20
   ```

2. **验证白名单配置**：
   - 登录 Admin 控制台：https://admin.powerx.dev/marketplace/offline-review
   - 确认目标租户在白名单中
   - 验证租户状态是否正常

3. **检查密钥解封状态**：
   ```bash
   # 查看密钥解封日志
   tail -f /var/log/powerx/key-unwrap.log | \
     grep -E "(failed|error|timeout)"

   # 验证 KMS 连接
   kubectl exec -it kms-client-pod -n powerx-marketplace -- \
     vault status
   ```

#### 2. 调查分析 (15-30 分钟)
1. **检查 .pxp 包完整性**：
   ```bash
   # 验证包完整性
   for file in /data/offline-packages/*.pxp; do
     echo "Checking $file..."
     px-plugin verify --package "$file"
   done

   # 检查包大小和哈希
   md5sum /data/offline-packages/*.pxp
   ```

2. **验证签名和加密**：
   - 检查 RSA-PSS 签名验证
   - 确认 AES-256-GCM 密钥封装正确
   - 验证证书链完整性

3. **审查员指派检查**：
   ```bash
   # 查看离线 reviewer 指派
   curl -X GET "https://api.powerx.dev/marketplace/reviewers/offline"

   # 检查 reviewer 工作负载
   curl -X GET "https://api.powerx.dev/marketplace/reviewers/status"
   ```

#### 3. 执行修复 (30-60 分钟)
1. **重新触发密钥解封**：
   ```bash
   # 手动触发密钥解封
   curl -X POST "https://api.powerx.dev/marketplace/offline/$PACKAGE_ID/unwrap-key" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"retry": true}'
   ```

2. **通知 reviewer**：
   - 发送 Slack 通知：@reviewer-team
   - 分配超时包给可用 reviewer
   - 设置 2 小时审核截止时间

3. **必要时回退上传**：
   ```bash
   # 标记包为需要重新上传
   curl -X POST "https://api.powerx.dev/marketplace/offline/$PACKAGE_ID/rollback" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"reason": "SLA_timeout"}'
   ```

## 回滚延迟超时 (Rollback Latency)

### 监控指标
- **Prometheus 指标**: `plugin_install_rollback_latency_seconds`
- **告警名称**: `PluginRollbackLatencyExceeded`
- **阈值**: 5 分钟 (300 秒)
- **告警级别**: **Critical**

### 告警触发条件
```
plugin_install_rollback_latency_seconds > 300s for 5m
```

### 应急响应流程 (立即响应)

#### 1. 紧急响应 (0-2 分钟)
1. **确认受影响租户**：
   ```bash
   # 获取超时回滚详情
   curl -X GET "https://api.powerx.dev/admin/deployments?status=rolling_back" \
     -H "Authorization: Bearer $TOKEN" | \
     jq '.[] | select(.duration > 300) | {deploymentId, tenantId, pluginId, startTime}'
   ```

2. **检查回滚状态**：
   ```bash
   # 查看 plugin deployer 状态
   kubectl logs -n powerx-admin deployment/plugin-deployer | \
     grep -E "rolling_back|timeout|failed" | tail -20

   # 查看回滚计时器状态
   kubectl exec -it plugin-deployer-pod -n powerx-admin -- \
     curl -X GET "http://localhost:8080/admin/internal/deployments/status"
   ```

3. **评估影响范围**：
   - 确认是否影响多个租户
   - 检查是否影响生产环境
   - 评估用户访问影响

#### 2. 诊断问题 (2-5 分钟)
1. **检查部署状态**：
   ```bash
   # 查看 deployment 详细信息
   kubectl describe deployment plugin-deployer -n powerx-admin

   # 查看 Pod 资源使用
   kubectl top pods -n powerx-admin

   # 检查 Pod 事件
   kubectl get events -n powerx-admin --sort-by='.lastTimestamp'
   ```

2. **查看回滚日志**：
   ```bash
   # 过滤回滚相关日志
   kubectl logs -n powerx-admin deployment/plugin-deployer | \
     grep -A 10 -B 5 "performRollback"

   # 查看具体 deployment 的回滚日志
   kubectl logs -n powerx-admin deployment/plugin-deployer | \
     grep "deploymentId=$DEPLOYMENT_ID"
   ```

3. **检查外部依赖**：
   - 确认数据库连接正常
   - 检查存储服务可用性
   - 验证网络连通性

#### 3. 执行修复 (5-10 分钟)
1. **手动触发回滚**：
   ```bash
   # 强制执行回滚
   kubectl exec -it plugin-deployer-pod -n powerx-admin -- \
     curl -X POST "http://localhost:8080/admin/internal/deployments/$DEPLOYMENT_ID/rollback" \
     -H "Content-Type: application/json" \
     -d '{"force": true}'
   ```

2. **重启部署服务**：
   ```bash
   # 重启 plugin deployer
   kubectl rollout restart deployment plugin-deployer -n powerx-admin

   # 等待重启完成
   kubectl rollout status deployment plugin-deployer -n powerx-admin
   ```

3. **扩容资源**：
   ```bash
   # 临时扩容以处理积压
   kubectl scale deployment plugin-deployer \
     -n powerx-admin --replicas=3
   ```

#### 4. 验证和监控 (10-20 分钟)
1. **确认回滚完成**：
   ```bash
   # 检查回滚状态
   curl -X GET "https://api.powerx.dev/admin/deployments/$DEPLOYMENT_ID" \
     -H "Authorization: Bearer $TOKEN" | \
     jq '.status'

   # 验证租户服务恢复
   curl -X GET "https://api.powerx.dev/tenant/$TENANT_ID/health"
   ```

2. **监控新部署**：
   - 观察是否有新的回滚需求
   - 监控服务资源使用情况
   - 确认告警状态回归正常

3. **用户通知**：
   ```bash
   # 通知租户管理员
   curl -X POST "https://api.powerx.dev/tenant/$TENANT_ID/notify" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{
       "type": "rollback_completed",
       "message": "插件回滚已完成，服务已恢复"
     }'
   ```

### 常见原因和解决方案

| 原因 | 症状 | 解决方案 |
|------|------|----------|
| 回滚脚本执行缓慢 | 脚本运行 > 5 分钟 | 优化脚本性能或并行化 |
| 存储 I/O 瓶颈 | 磁盘使用率 > 90% | 清理磁盘或扩容存储 |
| 数据库锁等待 | 事务超时 | 优化锁策略或重试机制 |
| 网络连接问题 | 连接超时 | 检查网络或启用重试 |
| 资源不足 | CPU/内存使用率 > 90% | 扩容或优化资源使用 |

## Dev 热加载超时 (Dev Hotload)

### 监控指标
- **Prometheus 指标**: `dev.hotload.cli_reload_duration_ms`
- **阈值**: 2 秒
- **告警名称**: `DevHotloadSlow`

### 应急响应流程

1. **检查 CLI 构建性能**：
   ```bash
   # 查看构建时间分布
   kubectl logs -n powerx-devapi deployment/dev-api | \
     grep "reload_duration" | tail -100
   ```

2. **优化构建流程**：
   - 启用增量构建
   - 增加资源限制
   - 优化构建缓存

## 部署成功率低 (Low Deployment Success Rate)

### 监控指标
- **Prometheus 指标**: `plugin_deployments_total`
- **计算公式**: `rate(plugin_deployments_total{status="success"}[5m]) / rate(plugin_deployments_total[5m])`
- **阈值**: < 99%
- **告警级别**: Critical

### 应急响应流程

1. **分析失败原因**：
   ```bash
   # 查看失败部署统计
   kubectl logs -n powerx-admin deployment/plugin-deployer | \
     grep -E "markFailed|error" | \
     jq -r '.reason' | sort | uniq -c
   ```

2. **修复通用问题**：
   - 网络连接问题 → 检查防火墙和路由
   - 资源不足 → 扩容或优化资源分配
   - 配置错误 → 验证配置参数
   - 依赖服务异常 → 重启依赖服务

3. **回滚失败的部署**：
   ```bash
   # 批量回滚失败部署
   curl -X POST "https://api.powerx.dev/admin/deployments/batch-rollback" \
     -H "Authorization: Bearer $TOKEN" \
     -d '{"status": "failed", "older_than": "1h"}'
   ```

## 告警配置

### Alertmanager 配置

导入 `config/alerts/publish-hub.yaml` 到 Alertmanager：

```yaml
# 全局配置
global:
  slack_api_url: 'SLACK_WEBHOOK_URL'
  pagerduty_url: 'PAGERDUTY_INTEGRATION_KEY'

# 路由配置
route:
  group_by: ['alertname']
  group_wait: 10s
  group_interval: 10s
  repeat_interval: 1h
  receiver: 'default'
  routes:
  - match:
      severity: critical
    receiver: 'pagerduty-critical'
  - match:
      severity: warning
    receiver: 'slack-warnings'

# 接收器配置
receivers:
- name: 'default'
  email:
    to: 'ops@powerx.dev'
    subject: 'Publish Hub Alert'

- name: 'slack-warnings'
  slack_configs:
  - channel: '#powerx-publish-hub'
    title: 'Publish Hub Warning'
    text: '{{ range .Alerts }}{{ .Annotations.summary }}{{ end }}'

- name: 'pagerduty-critical'
  pagerduty_configs:
  - service_key: 'PAGERDUTY_INTEGRATION_KEY'
    description: 'Critical Publish Hub Incident'
```

### Grafana 仪表板

**仪表板 URL**: https://grafana.powerx.dev/d/publish-hub-sla

包含以下 Panel：
1. 在线审核时延（95th percentile）
2. 离线审核时延（95th percentile）
3. 回滚延迟（平均值 + 最大值）
4. Dev 热加载时延
5. 部署成功率趋势
6. 活跃部署数量
7. 失败部署 Top 10
8. 告警状态总览

## 联系人

### 值班工程师
- **Primary**: On-call Engineer (PagerDuty)
- **Secondary**: Senior Engineer (Slack @powerx-oncall)

### 团队联系人
- **DevOps Team**: #devops
- **Marketplace Team**: #marketplace
- **Plugin Team**: #plugin-team

### 外部联系人
- **安全团队**: security@powerx.dev
- **网络团队**: network@powerx.dev
- **存储团队**: storage@powerx.dev

## 事后分析模板

### 事件概述
- **开始时间**: YYYY-MM-DD HH:MM:SS UTC
- **结束时间**: YYYY-MM-DD HH:MM:SS UTC
- **总时长**: X 分钟
- **影响范围**: 描述受影响的用户/租户/功能
- **告警数量**: N 个告警

### 时间线
1. **T0**: 告警触发
2. **T+5m**: 运维团队响应
3. **T+15m**: 完成问题诊断
4. **T+30m**: 执行修复措施
5. **T+45m**: 验证修复成功
6. **T+60m**: 关闭告警

### 根因分析
- **直接原因**:
- **根本原因**:
- **促成因素**:

### 改进措施
1. **短期措施** (1 周内):
   - [ ] Action 1
   - [ ] Action 2

2. **长期措施** (1 个月内):
   - [ ] Action 1
   - [ ] Action 2

3. **预防措施**:
   - [ ] 监控改进
   - [ ] 流程优化
   - [ ] 自动化提升

### 经验教训
- What went well?
- What could be improved?
- What should we do differently next time?

## 附录

### 常用命令速查

```bash
# 查看部署状态
kubectl get deployments -n powerx-admin

# 查看 Pod 日志
kubectl logs -f deployment/plugin-deployer -n powerx-admin

# 扩容服务
kubectl scale deployment plugin-deployer -n powerx-admin --replicas=5

# 查看 Grafana 仪表板
open https://grafana.powerx.dev/d/publish-hub-sla

# 查看 Prometheus 指标
open https://prometheus.powerx.dev/graph

# 触发手动回滚
curl -X POST "https://api.powerx.dev/admin/deployments/$ID/rollback" \
  -H "Authorization: Bearer $TOKEN"
```

### 监控链接

- **Grafana**: https://grafana.powerx.dev/d/publish-hub-sla
- **Prometheus**: https://prometheus.powerx.dev
- **Alertmanager**: https://alertmanager.powerx.dev
- **Admin Console**: https://admin.powerx.dev
- **Marketplace Console**: https://marketplace.powerx.dev

### 升级流程

当遇到无法解决的问题时：

1. **升级到高级工程师**:
   ```bash
   # 在 Slack 发起 @here
   /channel: #powerx-publish-hub
   @here Critical issue: Publish Hub SLA violation, need assistance
   ```

2. **升级到管理层**:
   - 如果影响 > 1 小时
   - 如果影响 > 10 个租户
   - 如果涉及安全或数据丢失

3. **发布状态页**:
   - 更新 https://status.powerx.dev
   - 通知所有用户
   - 设置预计恢复时间
