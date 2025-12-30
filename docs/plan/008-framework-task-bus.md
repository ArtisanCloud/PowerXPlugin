# PowerXPlugin Framework TaskBus 集成指引

本指引帮助将仓库中的渠道主数据与商品渠道任务迁移到 PowerXPlugin Framework TaskBus 提供的事件/观测通道。按步骤完成后，可在自身插件与宿主 PowerX 之间共享事件流、指标与后台任务调度能力。

## 1. 目标与范围

- **覆盖模块**：`backend/internal/jobs/channel/{product,master}`、`backend/internal/observability/channel/master`、`backend/internal/services/admin/{channel_master,product/spu}` 中所有直接写日志、直接触发 job 的逻辑。
- **输出能力**：审批 SLA 告警、渠道凭证巡检结果、KPI 刷新、SPU 发布/撤回/同步、任务中心反馈。
- **迁移原则**：保持业务层接口不变，新增事件发射器/监听器抽象，允许本地实现与 framework TaskBus 实现并存，便于灰度。

## 2. 事件契约设计

1. **命名规范**：`powerx.channel.<domain>.<action>.v1`。例如：
   - `powerx.channel.master.credential_inspection.v1`
   - `powerx.channel.product.publish_task.v1`
2. **通用字段**：
   - `tenantUuid`、`requestId`、`sourcePlugin`、`traceId`。
   - `occurredAt` (RFC3339)，`payloadVersion`。
3. **Payload Schema**：把现有 DTO/模型映射成事件体（可在 `docs/contracts/` 或 `contracts/channel-events.yaml` 内维护）。建议包含：
   - 凭证事件：`channelId`、`credentialType`、`expiresAt`、`status`、`alertLevel`。
   - KPI 事件：`channelId`、`window`、`metrics`（GMV、orders 等）、`healthScore`。
   - 发布事件：`spuId`、`versionId`、`channels[]`、`action`、`taskId`、`operator`。

## 3. 抽象接口

在 `backend/internal/observability/channel/master` 和 `backend/internal/jobs/channel/*` 中新增接口层：

```go
type ChannelEventEmitter interface {
    Emit(ctx context.Context, evt ChannelEvent) error
}

type ChannelEvent struct {
    Name    string
    Payload any
    Meta    map[string]string
}

type EventConsumer interface {
    Handle(ctx context.Context, evt ChannelEvent) error
}
```

- 默认实现：保留现有结构化日志/DB 写入，适用于本地或离线模式。
- TaskBus 实现：封装 `framework/backend/go/event` 提供的 `Emitter`、`Subscriber`，在 `app.Deps` 中注入，供服务层选择。

## 4. 框架接入步骤

1. **依赖注入**：在 `backend/cmd/plugin/main.go` 中创建 TaskBus 客户端（`frameworkevent.Client`），把 `ChannelEventEmitter` 实例放入 `app.Deps`。
2. **注册 Topic**：在 `plugin.yaml` 或 manifest 中声明事件发布/订阅权限，例如：
   ```yaml
   events:
     publish:
       - powerx.channel.master.*
     subscribe:
       - powerx.channel.product.*
   ```
3. **Handler 绑定**：使用 TaskBus router 将 `EventConsumer` 绑定到具体 Topic，支持并发度/重试配置：
   ```go
   frameworkevent.RegisterHandler(app, "powerx.channel.master.credential_inspection.v1", handler)
   ```
4. **配置开关**：在 `config/` 中增加 `EventBridge.Enabled` 标志，可在 `app.Deps` 中判断是走本地实现还是 TaskBus。

## 5. 代码迁移策略

| 步骤 | 动作 | 说明 |
| --- | --- | --- |
| S1 | 在服务层注入 `ChannelEventEmitter` | `ChannelMasterService`, `SPU Service` 等通过接口发事件。 |
| S2 | Job 改造为事件 Handler | 如 `CredentialChecker` 检测完后发事件，由 TaskBus Handler 写入 `channel_alerts`。 |
| S3 | 双写/双读 | 在一到两周内同时写旧日志与新事件；消费者同时监听旧路径与框架事件。 |
| S4 | 清理 Legacy | 待验证稳定后移除直接 DB/日志写入逻辑，仅保留事件驱动实现。 |

## 6. 测试与验证

- **单元测试**：为每个 emitter/handler 编写接口级别测试，模拟 TaskBus 事件输入输出。
- **集成测试**：启动本地 TaskBus (或使用内存 bus)，跑 `make test`/`make integration-smoke` 验证事件流。
- **回归脚本**：在 `reports/` 中记录迁移前后对 KPI/告警/任务链路的手动测试结果。

## 7. 运维与监控

- Dashboard：宿主 PowerX 可统一展示上述事件 Topic 的吞吐、失败率。
- 重试策略：使用 TaskBus 提供的死信队列或延迟重试，避免 job panic 影响主流程。
- 审计：事件 payload 中保留 operator/tenant，满足合规要求。

## 8. Checklist

1. [ ] 完成事件 schema 文档并通过评审。
2. [ ] 在 `app.Deps` 注入 TaskBus emitter/consumer。
3. [ ] 所有 job/observability 代码改用接口层。
4. [ ] 配置文件新增事件相关开关并默认开启本地实现。
5. [ ] 编写迁移计划与回滚步骤，更新 `docs/guides/channel_master.md`。
6. [ ] 在 staging 环境双写验证 ≥1 周，再切换到 TaskBus-only 模式。

完成上述步骤后，即使将 Channel 模块拆分为独立的 PowerXPlugin Framework 组件，也能直接复用这份事件抽象，实现即插即用的观测与后台任务能力。
