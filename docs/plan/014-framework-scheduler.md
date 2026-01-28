# PowerXPlugin Scheduler Bridge 接入指引

本指引描述插件如何在宿主模式下注册调度任务，并通过 PowerX 底座统一 Scheduler 触发执行。

## 1. 目标
- 插件通过统一 Scheduler Bridge 注册/更新/暂停/触发计划任务。
- 触发事件由底座发布 `scheduler.job.triggered`，插件负责消费。
- 支持 `local | corex | dual` 模式与自动降级。

## 2. 配置位置
- **宿主模式**：PowerX 注入 `host-values.yaml` / `config.yaml` 的 `scheduler` 块（如需）。
- **Standalone 模式**：插件自带 `skeleton/backend/etc/config.yaml`。
- **不在 plugin.yaml 配置**底座地址。

## 3. Scheduler Bridge
### 模式
- `local`：本地调度（仅开发/单体）
- `corex`：调用 PowerX Scheduler（HTTP/gRPC/SDK）
- `dual`：双写/双读，便于灰度与回滚

### Manifest 示例
```yaml
scheduler:
  jobs:
    - name: "sync-knowledge"
      schedule_type: "cron"
      schedule_expr: "0 * * * *"
      payload:
        plugin_action: "knowledge.sync"
        params:
          space_id: "..."
```

### API 示例（HTTP）
```http
POST /api/v1/admin/scheduler/jobs
Authorization: Bearer <TOKEN>
x-tenant-uuid: <TENANT_UUID>

{
  "tenant_uuid": "...",
  "owner_type": "plugin",
  "owner_id": "com.powerx.helloworld",
  "name": "sync-knowledge",
  "schedule_type": "cron",
  "schedule_expr": "0 * * * *",
  "payload": {"plugin_action": "knowledge.sync", "params": {"space_id": "..."}}
}
```

## 4. 事件消费
- 订阅 `scheduler.job.triggered`。
- payload 包含 `job_id/tenant_uuid/owner_id/scheduled_at/payload`。

### Handler 示例（Go 伪代码）
```go
func (h *EventHandler) Handle(ctx context.Context, evt Event) error {
  if evt.Topic != "scheduler.job.triggered" {
    return nil
  }
  action := evt.Payload["plugin_action"].(string)
  params := evt.Payload["params"].(map[string]any)
  switch action {
  case "knowledge.sync":
    return h.KnowledgeSync(ctx, params)
  default:
    return nil
  }
}
```

## 5. 幂等与重试
- Scheduler 触发事件为至少一次投递，插件 handler 必须幂等。
- 建议使用 `job_id + scheduled_at` 去重。
- 失败返回 nack，由底座按 retry_policy 重试。

## 6. 运行时注入与降级
- 底座不可用时自动降级为 `local`（可配置）。
- 推荐日志：记录降级原因与恢复时间。
