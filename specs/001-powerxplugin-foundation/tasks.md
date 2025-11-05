# Tasks: PowerXPlugin 仓库基线落地

**Input**: 设计文档位于 `/specs/001-powerxplugin-foundation/`
**Prerequisites**: `plan.md`、`spec.md`、`research.md`、`data-model.md`、`contracts/`

> 所有任务均按用户故事组织，确保每个增量可独立实现与验证。

## Phase 1: Setup（通用初始化）

**目的**: 建立仓库级基础配置，为多模块结构奠定基础。

- [X] T001 创建 Go workspace 根文件 `go.work`，注册 `./framework` 与 `./tools/cli`
- [X] T002 配置根级 `package.json` 以声明 npm workspaces（指向 `framework/frontend/nuxt/*`）
- [X] T003 撰写仓库根 `README.md`，引用 quickstart 与技术设计文档

---

## Phase 2: Foundational（阻塞性前置工作）

**目的**: 准备所有用户故事共享的核心目录与模块骨架；未完成前禁止进入任一用户故事。

- [X] T004 建立 `framework/backend/go/` 目录骨架（bootstrap/router/middleware 等子目录）
- [X] T005 初始化 `framework/go.mod` 并声明模块 `github.com/powerx-plugin/framework`
- [X] T006 初始化 `tools/cli/go.mod` 与 `tools/cli/cmd/` 目录（预留 init/package/dist/publish 命令）
- [X] T007 配置 `framework/frontend/nuxt` 下的 package.json 与目录结构
- [X] T008 创建 `skeleton/backend` 与 `skeleton/web-admin` 目录骨架（含 `cmd/plugin`、`internal/`、`app/`）
- [X] T009 建立 `scaffold/templates/backend/go-gin` 与 `scaffold/templates/web-admin/nuxt` 目录占位
- [X] T010 添加入仓 `config/config.yaml.example` 作为环境配置模板

**Checkpoint**: 目录骨架与多模块配置完成，可开始用户故事实现。

---

## Phase 3: 用户故事 1 - 可运行骨架与框架同步 (Priority: P1) 🎯 MVP

**Goal**: skeleton 后端/前端可直接运行；同时输出与 skeleton 对齐的 Go 框架与 Nuxt Layer，供外部插件引用。

**Independent Test**: `go run ./skeleton/backend/cmd/plugin` 返回 `GET /api/v1/ping`=200；在独立项目引用 `github.com/powerx-plugin/framework` 与 `@powerx-plugin/framework-admin` 可成功构建/启动。

### 实施任务

- [X] T011 [US1] 实现 `framework/backend/go/bootstrap/app.go`，定义 `App`/`Config` 结构体与 `NewAppFromEnv()`、`Run()`、`Shutdown()` 方法
- [X] T012 [P] [US1] 补充 `framework/backend/go/router/router.go` 与 `router.go`，导出 `RegisterFrameworkRoutes/RegisterPluginRoutes`
- [X] T013 [P] [US1] 定义 `framework/backend/go/manifest/manifest.go` 与类型 `Menu`、`Permission`
- [X] T014 [P] [US1] 定义 `framework/backend/go/rbac/rbac.go`，暴露角色与权限报告 API
- [X] T015 [P] [US1] 添加 `framework/backend/go/middleware/auth_guard.go` stub，默认返回 `501 Not Implemented`
- [X] T016 [P] [US1] 实现 `framework/backend/go/observability/metrics.go` 与 `tracing.go` 占位
- [X] T017 [US1] 创建 `skeleton/backend/go.mod`，引用 `github.com/powerx-plugin/framework`
- [X] T018 [US1] 编写 `skeleton/backend/cmd/plugin/main.go`，按六步装配流程注册框架与业务路由
- [X] T019 [P] [US1] 实现 `skeleton/backend/internal/routes/routes.go`，提供 `GET /api/v1/ping`
- [X] T020 [P] [US1] 实现 `skeleton/backend/internal/service/ping.go` 与 `handler/ping.go`
- [X] T021 [US1] 编写 `skeleton/backend/README.md`，记录 Go 1.24+ 依赖与启动命令
- [X] T022 [US1] 配置 `skeleton/web-admin/nuxt.config.ts`，引入 `definePowerXAdminConfig`
- [X] T023 [P] [US1] 创建 `skeleton/web-admin/app/pages/_p/com.powerx.sample/admin/index.vue` Starter 页面
- [X] T024 [P] [US1] 添加 `skeleton/web-admin/app/components/powerx/PXNav.vue` 覆盖示例
- [X] T025 [US1] 实现 `framework/frontend/nuxt/framework-admin/index.ts` 与 Layer `nuxt.config.ts`
- [X] T026 [P] [US1] 实现 `framework/frontend/nuxt/framework-client/api.ts` 与 `$fetch` 包装
- [X] T027 [US1] 验证 `go run ./skeleton/backend/cmd/plugin` 启动成功并对 `GET /api/v1/ping` 返回 200
- [X] T028 [US1] 验证 `cd skeleton/web-admin && npm run dev` 可访问 Starter 页面并加载 PX 布局
- [X] T029 [US1] 在 `examples/verify-external/` 构建临时插件工程，引用框架包并执行 `go build` / `npm run build` 以验证外部编译通过

**Checkpoint**: Skeleton + Framework + Nuxt Layer 均可运行，满足外部引用需求。

---

## Phase 4: 用户故事 2 - 契约驱动的跨层一致性 (Priority: P1)

**Goal**: Manifest/RBAC/健康检查/API 契约以 Schema 形式沉淀，CI 自动生成文档并在框架/SDK 中校验。

**Independent Test**: 修改任一契约文件后运行 CI，可看到文档更新；若实现未同步契约，将在构建阶段失败并提示修复。

### 实施任务

- [X] T030 [US2] 填充 `docs/contracts/manifest.json`，对齐 Schema 约束与示例
- [X] T031 [P] [US2] 填充 `docs/contracts/rbac.json`，列出默认角色与权限键示例
- [X] T032 [P] [US2] 更新 `docs/contracts/openapi.yaml`，记录 `/api/v1/ping` 与 `/api/v1/admin/manifest`
- [X] T033 [US2] 在 `framework/backend/go/manifest/validator.go` 集成 Manifest Schema 校验
- [X] T034 [P] [US2] 在 `framework/backend/go/rbac/validator.go` 集成 RBAC Schema 校验
- [X] T035 [US2] 在 `framework/frontend/nuxt/framework-client/api.ts` 引入契约驱动的错误提示
- [X] T036 [US2] 创建 `.github/workflows/ci.yml`，包含 `go test ./...`、`npm ci && npm run lint && npm run build`、契约 Schema 校验与文档生成扫描
- [X] T037 [P] [US2] 撰写 `docs/contracts/README.md`，说明 Schema 更新流程与 CI 钩子

**Checkpoint**: CI 与运行时均受契约约束，防止实现偏离协议。

---

## Phase 5: 用户故事 3 - CLI 脚手架与模板交付 (Priority: P2)

**Goal**: `px-plugin init` 可生成与 skeleton 对齐的工程，并在 CLI 中标注 `package/dist/publish` 的实验状态。

**Independent Test**: 在空目录执行 `px-plugin init com.demo.todo`，生成项目无需手工改动即可运行；调用 `px-plugin package` 返回实验性提示。

### 实施任务

- [X] T038 [US3] 创建 `scaffold/templates/backend/go-gin/cmd/plugin/main.go.tmpl` 与变量占位
- [X] T039 [P] [US3] 创建 `scaffold/templates/backend/go-gin/internal/routes.go.tmpl`、`handler/ping.go.tmpl`
- [X] T040 [P] [US3] 创建 `scaffold/templates/web-admin/nuxt/nuxt.config.ts.tmpl` 与 Starter 页面模板
- [X] T041 [P] [US3] 创建 `scaffold/templates/web-admin/nuxt/app/pages/_p/__plugin__/admin/index.vue.tmpl`
- [X] T042 [US3] 实现 `tools/cli/internal/templates/embed.go`，使用 `go:embed` 打包模板
- [X] T043 [US3] 实现 `tools/cli/cmd/init.go`，渲染模板并写入 `plugin.yaml`
- [X] T044 [P] [US3] 实现 `tools/cli/internal/contracts/embed.go`，内置 Manifest/RBAC/OpenAPI 元数据
- [X] T045 [US3] 实现 `tools/cli/cmd/package.go`（experimental 提示 + 构建占位）
- [X] T046 [P] [US3] 实现 `tools/cli/cmd/dist.go`（experimental 提示 + 打包占位）
- [X] T047 [P] [US3] 实现 `tools/cli/cmd/publish.go`（experimental 提示 + 上传占位）
- [X] T048 [US3] 更新 `tools/cli/README.md`，记录离线渲染、实验命令、使用示例

**Checkpoint**: CLI 输出与 skeleton/模板一致，离线可运行，实验命令有清晰标记。

---

## Phase 6: 用户故事 4 - 版本发布与示例验证 (Priority: P3)

**Goal**: 建立发布闭环，并提供 `examples/starter` 作为 CLI 生成物基准。

**Independent Test**: 运行 Release 流水线生成 Go Module Tag、npm 包与 CLI 二进制；对比 `examples/starter` 与 CLI 当次生成无差异。

### 实施任务

- [X] T049 [US4] 使用 CLI 生成新项目并固化到 `examples/starter/`
- [X] T050 [US4] 创建 `.github/workflows/release.yml`，串联 `px-plugin package/dist/publish`
- [X] T051 [P] [US4] 撰写 `docs/release.md`，描述版本化与回滚流程
- [X] T052 [P] [US4] 更新 `CHANGELOG.md` 模板，记录 CLI 与框架同步发布
- [X] T053 [US4] 更新 `docs/backlog/multi-language.md`，标注多语言支持的 TODO 与依赖

**Checkpoint**: 发布流程可执行，示例项目可验证 CLI 产物。

---

## Phase 7: Polish & Cross-Cutting

**目的**: 全局文档、验证与收尾工作，提升整体可维护性。

- [X] T054 [P] 整理 `docs/quickstart.md`，同步 quickstart 手册至仓库文档
- [X] T055 运行一次 `go test ./...`、`npm run lint`、`npm run build`、`px-plugin init` 验证 Quickstart 步骤
- [X] T056 [P] 更新 `README.md` 与 `docs/init-project.md`，明确仅支持 Go + Nuxt 并指向 `docs/backlog/multi-language.md`
- [X] T057 [P] 为 `framework/backend/go` 编写核心单元测试（例如 `bootstrap/app_test.go`）并生成覆盖率报告
- [X] T058 [P] 为 `skeleton/backend` 编写集成测试（例如 `internal/routes/routes_test.go`）验证 `/api/v1/ping`
- [X] T059 [P] 在 `skeleton/web-admin/tests/e2e/` 添加前端 E2E 测试（Playwright/Cypress）覆盖 Starter 页面
- [X] T060 [P] 同步 `spec.md`、`plan.md`、`tasks.md` 版本号与状态标记

---

## Dependencies & Execution Order

1. **Phase 1 → Phase 2**：Setup 完成后才能搭建多模块骨架。  
2. **Phase 2 → User Stories**：所有用户故事依赖基础目录与模块就绪。  
3. **用户故事优先级**：US1、US2 属 P1，应在开始 US3、US4 前完成；US3、US4 可在 P1 完成后并行。  
4. **Phase 7**：在目标用户故事完成后统一收尾。

---

## Parallel Opportunities

- Setup/Foundational 中标记 `[P]` 的任务可并行（不同路径、无依赖）。  
- US1 中 `framework` 与 `skeleton` 子任务可分配给不同开发者并行推进。  
- US2 契约文件与校验器任务可拆分并行；CI 配置需在校验器完成后验证。  
- US3 模板文件生成可与 CLI 命令实现并行，但 `tools/cli/cmd/init.go` 依赖模板 embed。  
- US4 文档/流水线任务可在示例生成后并行完善。

---

## Implementation Strategy

1. **MVP（US1）**：完成 Setup → Foundational → US1，即可提供可运行骨架与基础框架；此时可演示初版插件。  
2. **契约护栏（US2）**：在 MVP 稳定后补齐 Schema 与 CI 校验，确保后续改动受约束。  
3. **CLI 交付（US3）**：当骨架与契约就绪，推进 CLI 模板与实验命令，支持外部团队快速起步。  
4. **发布闭环（US4）**：最后完善示例与发布流程，确保可向外部发布稳定版本。  
5. **Polish**：统一文档、跑通 quickstart，准备最终交付。
