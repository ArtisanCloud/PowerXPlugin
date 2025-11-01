# com.powerx.plugin.base 迁移方案（2025-Q4 调整版）

## 1. 目标
- 在 `PowerXPlugin` 仓库沉淀一套可直接运行的 Base 模板：后端 CRUD、前端 Starter 页面、`plugin.yaml`/菜单/RBAC 清单。
- 将公共能力抽象到 `framework/` 与 `sdk/workspace/`，为 `px-plugin init` 输出一致的骨架。
- 严格遵守《docs/init-project.md》中 Phase 2~4 的约束（Go + Nuxt、无持久层、Skeleton/框架分离、模板可渲染）。

## 2. 可直接迁移的能力（✅ 高度可行）
- **后端分层模式**：`routes/handler/service/repo/model` 的切分与当前 Skeleton 规划一致，目录可一一映射到 `skeleton/backend/internal/**`。
- **目录约定**：沿用 `internal/transport/http`（Handler）、`internal/services`（业务编排）、`internal/domain/repository`（仓储）、`internal/domain/models`（数据模型）等路径，保持与 Base 插件一致的分层命名，便于 CLI 与文档互相校验。
- **前端页面与组件**：`intro.vue`、`templates/*.vue`、`TemplateFormModal.vue`、`ConfirmDialog.vue` 均可直接复用，路由结构与 Layer 预期保持一致。
- **API 客户端**：`useTemplateApi` 及 `_client.ts` 的职责与 `@powerx-plugin/framework-client` 相符，可迁移并扩展现有客户端能力。
- **启动流程**：相较 Base 插件复杂的 `main.go`，PowerXPlugin 的轻量启动更适合作为模板，可复用现有 Standalone 模式。
- **规范引用**：迁移后的 Repository/Service 层需继续遵循 `.specify/memory/constitution.md` 中的约束——包括内嵌 `repository.BaseRepository[T]`、提供 `NewXXXRepository` 构造函数、在事务中设置 `app.tenant_id` 以及禁止直接暴露裸 `*gorm.DB`。

## 3. 必须先完成的改造（⚠️ 阻塞项）
1. **Router Path Param 支持**：`framework/backend/go/router/router.go:140` 的 `Context.Param` 目前返回空，需实现路径参数解析，否则 `/templates/:id` 无法工作。
2. **数据存储层替换**：Base 插件依赖 GORM + 数据库；需改造成符合 `.specify/memory/constitution.md` 的内存仓储（map 或 `sync.Map`），同时保持 Repository 结构内嵌 `BaseRepository[T]` 并完整实现租户隔离规范。
3. **统一响应格式**：Base 使用 `{success, data, message, error, timestamp, request_id}`；框架需提供同结构的 JSON 响应助手，供 Skeleton 与未来模板共享。
4. **中间件简化**：`EnsureTenant`、`RequestID` 等中间件需要 Stub 化（读取 `X-Tenant-ID` 或 Standalone 默认 1），同时记录鉴权仍为 501 的限制。

## 4. 实施里程碑

### 阶段 1：框架增强（前置必须完成）
- 实现 Router 路径参数解析、Query/Body 绑定，并扩展 `bootstrap.Context`（`Param` / `Query` / `BindJSON` / `JSON`）。
- 输出统一响应助手（建议 `framework/backend/go/router/response.go`），兼容 `{success, data, message, error, timestamp, request_id}` 并默认注入请求 ID。
- 添加轻量 RequestID 与 Tenant Context 中间件（默认从 Header 读取，Fallback 到 `Standalone` 默认值）。
- 为 `@powerx-plugin/framework-client` 补充 `put/delete` 等基础方法并支持透传 Tenant header，确保包内仅包含通用 HTTP 基础设施；同步在 `framework-admin` Layer 内引入 Starter 配置开关。

### 阶段 2：Skeleton 后端模板
- 在 `skeleton/backend/internal/templates` 实现内存版本的 `model/service/repository/handler`，Repository 必须内嵌 `repository.BaseRepository[Template]`、提供 `NewTemplateRepository` 构造函数，并在 `BeginTenantTx` 中显式执行 `SET LOCAL app.tenant_id`（内存实现可通过上下文记录/校验该值，保持接口与调用顺序与数据库版本完全一致）。
- 更新 `skeleton/backend/internal/routes/routes.go`，新增 `/templates` CRUD，保留 `ping`。
- Service 层保持原先职责：HTTP Handler 只负责校验/鉴权/序列化，所有业务编排封装在 `internal/services/templates`，并为未来的 HTTP/gRPC 复用保留同一 Service 实例。
- 在 `manifestx/manifest.go` 声明菜单与 `base:template:*` 权限，添加示例租户。
- 更新 `skeleton/backend/README.md`，解释内存存储、环境变量及未来持久层扩展路径。

### 阶段 3：前端迁移与框架提炼
- 将 `intro.vue`、`templates/index.vue`、`templates/crud.vue` 及组件迁移到 `skeleton/web-admin/app`，去除仅针对宿主的桥接逻辑。
- Skeleton 在 `app/composables/api/useTemplateApi.ts` 中提供示例封装，内部调用 `usePluginApi`，作为业务层如何复用 framework-client 的参考；该文件仅做样例，CLI 生成项目可按需改写。
- 保持 `@powerx-plugin/framework-client` 仅输出通用 HTTP 封装（`get/post/put/delete`），不承载任何业务 API；框架更新后 Skeleton 与插件项目基于此自行封装。
- 在 `framework-admin` Layer 提供 `starterPages` 选项，自动注册 Starter 菜单、页面与必要的全局组件；Skeleton 默认启用。
- 整理 `i18n` 文案，保持 `zh/en` 最小集合，并与 Manifest 菜单键值对应。

### 阶段 4：CLI 模板与交付
- 将 Skeleton 结构渲染成 `scaffold/templates/backend/go-gin/**` 与 `scaffold/templates/web/nuxt/**`，占位符化插件 ID/名称/菜单。
- 更新 CLI 文档及实现，确保 `px-plugin init` 能生成与 Skeleton 一致的 CRUD 示例。
- 在 `docs/init-project.md`、`docs/guide/quickstart.md` 等文档同步新的模板能力与验证步骤。

## 5. 风险与限制
- **功能降级**：数据库降级为内存实现、租户隔离仅为 Stub、权限校验依旧返回 501；需在文档显式标注。
- **兼容性**：Router 参数解析与 Gin 行为可能存在差异，需在测试中覆盖常见场景；响应格式调整需确保前端解析兼容。
- **维护成本**：未来需同时维护框架 Stub 与真实实现，建议在 CHANGELOG/ADR 中持续同步差异。

## 6. 验证清单
- `go test ./skeleton/...`、`go test ./framework/backend/go/...`
- `npm install && npm run lint`（`skeleton/web-admin` 与 `sdk/workspace/frontend/nuxt/framework-admin`）
- Standalone 自测：`go run ./skeleton/backend/cmd/plugin` + `npm run dev`，浏览器验证 `/intro`、`/templates/crud` CRUD 流程。
- CLI 输出验证（准备就绪后）：`px-plugin init com.powerx.demo --ui=starter`，确保产物可直接运行。

## 7. 结论与优先级建议
- 迁移总体**可行**，与 PowerXPlugin 设计目标高度一致。
- **优先完成阶段 1（Router & 响应 & 中间件）**，这是 Templates 模块迁移的唯一硬性前置。
- 阶段 2 可选 Templates 模块验证迁移流程；阶段 3/4 聚焦框架抽象与 CLI 交付。
- 完成后可为团队提供标准化 CRUD Skeleton，加速新插件启动，同时保留向真实生产实现升级的明确路径。
