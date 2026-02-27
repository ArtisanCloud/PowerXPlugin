# Tasks: Framework WS Bus Adapter

## Phase 1: Setup

- [x] T001 校对规范与范围并在 `docs/plan/develop/ws-bus-adapter.md` 记录本次实现边界

## Phase 2: Foundational

- [x] T002 定义 topic 白名单常量（`framework/backend/go/runtime/wsbus/whitelist.go`）
- [x] T003 定义统一发布接口与结果结构（`framework/backend/go/runtime/wsbus/publisher.go`）

## Phase 3: User Story 1 (P1) — Unified Progress Publish

**Story goal**: 业务层调用统一发布接口，不感知宿主/standalone。

**Independent test**: 宿主与 standalone 模式下触发同一发布接口，前端订阅均可收到。

- [x] T004 [P] [US1] 实现宿主模式发布客户端（`framework/backend/go/runtime/wsbus/host_client.go`）
- [x] T005 [P] [US1] 实现 standalone 本地 WS 发布适配（`framework/backend/go/runtime/wsbus/local_publisher.go`）
- [x] T006 [US1] 实现模式选择与统一入口（`framework/backend/go/runtime/wsbus/factory.go`）
- [ ] T007 [US1] 接入业务发布入口（确认模板事件发布位置后更新到真实路径）

## Phase 4: User Story 2 (P2) — Secure Tenant-Scoped Publishing

**Story goal**: 只允许授权与租户正确的发布请求。

**Independent test**: 非白名单 topic、无 tenant、无授权时发布失败。

- [x] T008 [US2] 校验 topic 白名单（`framework/backend/go/runtime/wsbus/validator.go`）
- [x] T009 [US2] 统一注入 tenant_uuid/trace_id（`framework/backend/go/runtime/wsbus/adapter.go`）
- [x] T010 [US2] 宿主模式发布请求注入 STS/JWT 鉴权（`framework/backend/go/runtime/wsbus/host_client.go` + `framework/backend/go/middleware/bearer_token.go`）

## Phase 5: User Story 3 (P3) — Task 驱动执行链路

**Story goal**: 任务执行由 Task 机制驱动，页面仅消费事件。

**Independent test**: 不依赖页面轮询即可完成任务执行与状态推进。

- [ ] T011 [US3] 记录“任务驱动 + 事件消费”验证步骤（`docs/guides/async_runtime/event_fabric/integration_playbook.md`）

## Phase 6: Polish & Cross-Cutting

- [ ] T012 更新本次功能快速验证步骤（`specs/015-framework-websocket/quickstart.md`）
- [ ] T013 补充验收说明与 topic 命名策略（`specs/015-framework-websocket/research.md`）
- [x] T014 补充发布失败的统一错误结构与示例（`framework/backend/go/runtime/wsbus/publisher.go`）
- [x] T015 [Polish] 补齐调试发布端点与 RBAC 对齐（Gin/FastAPI）

## Dependencies

- US1 完成后才能进行 US2 的鉴权与白名单校验联调
- US3 可并行验证但不阻塞 US1/US2

## Parallel Execution Examples

- T004 与 T005 可并行（宿主/standalone 适配分离）
- T008 与 T009 可并行（白名单与上下文注入分离）

## Implementation Strategy

先打通统一发布通路（US1），再加鉴权/白名单（US2），最后验证 Task 驱动链路与文档收敛（US3）。
