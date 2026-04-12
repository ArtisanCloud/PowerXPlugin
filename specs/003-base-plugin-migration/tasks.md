# Tasks: Base Plugin Migration

**Input**: Design documents from `/specs/003-base-plugin-migration/`
**Prerequisites**: plan.md (required), spec.md (required for user stories)

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Capture reference behaviours from `com.powerx.plugins.base` before implementation

- [x] T001 Extract response envelope schema与错误码说明到 specs/003-base-plugin-migration/research.md
- [x] T002 汇总模板 CRUD 请求/响应示例（含租户 ID 类型）到 specs/003-base-plugin-migration/research.md

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Prepare shared package structure用于后续各故事共用

- [x] T003 创建 skeleton/backend/go-gin/internal/templates/doc.go，记录与 .specify/memory/constitution.md 对齐的包注释
- [x] T004 建立 skeleton/web-admin/nuxt/app/pages/templates/.gitkeep 与 app/components/.gitkeep 以固定目录结构

**Checkpoint**: 基础目录与规范提示已就绪，可开始用户故事实现

---

## Phase 3: User Story 1 - Framework groundwork ready (Priority: P1) 🎯 MVP

**Goal**: 提供 Router/中间件/客户端基础设施，支持 CRUD 与租户上下文

**Independent Test**: `go test ./framework/backend/go/...` 通过且覆盖新路由、响应助手与中间件逻辑

### Implementation for User Story 1

- [x] T005 [US1] 扩展 bootstrap Context 接口以暴露 Param/Query/BindJSON 于 framework/backend/go/bootstrap/router_port.go
- [x] T006 [US1] 实现 Path 参数与 Query 解析逻辑于 framework/backend/go/router/router.go
- [x] T007 [US1] 新增 JSON Response 助手（含 timestamp/request_id）于 framework/backend/go/router/response.go
- [x] T008 [P] [US1] 实作 RequestID 中间件于 framework/backend/go/middleware/request_id.go
- [x] T009 [P] [US1] 实作 Tenant Context 中间件于 framework/backend/go/middleware/tenant_context.go
- [x] T010 [US1] 为 usePluginApi 补充 put/delete 能力于 framework/frontend/nuxt/framework-client/api.ts
- [x] T011 [US1] 编写 Router 与中间件单元测试于 framework/backend/go/router/router_test.go
- [x] T012 [US1] 更新框架示例或文档注释以引用新助手于 framework/backend/go/router/doc.go
- [x] T013 [US1] 评估并创建必要的 BaseRepository 内存适配层于 framework/backend/go/internal/memory_repository.go（如 Skeleton 需复用）
- [x] T014 [US1] 运行 `go test ./framework/backend/go/... -coverprofile=coverage.out` 并确认语句覆盖率 ≥90%，记录结果至 specs/003-base-plugin-migration/research.md

**Checkpoint**: Router/Middleware/Client 测试绿色，提供 CRUD 基础能力

---

## Phase 4: User Story 2 - Skeleton backend CRUD sample (Priority: P2)

**Goal**: Skeleton 后端提供内存版 Templates CRUD 并遵循租户隔离规则

**Independent Test**: `go test ./skeleton/backend/go-gin/...` & `curl` 验证不同租户 CRUD 返回 200/404

### Implementation for User Story 2

- [x] T015 [US2] 定义模板数据结构于 skeleton/backend/go-gin/internal/templates/model.go
- [x] T016 [US2] 实作内存 Repository（嵌入 BaseRepository）于 skeleton/backend/go-gin/internal/templates/repository.go
- [x] T017 [US2] 在 Repository 中实现 BeginTenantTx 并执行 `SET LOCAL app.tenant_uuid`（含相关注释）
- [x] T018 [US2] 实作 Templates Service 逻辑于 skeleton/backend/go-gin/internal/templates/service.go
- [x] T019 [US2] 编写 HTTP Handler 使用新框架上下文于 skeleton/backend/go-gin/internal/templates/handler.go
- [x] T020 [US2] 在 skeleton/backend/go-gin/internal/routes/routes.go 注册 CRUD 路由并执行种子数据初始化
- [x] T021 [P] [US2] 补充 Repository/Service 单元测试于 skeleton/backend/go-gin/internal/templates/service_test.go
- [x] T022 [P] [US2] 为缺失/非法租户 ID、非数字模板 ID 等边界场景编写测试于 skeleton/backend/go-gin/internal/templates/service_test.go
- [x] T023 [US2] 更新 skeleton/backend/go-gin/README.md 说明内存存储、租户隔离与 `SET LOCAL` 行为

**Checkpoint**: Skeleton 后端可独立提供多租户 CRUD 能力

---

## Phase 5: User Story 3 - Admin starter & CLI alignment (Priority: P3)

**Goal**: 前端与 CLI 模板展示 CRUD 页面并使用 framework-client

**Independent Test**: `npm run lint` (skeleton/web-admin/nuxt) 通过，`px-plugin init` 生成项目后可执行 CRUD 操作

### Implementation for User Story 3

- [x] T024 [US3] 迁移 Intro 页面至 skeleton/web-admin/nuxt/app/pages/intro.vue
- [x] T025 [US3] 迁移 Templates 列表与 CRUD 页面至 skeleton/web-admin/nuxt/app/pages/templates/
- [x] T026 [P] [US3] 迁移模态/确认/通知组件至 skeleton/web-admin/nuxt/app/components/
- [x] T027 [US3] 创建 useTemplateApi 示例至 skeleton/web-admin/nuxt/app/composables/api/useTemplateApi.ts
- [x] T028 [US3] 扩展 framework-admin Layer starterPages 逻辑于 framework/frontend/nuxt/framework-admin/layer/nuxt.config.ts
- [x] T029 [US3] 同步 scaffold/backend 与 scaffold/web 模板以输出新 Skeleton 于 scaffold/templates/
- [x] T030 [US3] 验证并记录 `px-plugin init` 生成流程于 specs/003-base-plugin-migration/research.md

**Checkpoint**: CLI 生成项目具备与 Skeleton 一致的前端 CRUD 体验

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 文档与最终验证

- [x] T031 [P] 更新 docs/guide/quickstart.md 加入 CRUD 验证步骤与延迟测试说明
- [x] T032 [P] 更新 docs/guides/develop/standalone-mode.md 说明内存存储与租户操作
- [x] T033 对 docs/plan/002-plan-base-plugin-migration.md 补记关键实施结果与偏差
- [x] T034 执行 go test ./...、npm run lint、px-plugin init 并记录结果到 specs/003-base-plugin-migration/research.md
- [x] T035 使用 `curl -w` 对不同租户执行 CRUD 测试并确认平均延迟 ≤1s，记录到 specs/003-base-plugin-migration/research.md
- [x] T036 整理最终交付总结至 CHANGELOG 或 README（如需要）

---

## Phase 7: Base Parity 差距收敛（新增）

**Purpose**: 对齐 Base 插件剩余能力，明确未迁部分与后续计划

- [x] T037 [US2] 迁移 Base DTO/错误码逻辑至 skeleton/backend/go-gin/internal/templates/dto.go，并更新 handler/service 使用
- [x] T038 [US2] 将 DTO/错误码改动同步到 scaffold/templates/backend 与 CLI 内嵌模板
- [x] T039 [US4] 对照 `com.powerx.plugins.base/web-admin/nuxt.config.ts`，补齐 Skeleton/CLI Nuxt 配置（runtimeConfig、Nitro headers、HMR/代理、devtools），并在 research.md 记录 diff
- [x] T040 [US4] 校正语言包路径与加载策略（`langDir`、目录结构），确保 Skeleton 与 CLI 在 Standalone/宿主双模式均能加载
- [x] T041 [US4] 更新 CLI README / docs，说明新增环境变量、依赖与运行方式
- [x] T042 [Doc] 汇总未迁移能力（Bridge、Stores、Tailwind4 等）及后续 roadmap，写入 docs/plan/002-plan-base-plugin-migration.md 与 research.md

---

## Dependencies & Execution Order

- Setup (Phase 1) → Foundational (Phase 2) → User Stories (Phase 3~5) → Polish (Phase 6) → Parity Closure (Phase 7)
- User Stories依规格优先级顺序：US1 → US2 → US3
- US2/US3 依赖 US1 的框架增强完成；US3 还需 US2 提供稳定后端以供 UI/CLI 验证；Phase 7 关注 Base 差距收敛，需建立在 Phase 2~5 的产物之上

### Parallel Opportunities

- T008/T009 可并行（不同中间件文件）
- US2 内 T015/T016 可与 T021 协同进行（模型 & 仓储实现 vs 测试编写）
- US3 内 T026 可与 T025 并行（组件 vs 页面）
- Polish 阶段 T031/T032 可并行更新文档

### Implementation Strategy

1. 完成 Setup/Foundational 以锁定规范与目录  
2. 聚焦 US1 提供可复用框架能力（MVP）  
3. 迁移后端 CRUD（US2），随后联动前端/CLI（US3）  
4. 最后执行文档与整体验证收尾  
