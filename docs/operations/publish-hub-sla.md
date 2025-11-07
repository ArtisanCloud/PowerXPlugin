# Publish Hub SLA Runbook

## Online Review (≤4h)
- Metric: `plugin_publish_pipeline_duration_ms`
- Alert: `PublishOnlineSLAExceeded`
- Action:
  1. 查看 reviewer 队列是否堆积。
  2. 若自动化扫描卡住，重启 `marketplace-review` worker。
  3. 告知发布者预计恢复时间。

## Offline Review (≤1 workday)
- Metric: `plugin_offline_approval_duration_minutes`
- Alert: `PublishOfflineSLAExceeded`
- Action:
  1. 核对 `.pxp` 下载/解密是否失败。
  2. 确认 reviewer 是否收到白名单指派。
  3. 在 2 小时内给出审核更新，必要时回退上传。

## Rollback Latency (≤5min)
- Metric: `plugin_install_rollback_latency_seconds`
- Alert: `PluginRollbackLatencyExceeded`
- Action:
  1. 检查 Admin 安装任务与日志。
  2. 确认回滚脚本/脚本权限正常。
  3. 若超 SLA，通知租户并触发 incident。
