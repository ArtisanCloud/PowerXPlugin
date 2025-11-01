# Research: Base Plugin Migration

## Phase 0 Discovery Summary

- 目标：梳理 `com.powerx.plugin.base` 中与 Templates CRUD 相关的响应结构、错误码、租户上下文约定，为 Phase 1/2 的框架与 Skeleton 实现提供基准。
- 资料来源：`com.powerx.plugin.base/backend/internal/transport/http/admin/templates/**`、`internal/services/admin/templates/**`、`internal/domain/repository/template/**`、`plugin.yaml`、`web-admin/app/composables/api/useTemplate.ts`。

## Response Envelope 观测

- 标准返回格式：`{ success: boolean, data?: any, error?: { code, message, details? }, message?: string, timestamp: RFC3339, request_id?: string }`。
- 模板 CRUD 接口错误码示例：
  - 请求参数错误 → `ErrCodeInvalidRequest`
  - 数据不存在 → `ErrCodeNotFound`
  - 权限不足 / 鉴权失败 → `ErrCodeUnauthorized`
  - 内部错误 → `ErrCodeInternalError`
- HTTP 状态码与错误码的映射需在框架响应助手中保持可扩展性（错误码通过常量定义，HTTP 状态由 handler 控制）。

## 租户上下文与数据类型

- `X-Tenant-ID` Header 为主要租户来源，类型为 **uint64**（Base 插件中通过 `strconv.ParseUint` 处理）。
- Repository 层要求：
  - 内嵌 `repository.BaseRepository[T]`
  - 在查询/更新时追加 `tenant_id = ?` 条件
  - 在事务或连接初始化时调用 `SET LOCAL app.tenant_id`
- Skeleton 内存实现需持续保存 `tenant_id`（可通过 `map[uint64][]Template` 或在记录上保留原字段），并在无 Header 时回退至 Standalone 默认租户 `1`。

## 组件复用清单

- 前端页面：`intro.vue`、`templates/index.vue`、`templates/crud.vue`
- 公共组件：`TemplateFormModal.vue`、`ConfirmDialog.vue`、`ToastAlert.vue`
- Composable：`useTemplateApi`（主要引用 `$fetch` 和 `X-Tenant-ID` 逻辑）

## 未决事项 / 待确认

- Base 插件是否存在特殊的错误码扩展（例如安全、配额相关），若需要在 Skeleton 中预留位置需继续盘点。
- CLI 模板中 starterPages 的默认开关策略（是否允许通过参数关闭）。

> 后续任务：T001/T002/T026/T030 将补充具体示例、执行验证步骤与输出差异记录。
