# PowerXPlugin Framework TaskBus 集成指引（Event Bridge）

本指引描述 PowerXPlugin Skeleton 后端如何通过统一的事件出口（EventBridge）对接宿主 TaskBus（若可用），并在 TaskBus 不可用时自动降级到本地实现，支持灰度/回滚与契约治理。

本仓库对应的落地实现与 Spec/Tasks 位于：`specs/008-framework-task-bus/`。
快速上手与验证步骤见：`specs/008-framework-task-bus/quickstart.md`。
开发前准备与缺口清单见：`specs/008-framework-task-bus/readiness.md`。

## 1. 目标与范围

- **目标**：把“直接写日志/直接触发 job/直接写 DB”的副作用收敛到统一事件出口；业务侧只负责“发事件/处理事件”。
- **范围**：以 Channel 域事件为先导（凭证巡检、KPI 刷新、SPU 发布任务），逐步扩展到其他域。
- **约束**：多租户上下文使用 `tenant_uuid`；Topic 权限按最小权限声明；TaskBus 不可用时主流程不 panic（自动降级）。

## 2. 事件契约设计

1. **命名规范**：`powerx.channel.<domain>.<action>.v1`。例如：
   - `powerx.channel.master.credential_inspection.v1`
   - `powerx.channel.product.publish_task.v1`
2. **通用字段**：
   - `tenant_uuid`、`request_id`、`source_plugin`、`trace_id`。
   - `occurred_at`（RFC3339），`payload_version`。
3. **契约文件（仓库内治理）**：事件契约集中在 `specs/008-framework-task-bus/contracts/channel-events.yaml`，并由 CI 校验（topic 唯一、meta.required 必填字段、敏感字段名 lint）。
   - 本地校验入口：`./scripts/contracts/validate-taskbus-contracts.sh`
   - 校验实现：`tools/contracts/validate-taskbus-contracts.go`

## 3. 抽象接口

- **事件模型（Framework 对外包）**：`framework/backend/go/event/*`
- **事件出口（Framework 对外包）**：`framework/backend/go/eventbridge/*`
- **业务侧适配器（示例：Channel）**：`skeleton/backend/go-gin/internal/observability/channel/event_emitter.go`
- **Consumer/Dispatcher（Framework）**：`framework/backend/go/eventbridge/consumer.go`
- **权限与运行时边界**：`skeleton/backend/go-gin/internal/security/event_permissions.go`（从 `skeleton/plugin.yaml` 读取 publish/subscribe 并执行 deny + log）

说明：
- 本仓库以“本地 emitter + 可注入 TaskBus provider”的方式完成切换与灰度；真实 TaskBus SDK 由宿主/框架提供后再实现 provider。

## 4. 框架接入步骤

1. **依赖注入**：在 `skeleton/backend/go-gin/cmd/plugin/main.go` 初始化 `event_bridge.Factory` 并注入到 `app.Deps.EventEmitter`。
2. **声明 Topic 权限（开发态）**：在 `skeleton/plugin.yaml` 增加：
   ```yaml
   events:
     publish:
       - powerx.channel.master.credential_inspection.v1
       - powerx.channel.master.kpi_refreshed.v1
       - powerx.channel.product.publish_task.v1
     subscribe: []
   ```
   - Manifest 路径可通过环境变量覆盖：`POWERX_PLUGIN_MANIFEST_PATH`
3. **配置开关**：在 `skeleton/backend/go-gin/internal/config/config.go` 使用 `event_bridge` 配置：
   - `event_bridge.enabled`：开启/关闭 TaskBus 模式
   - `event_bridge.mode`：`local|taskbus|dual`
   - `event_bridge.fallback_to_local`：TaskBus 不可用时是否自动降级
4. **消费侧绑定（示例）**：本仓库提供本地 `Dispatcher` + `IdempotencyFilter` 的示例实现；真实 TaskBus 订阅由宿主/框架接入后完成。

## 4.5 Scheduler Bridge（统一调度入口）
**接口优先级：**
- HTTP 为第一优先。
- gRPC 与 SDK 作为后续扩展（与宿主能力保持一致）。

**幂等与重试：**
- Scheduler 触发事件为至少一次投递，插件 handler 必须幂等。
- 建议使用 `event_id` 或 `job_id + scheduled_at` 做去重。
- 失败返回 nack，将由底座按 retry_policy 重试。

**请求头约定（调用底座）：**
- `Authorization: Bearer <TOKEN>`
- `x-powerx-tenant: <TENANT_UUID>`
- `Idempotency-Key: <uuid>`（可选）

**Handler 示例（Go 伪代码）：**
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


- **目标**：让插件用统一接口注册/更新计划任务，并可切换到底座 Scheduler 或本地实现。
- **模式**：
  - `local`：仅本地调度
  - `corex`：调用 PowerX Scheduler（HTTP/gRPC/SDK）
  - `dual`：双写/双读，便于灰度与回滚
- **降级**：底座不可用时自动降级到本地实现（不影响主流程）。

**配置示例（plugin config）：**
```yaml
scheduler_bridge:
  enabled: true
  mode: corex
  fallback_to_local: true
```

**Manifest 示例：**
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

**消费约定：**
- Scheduler 触发事件 `scheduler.job.triggered`。
- 插件按 Manifest 声明订阅，并由统一 handler 处理。

## 5. 代码迁移策略

| 步骤 | 动作 | 说明 |
| --- | --- | --- |
| S1 | 在服务层注入 `EventEmitter` | 业务层通过统一 emitter 发事件（Topic + payload + meta）。 |
| S2 | 把“处理副作用”迁移到 Consumer | 例如将巡检结果写入/告警改为 consumer handler 处理。 |
| S3 | 双写/双读 | `event_bridge.mode=dual` 支持迁移期双写；对比旧链路与新链路结果一致性。 |
| S4 | 清理 Legacy | 待验证稳定后移除直接 DB/日志写入逻辑，仅保留事件驱动实现。 |

## 6. 测试与验证

- **单元测试**：为每个 emitter/handler 编写接口级别测试，模拟 TaskBus 事件输入输出。
- **集成测试**：启动本地 TaskBus (或使用内存 bus)，跑 `make test`/`make integration-smoke` 验证事件流。
- **回归脚本**：在 `reports/` 中记录迁移前后对 KPI/告警/任务链路的手动测试结果。

## 7. 运维与监控

- 指标：本仓库提供最小 metrics hooks（Prometheus exposition），见 `skeleton/backend/go-gin/internal/observability/event_bridge/metrics.go`。
- 指标抓取（本仓库 Skeleton Admin）：`GET /api/v1/admin/runtime/metrics`
- 建议关注：
  - `plugin_event_bridge_emit_total` / `plugin_event_bridge_consume_total`（按 topic/tenant_uuid/result）
  - `plugin_event_bridge_latency_ms`（emit/consume 最近一次延迟（ms），按 op/topic/tenant_uuid）
- 回滚：当 TaskBus 不可用/失败率升高时，优先切回 `event_bridge.enabled=false` 或 `event_bridge.mode=local`（本地实现兜底）。

## 8. Checklist

1. [ ] 契约变更走 PR + CI 校验（`./scripts/contracts/validate-taskbus-contracts.sh`）。
2. [ ] 确认 manifest 事件权限（`skeleton/plugin.yaml`）满足最小权限，并在运行时 enforcement 生效。
3. [ ] 在 `app.Deps` 注入 emitter，并完成 TaskBus provider 对接（若宿主 SDK 已就绪）。
4. [ ] 业务路径完成至少一个 job → consumer 的迁移示例，并保留双写/回滚路径。
5. [ ] staging 双写验证 ≥1 周，对比一致性与 `emit/consume` 指标趋势后，再切换到 TaskBus-only 模式。

完成上述步骤后，即使将 Channel 模块拆分为独立的 PowerXPlugin Framework 组件，也能直接复用这份事件抽象，实现即插即用的观测与后台任务能力。
