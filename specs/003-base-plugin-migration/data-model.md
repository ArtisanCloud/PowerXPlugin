# Data Model: Base Plugin Migration

## TemplateRecord

- **id** (uint64) — 自增主键，对应 `models.BaseModel.ID`
- **tenant_id** (uint64) — 必填，来自 `X-Tenant-ID`，用于多租户隔离
- **name** (string, ≤255) — 模板名称
- **description** (string, text) — 模板描述
- **content** (string, text) — 模板内容
- **created_at / updated_at** (time.Time) — 自动维护的时间戳

### 行为约束

- 所有查询/更新必须附带 `tenant_id = ?`
- 删除采用软删除（跟随 `gorm.DeletedAt` 约定）
- Skeleton 内存实现需保持同样字段，允许未来轻松切换数据库

## TenantContext

- **tenant_id** (uint64) — 中间件解析得到，如果缺失则在 Standalone 模式下默认 1
- **request_id** (string) — RequestID 中间件生成的标识，用于响应包与日志关联

### 作用范围

- 中间件写入 `context.Context`，Repository 与 Service 读取
- 未来可扩展 `user_id`、`roles` 等权限字段

## ResponseEnvelope

- **success** (bool)
- **data** (any, optional)
- **error** (struct，包含 `code`、`message`、`details?`)
- **message** (string, optional)
- **timestamp** (RFC3339 string)
- **request_id** (string, optional)

### 错误码枚举（与 Base 插件对齐）

- `ErrCodeInvalidRequest`
- `ErrCodeNotFound`
- `ErrCodeUnauthorized`
- `ErrCodeInternalError`

> 以上模型为 Phase 1/2 的实现基准；后续若扩展更多实体，请在此文件补充。
