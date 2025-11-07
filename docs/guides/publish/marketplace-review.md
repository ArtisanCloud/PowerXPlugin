# Marketplace 审核指南

本指南面向 Marketplace Reviewer，描述在线/离线审核操作步骤、SLA 指标与告警处理流程。

## 1. 在线审核
1. 打开 Admin `Marketplace → 审核队列`（`examples/starter/web-admin/app/pages/marketplace/review.vue` 样例）。
2. 查看 `publishId`、版本、渠道、自动化扫描结果（由 `framework/backend/go/runtime/marketplace/services/scanner.go` 输出）。
3. 若需要重跑自动化检查，可触发后台服务 `/runtime/marketplace/handlers/publish.go` 的再扫描。
4. 在 4 小时 SLA 内作出通过/拒绝决策；若即将超时，平台会触发 `PublishOnlineSLAExceeded` 告警（见 `config/alerts/publish-hub.yaml`）。

## 2. 离线审核
1. 上传 `.pxp` 与 `integrity.txt`，等待 `offline_sla_tracker.go` 记录审批耗时。
2. 在 1 个工作日 SLA 内完成审批，超时会触发 `PublishOfflineSLAExceeded` 告警并参见 runbook。

## 3. 审核 Checklist
- 验证签名、哈希与 manifest 一致。
- 确认权限、租户白名单及灰度策略。
- 扫描结果无高危。
- 记录审批结论与回滚方案（若适用）。

更多操作手册与 SLA 处理流程请参考 `docs/operations/publish-hub-sla.md`。
