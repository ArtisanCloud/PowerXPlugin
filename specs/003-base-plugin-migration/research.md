# Research: Base Plugin Migration

## Phase 0 Discovery Summary

- 目标：梳理 `com.powerx.plugin.base` 中与 Templates CRUD 相关的响应结构、错误码、租户上下文约定，为 Phase 1/2 的框架与 Skeleton 实现提供基准。
- 资料来源：`com.powerx.plugin.base/backend/internal/transport/http/admin/templates/**`、`internal/services/admin/templates/**`、`internal/domain/repository/template/**`、`plugin.yaml`、`web-admin/app/composables/api/useTemplate.ts`。

## Response Envelope 观测

### APIResponse 字段（源：`backend/internal/contracts/api.go`）

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `success` | bool | 请求是否成功 |
| `data` | any (可选) | 成功时返回的实体或分页对象 |
| `error` | `{ code, message, details? }` (可选) | 失败时的错误结构 |
| `message` | string (可选) | 附加的成功信息 |
| `timestamp` | RFC3339 | 服务器生成时间 | 
| `request_id` | string (可选) | 由 RequestID 中间件提供，用于日志关联 |

`APIError` 结构固定包含 `code` 与 `message`，并允许附带 `details` 说明（如字段验证信息）。

### 错误码清单（`backend/internal/contracts/api.go`）

- 通用：`INTERNAL_ERROR`、`INVALID_REQUEST`、`UNAUTHORIZED`、`FORBIDDEN`、`NOT_FOUND`、`CONFLICT`、`VALIDATION_FAILED`
- 模板 CRUD 实际使用：
  - 参数错误 / JSON 解析失败 → `INVALID_REQUEST`（HTTP 400）
  - 资源不存在 → `NOT_FOUND`（HTTP 404）
  - 缺少租户信息或权限不足 → `UNAUTHORIZED` / `FORBIDDEN`（HTTP 401/403）
  - 未处理异常 → `INTERNAL_ERROR`（HTTP 500）
- 其他领域扩展：`TENANT_NOT_FOUND`、`TENANT_MISMATCH`、`AGENT_TOOL_NOT_FOUND` 等（Skeleton 目前暂未使用，但需要保留扩展点）。

HTTP 状态码与错误码的映射在 handler 中明确调用 `contracts.Response*` 方法控制，框架响应助手需支持自定义 status + code 组合。

## 租户上下文与数据类型

- `X-Tenant-ID` Header 为主要租户来源，类型为 **uint64**（Base 插件中通过 `strconv.ParseUint` 处理）。
- Repository 层要求：
  - 内嵌 `repository.BaseRepository[T]`
  - 在查询/更新时追加 `tenant_id = ?` 条件
  - 在事务或连接初始化时调用 `SET LOCAL app.tenant_id`
- Skeleton 内存实现需持续保存 `tenant_id`（可通过 `map[uint64][]Template` 或在记录上保留原字段），并在无 Header 时回退至 Standalone 默认租户 `1`。

## Templates CRUD 请求 / 响应样例

### 路径与方法（源：`backend/internal/transport/http/admin/templates`）

| Method | Path | 说明 | 请求体 | 响应摘要 |
| --- | --- | --- | --- | --- |
| GET | `/api/v1/templates` | 查询模板列表，支持 `q`、`page`、`page_size` | n/a | `data` 为 `Page` 结构：`{"list": [...], "page_index": 1, "page_size": 20, "total": 1}` |
| GET | `/api/v1/templates/:id` | 获取单个模板 | n/a | `data` 为单个模板对象 |
| POST | `/api/v1/templates` | 创建模板 | `{ "name": "...", "description": "...", "content": "..." }` | 返回新建模板对象 |
| PUT | `/api/v1/templates/:id` | 更新模板 | 同 POST | 返回更新后的模板对象 |
| DELETE | `/api/v1/templates/:id` | 删除模板 | n/a | `data` 为 `{ "ok": true }` |

### 成功响应示例

```json
{
  "success": true,
  "data": {
    "list": [
      {
        "id": 42,
        "tenant_id": 100,
        "name": "Draft",
        "description": "Demo template",
        "content": "Hello",
        "created_at": "2025-10-31T12:34:56Z",
        "updated_at": "2025-10-31T12:34:56Z"
      }
    ],
    "page_index": 1,
    "page_size": 20,
    "total": 1
  },
  "timestamp": "2025-10-31T12:34:56.789Z",
  "request_id": "req-123"
}
```

### 常见错误场景

- `GET /api/v1/templates/abc` → HTTP 400 + `INVALID_REQUEST`（编号解析失败）
- `GET /api/v1/templates/1` 且非本租户 → HTTP 404 + `NOT_FOUND`
- 缺少 `X-Tenant-ID` → HTTP 401 + `"tenant context missing"`
- 未处理异常 → HTTP 500 + `INTERNAL_ERROR`

> 以上示例基于 handler 与 repository 行为推导；实现阶段应在测试中复现并记录实际响应。

## 组件复用清单

- 前端页面：`intro.vue`、`templates/index.vue`、`templates/crud.vue`
- 公共组件：`TemplateFormModal.vue`、`ConfirmDialog.vue`、`ToastAlert.vue`
- Composable：`useTemplateApi`（主要引用 `$fetch` 和 `X-Tenant-ID` 逻辑）

## 未决事项 / 待确认

- Base 插件是否存在特殊的错误码扩展（例如安全、配额相关），若需要在 Skeleton 中预留位置需继续盘点。
- CLI 模板中 starterPages 的默认开关策略（是否允许通过参数关闭）。
- BaseRepository 内存适配层：Skeleton 使用局部 map 存储即可，暂未发现需要在 framework 层抽象公共适配器的需求，后续若多项目共享可再单独拆出。

> 后续任务：T001/T002/T026/T030 将补充具体示例、执行验证步骤与输出差异记录。

## 覆盖率记录

- 2025-11-01：`go test ./framework/backend/go/... -coverprofile=framework/backend/go/coverage.out`；router 包覆盖率 93.8%，结果已留存。
