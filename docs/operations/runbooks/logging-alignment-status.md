# 日志对齐状态清单（PowerXPlugin）

更新时间：2026-04-29

## 已完成对齐（核心链路）

- framework 统一门面与规则层
  - `framework/backend/go/runtime/common/logging/facade.go`
  - `framework/backend/go/runtime/common/logging/facade_test.go`
- framework runtime 总线链路
  - `framework/backend/go/runtime/wsbus/adapter.go`
  - `framework/backend/go/runtime/wsbus/redis_hub.go`
  - `framework/backend/go/runtime/taskbus/provider.go`
- skeleton 统一入口
  - `skeleton/backend/go-gin/internal/shared/app/deps.go`
- skeleton runtime ops
  - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/sessions_handler.go`
  - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/quota_handler.go`
  - `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/ws_bus_gateway_auth.go`
- skeleton integration
  - `skeleton/backend/go-gin/internal/integrations/gateway/client.go`
  - `skeleton/backend/go-gin/internal/services/integration/dispatch_service.go`
  - `skeleton/backend/go-gin/internal/jobs/integration/scheduler.go`
  - `skeleton/backend/go-gin/internal/jobs/integration/scheduler_event_dispatcher.go`
  - `skeleton/backend/go-gin/internal/jobs/integration/webhook_retry_worker.go`
  - `skeleton/backend/go-gin/internal/jobs/integration/secret_rotation_worker.go`
- skeleton marketplace / recommendation
  - `skeleton/backend/go-gin/internal/services/marketplace/analytics_service.go`
  - `skeleton/backend/go-gin/internal/services/marketplace/listing_service.go`
  - `skeleton/backend/go-gin/internal/services/marketplace/license_cache.go`
  - `skeleton/backend/go-gin/internal/services/marketplace/license_service.go`
  - `skeleton/backend/go-gin/internal/services/recommendation/engine.go`
- 查询 runbook
  - `docs/operations/runbooks/logging-query-alignment.md`

## 待对齐（高优先级）

- customer / federated auth 入口日志
  - `skeleton/backend/go-gin/internal/services/customer/delegate_authenticator.go`
  - `skeleton/backend/go-gin/internal/observability/customer/audit_logger.go`
  - `skeleton/backend/go-gin/internal/transport/http/public/auth/federated_handler.go`
- grpc server/client 运行日志
  - `skeleton/backend/go-gin/internal/grpc/server/server.go`
  - `skeleton/backend/go-gin/internal/grpc/client/powerx.go`
- capability catalog 与 manager
  - `skeleton/backend/go-gin/internal/capabilities/manager.go`
  - `skeleton/backend/go-gin/internal/integrations/powerx/capability_client.go`

## 待对齐（中优先级）

- security/ops 子域服务
  - `skeleton/backend/go-gin/internal/services/admin/security/privacy_service.go`
  - `skeleton/backend/go-gin/internal/services/admin/security/baseline_service.go`
  - `skeleton/backend/go-gin/internal/security/event_permissions.go`
- marketplace jobs 子域
  - `skeleton/backend/go-gin/internal/jobs/marketplace/license_renewal_notifier.go`
  - `skeleton/backend/go-gin/internal/jobs/marketplace/recommendation_sync.go`

## 对齐规则（执行口径）

- 固定 labels：`system/service/env/instance/module/plugin_id`
- 白名单 labels：`biz_scene/biz_domain`
- 高基数字段只进正文 fields：
  - `tenant_uuid/session_id/message_id/trace_id/request_id/user_id`
- 新增日志调用优先走 `Deps.RuntimeLogger(...)`，并补 `biz_scene/biz_domain`。

