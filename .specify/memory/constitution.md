# PowerXPlugin 宪章

## 核心原则

### I. 双重使命仓库
PowerXPlugin 同时提供可运行的脚手架骨架和可复用的框架；任何改动都必须保持脚手架产出与框架包一致，让下游插件延续统一体验。

### II. 契约优先兼容性
Manifest、RBAC、健康检查与 API 契约是唯一真相：相关 Schema 必须存放于 `docs/contracts/**`，生成 OpenAPI/JSON Schema 工件，并在上线前通过代码生成或运行时校验被实现端消费。

### III. Go + Nuxt 基线
默认技术栈为 Go（Gin）+ Nuxt；仓库依赖 `go.work` 管理离散模块（如 `framework/`、`tools/cli/`），并在 `sdk/workspace/` 下维护 npm workspace，锁定依赖并输出 `@powerx-plugin/framework-*` 套件。

### IV. 脚手架与 CLI 纪律
`px-plugin` CLI 模板具有最高约束力：必须渲染当前骨架（后端 `go run ./cmd/plugin`、管理端 `npm run dev` 可直接运行），并清晰标注实验参数、占位实现（如 AuthGuard）及暂不支持的流程。

### V. 透明交付与一致性
文档、TODO 状态与发布记录必须真实反映实现情况；CI 强制执行 Go lint/test 与 `npm run build`，每个阶段推进都需满足 `docs/init-project.md` 的检查清单或明确记录延期。

## 实施约束

- 维持根目录 `go.work` 的多模块结构，确保 `framework/` 与 `tools/cli/` 可独立构建，并通过 `github.com/powerx-plugin/framework/...` 暴露导入路径。
- 前端产物必须位于 `sdk/workspace/` 下的 npm workspace，锁定依赖版本并发布 `@powerx-plugin/framework-admin`、`@powerx-plugin/framework-client`，让 Nuxt 项目可直接安装使用。
- 脚手架模板需提供可运行默认项：后端串联 `bootstrap`、`router` 与 Manifest 注册；前端提供 Nuxt 布局层、导航壳与可覆写的 API 助手。
- `plugin.yaml` 元数据、脚手架模板（`scaffold/templates/**`）与 CLI 命令（`init` 及计划中的 `package/dist/publish`）必须保持一致，并为未实现命令标注“设计稿”状态。
- 共用中间件（如 AuthGuard 占位）、可观测性钩子与契约适配器需沉淀于 `framework/` 内，以便插件项目在其之上扩展而非复制。

## 开发流程与质量闸门

- 严格遵循分阶段路线：先完成仓库地基（Phase 0），再推进协议沉淀（Phase 1）、骨架抽取（Phase 2）、框架拆分（Phase 3）、CLI/模板扩展（Phase 4），最终通过生成示例验收（Phase 5）。
- 契约变更视为上线闸门——更新 Schema 或 OpenAPI 时，必须同步提交代码生成/校验及文档改动。
- 新增 CLI 能力或模板调整，需通过 `px-plugin init <plugin-id>` 实际生成验证，并在 `examples/` 中记录，审查时比对基准插件差异。
- CI 必须覆盖 Go lint/test 与前端构建；除非在 `docs/init-project.md` 中明示临时豁免，否则禁止手动合并未通过检查的变更。
- 任何占位或实验功能都要附带 TODO 与路线图关联，提醒使用者成熟度与风险。

## VI. 插件项目产出规范

PowerXPlugin 脚手架产出的插件项目必须遵循以下基本约束，以确保下游插件延续统一架构与体验：

### 目录分层
```
backend/
└── internal/
    ├── transport/http/{admin,agent,...}/<domain>/
    ├── services/{admin,agent,...}/<domain>/
    └── domain/{models,repository}/<domain>/
```
- 使用 lower_snake_case 作为目录名
- Handler 仅负责入参校验→鉴权→调用 Service→序列化
- 业务编排逻辑必须在 `internal/services` 中完成
- HTTP 与 gRPC 复用同一 Service 层

### Repository 层约束
- **必须内嵌** `repository.BaseRepository[T]` 并提供 `NewXXXRepository` 构造函数
- **禁止直接暴露裸 `*gorm.DB` 字段**（维持读写封装一致性）
- 所有读写操作须在租户上下文中执行：`BeginTenantTx` → `SET LOCAL app.tenant_id`
- 参考完整规范：`com.powerx.plugin.base/.specify/memory/rulesets/crud/repository.yaml`

### 统一响应格式
```json
{
  "success": true,
  "data": {},
  "message": "",
  "error": null,
  "timestamp": "2024-12-09T12:00:00Z",
  "request_id": "rq-123"
}
```
- 分页响应：`data` 字段内返回 `{ "items": [...], "total": 135, "page": 1, "page_size": 20 }`
- 错误响应：`error` 字段包含 `{ "code": "ERROR_CODE", "message": "...", "details": {} }`

### 中间件栈顺序
1. `request_id` — 生成/透传请求 ID
2. `ctx_verify` — JWT/HMAC 验签，抽取 `tenant_id/user_id/permissions`
3. `rbac_guard` — 服务端权限判定
4. `tenant_ctx` — 设置 DB 会话变量（RLS 支持）
5. `recovery/logging` — 统一结构化日志与 panic 保护

> 完整插件开发规范、DTO 校验、测试策略等详细内容，参见：
> - `com.powerx.plugin.base/.specify/memory/constitution.md`
> - `com.powerx.plugin.base/.specify/memory/rulesets/`

## 治理

本宪章优先于既有实践；任何修订都必须经架构评审，更新路线图条目，提供现有插件的迁移指引，并同步更新 `docs/` 文档。评审需核验是否遵循上述原则、约束与流程闸门。

**版本**: 0.1.0 | **批准日期**: 2025-10-29 | **最后修订**: 2025-11-01
