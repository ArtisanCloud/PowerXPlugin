# Readiness Checklist — 008-framework-task-bus

> 目的：为“下一阶段 Event / TaskBus 开发”提供统一开工基线，明确**已完成**、**未完成**与**落地顺序**。

## 1) 当前实现状态（基于本仓库代码）

### 1.1 已完成

- 统一事件模型：`framework/backend/go/event/models.go`、`framework/backend/go/event/meta.go`
- 统一发事件接口：`framework/backend/go/eventbridge/emitter.go`
- 本地 emitter（队列）与 dual/fallback 基础能力：`framework/backend/go/eventbridge/local_emitter.go`
- Dispatcher + 幂等过滤：`framework/backend/go/eventbridge/consumer.go`、`framework/backend/go/eventbridge/idempotency.go`
- TaskBus Provider 抽象 + Stub：`framework/backend/go/eventbridge/taskbus_provider.go`、`framework/backend/go/eventbridge/taskbus_stub.go`
- Skeleton 配置项：`event_bridge.enabled|mode|fallback_to_local|local_queue_size|taskbus_provider`
- Provider 解析优先级：`config.yaml(event_bridge.taskbus_provider)` > `POWERX_EVENT_BRIDGE_TASKBUS_PROVIDER`（仅本地兜底） > `host`
- 指标：`emit_total` / `consume_total` / `latency_ms`

### 1.2 未完成（本次开发重点）

- **真实宿主 TaskBus Provider 未落地**（当前仅接口与 stub）
- `cmd/plugin` 已注入 `WithTaskBusProvider(...)`（当前为 host/stub 选择与兜底，真实 host provider 待落地）
- LocalEmitter 队列满已记录 drop 指标（下一步补充告警阈值与运行手册）
- 版本发布与迁移节奏（`v0.0.3-alpha+`）未形成固定流程

## 2) 设计口径（必须统一）

- 事件语义：**at-least-once**
- 幂等键默认：`topic + tenant_uuid + trace_id`
- 模式：`local | taskbus | dual`
- 降级：`taskbus` 不可用时，按 `fallback_to_local` 决定是否回落
- 事件最小必填 meta：`tenant_uuid`、`request_id`、`source_plugin`、`trace_id`、`occurred_at`、`payload_version`

## 3) 开工前检查（Day-0）

- [ ] 确认 `skeleton/backend/go-gin/go.mod` 使用本地 replace 或升级到目标 framework 版本。
- [ ] 确认 `event_bridge` 配置已加载且通过校验。
- [ ] 确认 `events.publish/subscribe` 在 `skeleton/plugin.yaml` 已按最小权限声明。
- [ ] 确认 runtime 调试路由可访问（默认注册 `event-bridge/emit`）。
- [ ] 本地回归：`go test ./skeleton/backend/go-gin/... -run 'EventBridge|event_bridge' -v`。

## 4) 下一阶段实施顺序（建议）

1. 实现 Host TaskBus Provider（framework/runtime）。
2. 在 `cmd/plugin` 注入 provider（`Factory.WithTaskBusProvider`）。
3. 增加 drop 计数 + 告警字段（local 队列满）。
4. 补齐集成测试矩阵（taskbus/dual/fallback/permission）。
5. 发布新 alpha，输出迁移说明（外部插件只需 adapter 映射）。

## 5) 完成定义（DoD）

- [ ] `mode=taskbus` 且 provider 可用时，事件经宿主链路发出。
- [ ] `mode=taskbus` 且 provider 异常时，`fallback_to_local=true` 可自动回落。
- [ ] `mode=dual` 时，主链路失败短路；主成功后再写本地。
- [ ] 指标包含 `success/error`，并可区分 emit/consume。
- [ ] 迁移文档可指导外部插件在 1 次迭代内接入。

## 6) 一次性开工命令清单（可直接执行）

### 6.1 Day-0 校验

```bash
# 1) 契约校验
./scripts/contracts/validate-taskbus-contracts.sh
#    备用：make -f make-files/validate.mk validate-taskbus-contracts

# 2) EventBridge 相关快速回归
mkdir -p tmp/gocache
GOCACHE="$PWD/tmp/gocache" go test ./skeleton/backend/go-gin/... -run 'EventBridge|event_bridge' -v

# 3) 全量回归（可选，耗时较长）
go test ./skeleton/backend/go-gin/...
```

### 6.2 本地联调（event-bridge debug endpoint）

```bash
# 终端 A：启动后端（runtime 调试路由默认开启）
cd skeleton/backend/go-gin
go run ./cmd/plugin

# 终端 B：触发一次 emit（仅开发调试）
curl -sSf -X POST http://127.0.0.1:8078/api/v1/admin/runtime/event-bridge/emit \
  -H 'Content-Type: application/json' \
  -d '{
    "tenant_uuid": "00000000-0000-0000-0000-000000000001",
    "topic": "powerx.channel.master.credential_inspection.v1",
    "payload": {"channel_id":"c1","status":"ok"}
  }'

# 终端 B：检查指标
curl -sSf http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_'
```

## 7) 角色分工与交付物（建议）

- Framework Owner：负责 `TaskBusProvider` 真实实现、Factory 注入、错误语义稳定。
- Plugin Owner：负责业务侧 emitter 接入、topic 权限声明、灰度回归。
- QA/Platform：负责 taskbus/dual/fallback 回归矩阵、指标告警与发布验收。

发布前必须产出：

- [ ] provider 连通性测试记录（taskbus 可用/不可用）
- [ ] fallback 验证记录（`fallback_to_local=true`）
- [ ] 指标截图（emit/consume/latency + error）
- [ ] 外部插件迁移清单（见 docs/guides/async_runtime/event_fabric/integration_playbook.md）
