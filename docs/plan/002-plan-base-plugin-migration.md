# com.powerx.plugins.base 迁移方案（2025-Q4 调整版）

## 1. 目标
- 在 `PowerXPlugin` 仓库沉淀一套可直接运行的 Base 模板：后端 CRUD、前端 Starter 页面、`plugin.yaml`/菜单/RBAC 清单。
- 将公共能力抽象到 `framework/`（Go runtime + Nuxt Layer），为 `px-plugin init` 输出一致的骨架。
- 严格遵守《docs/init-project.md》中 Phase 2~4 的约束（Go + Nuxt、无持久层、Skeleton/框架分离、模板可渲染）。

## 2. 可直接迁移的能力（✅ 高度可行）
- **后端分层模式**：`routes/handler/service/repo/model` 的切分与当前 Skeleton 规划一致，目录可一一映射到 `skeleton/backend/go-gin/internal/**`。
- **目录约定**：沿用 `internal/transport/http`（Handler）、`internal/services`（业务编排）、`internal/entity/repository`（仓储）、`internal/entity/models`（数据模型）等路径，保持与 Base 插件一致的分层命名，便于 CLI 与文档互相校验。
- **前端页面与组件**：`intro.vue`、`templates/*.vue`、`TemplateFormModal.vue`、`ConfirmDialog.vue` 均可直接复用，路由结构与 Layer 预期保持一致。
- **API 客户端**：`useTemplateApi` 及 `_client.ts` 的职责与 `@artisan-cloud/plugin-framework-client` 相符，可迁移并扩展现有客户端能力。
- **启动流程**：相较 Base 插件复杂的 `main.go`，PowerXPlugin 的轻量启动更适合作为模板，可复用现有 Standalone 模式。
- **规范引用**：迁移后的 Repository/Service 层需继续遵循 `.specify/memory/constitution.md` 中的约束——包括内嵌 `repository.BaseRepository[T]`、提供 `NewXXXRepository` 构造函数、在事务中设置 `app.tenant_uuid` 以及禁止直接暴露裸 `*gorm.DB`。

## 3. 必须先完成的改造（⚠️ 阻塞项）
1. **Router Path Param 支持**：`framework/backend/go/router/router.go:140` 的 `Context.Param` 目前返回空，需实现路径参数解析，否则 `/templates/:id` 无法工作。
2. **数据存储层替换**：Base 插件依赖 GORM + 数据库；需改造成符合 `.specify/memory/constitution.md` 的内存仓储（map 或 `sync.Map`），同时保持 Repository 结构内嵌 `BaseRepository[T]` 并完整实现租户隔离规范。
3. **统一响应格式**：Base 使用 `{success, data, message, error, timestamp, request_id}`；框架需提供同结构的 JSON 响应助手，供 Skeleton 与未来模板共享。
4. **中间件简化**：`EnsureTenant`、`RequestID` 等中间件需要 Stub 化（读取 `tenant_uuid` 或 Standalone 默认 1），同时记录鉴权仍为 501 的限制。

## 4. 实施里程碑

### 阶段 1：框架增强（前置必须完成）
- 实现 Router 路径参数解析、Query/Body 绑定，并扩展 `bootstrap.Context`（`Param` / `Query` / `BindJSON` / `JSON`）。
- 输出统一响应助手（建议 `framework/backend/go/router/response.go`），兼容 `{success, data, message, error, timestamp, request_id}` 并默认注入请求 ID。
- 添加轻量 RequestID 与 Tenant Context 中间件（默认从 Header 读取，Fallback 到 `Standalone` 默认值）。
- 为 `@artisan-cloud/plugin-framework-client` 补充 `put/delete` 等基础方法并支持透传 Tenant header，确保包内仅包含通用 HTTP 基础设施；同步在 `framework-admin` Layer 内引入 Starter 配置开关。

### 阶段 2：Skeleton 后端模板
- 在 `skeleton/backend/go-gin/internal/templates` 实现内存版本的 `model/service/repository/handler`，Repository 必须内嵌 `repository.BaseRepository[Template]`、提供 `NewTemplateRepository` 构造函数，并在 `BeginTenantTx` 中显式执行 `SET LOCAL app.tenant_uuid`（内存实现可通过上下文记录/校验该值，保持接口与调用顺序与数据库版本完全一致）。
- 更新 `skeleton/backend/go-gin/internal/routes/routes.go`，新增 `/templates` CRUD，保留 `ping`。
- Service 层保持原先职责：HTTP Handler 只负责校验/鉴权/序列化，所有业务编排封装在 `internal/services/templates`，并为未来的 HTTP/gRPC 复用保留同一 Service 实例。
- 在 `manifestx/manifest.go` 声明菜单与 `base:template:*` 权限，添加示例租户。
- 更新 `skeleton/backend/go-gin/README.md`，解释内存存储、环境变量及未来持久层扩展路径。

### 阶段 3：前端迁移与框架提炼
- 将 `intro.vue`、`templates/index.vue`、`templates/crud.vue` 及组件迁移到 `skeleton/web-admin/nuxt/app`，去除仅针对宿主的桥接逻辑。
- Skeleton 在 `app/composables/api/useTemplateApi.ts` 中提供示例封装，内部调用 `usePluginApi`，作为业务层如何复用 framework-client 的参考；该文件仅做样例，CLI 生成项目可按需改写。
- 保持 `@artisan-cloud/plugin-framework-client` 仅输出通用 HTTP 封装（`get/post/put/delete`），不承载任何业务 API；框架更新后 Skeleton 与插件项目基于此自行封装。
- 在 `framework-admin` Layer 提供 `starterPages` 选项，自动注册 Starter 菜单、页面与必要的全局组件；Skeleton 默认启用。
- 整理 `i18n` 文案，保持 `zh/en` 最小集合，并与 Manifest 菜单键值对应。

### 阶段 4：CLI 模板与交付
- 将 Skeleton 结构渲染成 `scaffold/templates/backend/go-gin/**` 与 `scaffold/templates/web-admin/nuxt/**`，占位符化插件 ID/名称/菜单。
- 更新 CLI 文档及实现，确保 `px-plugin init` 能生成与 Skeleton 一致的 CRUD 示例。
- 在 `docs/init-project.md`、`docs/guide/quickstart.md` 等文档同步新的模板能力与验证步骤。

## 5. 风险与限制
- **功能降级**：数据库降级为内存实现、租户隔离仅为 Stub、权限校验依旧返回 501；需在文档显式标注。
- **兼容性**：Router 参数解析与 Gin 行为可能存在差异，需在测试中覆盖常见场景；响应格式调整需确保前端解析兼容。
- **维护成本**：未来需同时维护框架 Stub 与真实实现，建议在 CHANGELOG/ADR 中持续同步差异。

## 6. 验证清单
- `go test ./skeleton/...`、`cd framework/backend/go && go test ./...`
- `npm install && npm run lint`（`skeleton/web-admin/nuxt` 与 `framework/frontend/nuxt/framework-admin`）
- Standalone 自测：`go run ./skeleton/backend/go-gin/cmd/plugin` + `npm run dev`，浏览器验证 `/intro`、`/templates/crud` CRUD 流程。
- CLI 输出验证（准备就绪后）：`px-plugin init com.powerx.demo --ui=starter`，确保产物可直接运行。

## 7. 结论与优先级建议
- 迁移总体**可行**，与 PowerXPlugin 设计目标高度一致。
- **优先完成阶段 1（Router & 响应 & 中间件）**，这是 Templates 模块迁移的唯一硬性前置。
- 阶段 2 可选 Templates 模块验证迁移流程；阶段 3/4 聚焦框架抽象与 CLI 交付。
- 完成后可为团队提供标准化 CRUD Skeleton，加速新插件启动，同时保留向真实生产实现升级的明确路径。

---

## 8. 2025-11-02 更新：差异快照与补充行动

| 模块 | Base 插件 | PowerXPlugin 当前状态 | 行动项 |
| --- | --- | --- | --- |
| Router / Middleware | Gin + 自定义 | 框架 Param/Response/Tenant 已实现 | ✅ 完成 |
| Templates 后端 | GORM + 租户隔离 | 内存 map + BaseRepository Stub | 补齐 DTO/错误码/服务编排 |
| Nuxt 配置 | 完整 baseURL/runtimeConfig/Nitro | Skeleton 已复制主要常量，语言包路径调整 | ⚠️ 校准 `langDir`、`POWERX_PROXY`、代理、headers |
| 前端页面/组件 | intro、templates、bridge-dev 等 | intro + templates + 首页，新组件就绪 | ⚠️ 未迁 Bridge/Stores，需列差异或规划 |
| CLI 模板 | 与 Base 对齐 | 结构已同步，但配置/依赖待更新 | ⚠️ 更新 nuxt.config、package, docs |
| 文档 | README/Quickstart/Standalone | Quickstart/Standalone 已补充 CRUD | ⚠️ 需追加差异说明与路线 |

---

## 9. 2025-11-02 更新：阶段计划修订

- **Phase 0**：完成基线调研及 Clarifications。
- **Phase 1**：框架增强已交付并达到覆盖率要求。
- **Phase 2**：需要第二轮补强（DTO、错误码、README 升级路径）。
- **Phase 3**：Nuxt 配置对齐 Base（`compatibilityDate`、`ssr`、`baseURL`、`runtimeConfig`、Nitro headers、代理、`@nuxt/icon`、`@pinia/nuxt`、`@nuxtjs/color-mode`）；语言包目录统一为 `i18n/locales`（Base 位于 `app/locales`，Skeleton 因 `srcDir: 'app'` 需通过 `langDir: '../i18n/locales'` 以兼容 CLI 模板与 Standalone），Bridge/Stores 评估是否迁移。
- **Phase 4**：CLI 模板伴随 Skeleton 同步，`px-plugin init` 生成物需开箱即用。
- **Phase 5**：文档更新（Quickstart、Standalone、Plan、Spec、Tasks）。
- **Phase 6（新增）**：未迁功能评估与路线（Bridge、Operations、Dev Console、Tailwind4 等），给出优先级与排期建议。

> Phase 6 评估维度：
> 1. **使用频率**（高频能力优先迁移）
> 2. **实现复杂度**（低复杂度优先落地）
> 3. **依赖关系**（独立模块优先处理）
>
> 检查清单：Bridge 调试（`app/bridge/`）、Pinia Stores（`operations/`、`dev-console/`）、Tailwind v4 升级、运营面板、合规/安全工具等。评估结论需记录在 `research.md` 与 README/Plan 中，并明确「迁移 / Stub / 暂不支持」策略。

---

## 10. 风险与对策（补充）

| 风险 | 描述 | 应对策略 |
| --- | --- | --- |
| 框架回归 | Router/Middleware 变更影响现有用户 | 全量单测 + `cd framework/backend/go && go test ./...` 作为必跑检查 |
| Nuxt 配置偏差 | Skeleton/CLI 与 Base 差异导致路由/i18n 异常 | 维护 `research.md` 差异记录，完成 Phase 7 T039/T040 |
| CLI 模板偏离 | Skeleton 与模板不同步 | `px-plugin init` smoke + diff，必要时在 CI 加自动对比 |
| 文档滞后 | 新能力未在 Quickstart/Standalone 体现 | 将文档更新纳入任务检查清单 |
| 未迁 Bridge/Stores | Base 的扩展功能暂未示例 | Phase 6 输出 roadmap 或标明不在模板范围 |

---

## 11. 验证清单（更新）

1. `cd framework/backend/go && go test ./... -coverprofile=coverage.out`（≥90%，SC-001）
2. `go test ./skeleton/backend/go-gin/... -v` + `curl` 多租户 CRUD（SC-002）
3. `curl -w 'time_total:%{time_total}'` 记录延迟；数据写入 `research.md`
4. `npm run lint --silent` / `npm run build`（skeleton/web-admin/nuxt、workspace layer）
5. Skeleton 前端手动验证 `/`、`/_p/{pluginId}/admin/templates/crud`；`POWERX_PROXY=1` 场景验证 baseURL/代理
6. `./bin/px-plugin init <plugin-id>` 后执行 `go test ./...`、`npm run lint --silent`，手动验证页面
7. 文档（Quickstart、Standalone、Plan、Spec、Tasks）同步更新
8. Phase 6 差异清单在 README/Plan 中明确

---

## 12. 当前状态（2025-11-02）

- Phase 1 完成；Phase 2 需补强 DTO/错误码。
- Phase 3/4 正在推进：Nuxt 配置已对齐核心常量，语言包路径统一为 `i18n/locales`，并在文档中说明与 Base (`app/locales`) 的差异及原因。
- CLI 模板和依赖正在同步；`px-plugin init` Smoke 已执行，后续要补充新依赖文档。
- 下一步：
  1. 完成 Phase 2 补强（T037/T038）。
  2. Phase 3 对齐 Nuxt 配置并在 `research.md` 写明差异（T039/T040）。
  3. 更新 CLI 文档与 README（T041），并输出未迁能力清单（T042）。

## 13. 依赖检查清单

- [ ] `framework-admin` Layer 可用，StarterPages toggle 生效
- [ ] `@artisan-cloud/plugin-framework-client` 暴露 `get/post/put/delete` 并透传 `tenant_uuid`
- [ ] Skeleton 后端可独立运行并完成 CRUD Smoke（Phase 2 验证）
- [ ] Skeleton 前端（含 `POWERX_PROXY=1` 场景）可访问首页与 Admin CRUD 页面

---

## 14. Phase 7 收尾（2026-03-25）

### 14.1 已完成

1. DTO/错误码统一：Templates Handler 使用统一校验错误输出（`VALIDATION_FAILED + details.field/template_code`）。
2. 模板同步：同一改动已同步到 `scaffold/templates` 与 `tools/cli/internal/templates/data`。
3. Nuxt 配置三方一致：Skeleton/CLI/scaffold 在 `runtimeConfig`、Nitro headers、HMR/代理、i18n `langDir` 上一致。
4. CLI 文档补齐：新增 Web Admin 关键环境变量与 Standalone/Proxy 启动说明。

### 14.2 未迁移能力与路线

1. Bridge 完整通信链路：先落最小 stub（P2）。
2. Pinia 业务 store 示例：补 1-2 个可复用样例（P2）。
3. Tailwind4 深度流水线：独立评估，不阻塞 Starter（P3）。
4. Dev Console/Operations 高阶模块：分期迁移，维持最小 starter 负担（P3）。
