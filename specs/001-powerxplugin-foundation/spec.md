# 功能规格：PowerXPlugin 仓库基线落地

**功能分支**: `001-powerxplugin-foundation`  
**创建日期**: 2025-10-29  
**状态**: Polish  
**输入**: 用户描述："Implement PowerXPlugin repository foundation covering multi-module Go structure, npm workspace, contracts, skeleton, CLI workflow per docs/init-project.md roadmap"

> 📘 **技术实现参考**  
> 本规格聚焦“做什么”。关于“如何实现”，请查阅：  
> - `docs/init-project.md`：整体架构、框架 API 签名、代码示例与交付路径  
> - `docs/init-project.md#powerxplugin-structure`：仓库目录结构与职责划分  
> - `docs/init-project.md#framework-signatures`：后端 bootstrap/router 契约与 Nuxt Layer 约束

## 用户场景与测试 *(必填)*

### 用户故事 1：可运行骨架与框架同步 (Priority: P1)

作为 PowerXPlugin 仓库维护者，我希望仓库既能直接运行 skeleton 后端与管理端，又能产出 `github.com/ArtisanCloud/PowerXPlugin/framework/...` 与 `@powerx-plugin/framework-*` 套件，这样外部插件能立即接入一致的默认能力。

**优先级原因**：骨架与框架的一致性是仓库价值核心，缺失会导致后续 CLI 与插件建设全部失效。

**独立验证**：在全新环境克隆仓库，安装依赖后单独验证 skeleton 后端 `go run` 与管理端 `npm run dev` 可启动，并在独立项目中引用框架包成功编译运行。

**验收场景**：

1. 若开发者在全新拉取的仓库中安装 Go/Node 依赖，当其于 `skeleton/backend` 执行 `go run ./cmd/plugin` 时，应成功启动服务并返回 `GET /api/v1/ping` 正常响应。
2. 若外部插件项目引用 `github.com/ArtisanCloud/PowerXPlugin/framework/router` 与 `@powerx-plugin/framework-admin`，当编译后端并启动 Nuxt 管理端时，应无编译错误且默认路由/布局可用。

---

### 用户故事 2：契约驱动的跨层一致性 (Priority: P1)

作为架构负责人，我需要 Manifest、RBAC、健康检查与 API 契约以 Schema 形式沉淀，并被框架与 SDK 自动校验，这样协议变化可控且不会出现实现与文档脱节。

**优先级原因**：契约违背会直接破坏宿主与插件互通，影响安全与运维，故与骨架同级优先。

**独立验证**：更新任意契约 Schema 后运行 CI，验证自动生成文档同步更新，框架/SDK 校验失败时能阻止合并。

**验收场景**：

1. 若开发者修改 `docs/contracts/manifest.json`，当运行仓库 CI 时，应生成最新可阅读文档并触发框架层兼容性校验。
2. 若 CLI 渲染出的插件缺失必需的 Manifest 字段，当执行框架启动流程时，应返回明确错误并提示依据 Schema 修复。

---

### 用户故事 3：CLI 脚手架与模板交付 (Priority: P2)

作为 CLI 使用者，我希望通过 `px-plugin init` 在空目录生成与仓库 skeleton 一致的工程，并能在同一 CLI 中查看后续打包发布命令的指引，这样团队能快速进入业务开发。

**优先级原因**：CLI 是外部团队获取骨架的入口，若缺失会引发重复搭建工作，但可在契约与框架落地后推进，故排在 P2。

**独立验证**：在新目录执行 `px-plugin init com.example.demo`，验证生成项目结构、依赖版本、 `plugin.yaml` 与 README 与仓库定义一致；运行 CLI 时未实现的命令需给出明确“设计中”提示。

**验收场景**：

1. 若开发者安装发布的 `px-plugin`，当执行 `px-plugin init com.demo.todo` 时，生成的后端/前端项目应分别通过 `go run` 与 `npm run dev` 启动且 Manifest/RBAC 占位与文档一致。
2. 若开发者尝试运行 `px-plugin package`，当命令仍处最小实现阶段时，CLI 应执行约定的构建步骤或返回清晰的 TODO/实验性提示（非静默失败）。

---

### 用户故事 4：版本发布与示例验证 (Priority: P3)

作为交付负责人，我希望仓库提供标准化发布流程与 `examples/starter` 生成物，这样可以验证 CLI 成品并将 Go Module、npm 包、CLI 二进制一并发布。

**优先级原因**：发布闭环能证明框架可被生产使用，但必须建立在骨架、契约与 CLI 基础之上，因此归为 P3。

**独立验证**：使用 Release 流水线完成一次从 `px-plugin init` 生成、`package`/`dist` 打包到 Git Tag 与 npm 发布的流程，并将生成物保存于 `examples/starter`。

**验收场景**：

1. 若 Release CI 运行，当执行打包与发布任务时，应生成 Go Module Tag、npm 包版本、CLI 二进制并更新变更日志。
2. 若版本发布后开发者对照 `examples/starter`，当在本地运行生成项目时，应与 CLI 实际输出无差异并包含多语言扩展 TODO 提示。

### 边界场景

- 当用户在离线环境运行 `px-plugin init` 时，CLI 必须仍能渲染模板（确保二进制内置模板与契约数据）。
- 当使用 `--backend` 或 `--frontend` 传入非 Go/Nuxt 选项时，CLI 应返回明确的实验性提示并继续生成默认模板。
- 当契约 Schema 更新但框架未同步时，CI 必须阻止合并并提示需要的生成/校验步骤。
- 当 `npm` 或 `go` 构建失败（如缺少本地工具链）时，脚手架 README 需提供排查指引而不是沉默失败。
- 当 Marketplace API 未返回成功状态时，`px-plugin publish` 应回滚本地临时产物并输出可追踪的错误信息。

## 技术设计概览 *(对齐参考)*

- **仓库结构**：与 `docs/init-project.md#powerxplugin-structure` 对齐，涵盖 `framework/`（共享后端能力）、`framework/frontend/nuxt`（Nuxt Layer 与客户端）、`skeleton/backend|web-admin`（可运行示例）以及 `scaffold/templates/**`（CLI 渲染模板），要求 skeleton 与模板职责对应。
- **后端框架契约**：`bootstrap.App` 暴露 `Listen`、`Env`、`Standalone` 等配置，并注入日志、数据库与可插拔路由；`router.RegisterFrameworkRoutes(app)` 负责 `/healthz` 等系统端点，`router.RegisterPluginRoutes(app, internal.Routes)` 挂载业务入口。`manifest.Register(app, manifestx.Plugin())` 与 `rbac.Report(app)` 负责清单及权限上报，详见 `docs/init-project.md#framework-signatures`。
- **运行模式**：实现需兼容 Standalone（`STANDALONE=true`，直接 `go run ./cmd/plugin` 暴露接口）与宿主代理模式（宿主透传 `/_p/<plugin-id>/api/**`）。CLI 与 skeleton README 需说明两种模式的切换方式。
- **前端 Nuxt Layer**：`@powerx-plugin/framework-admin` 以 Layer + Module 形式提供 `definePowerXAdminConfig`，统一设置基准路由 `/_p/<plugin-id>/admin`，注入鉴权/租户中间件，自动导入 `PX*` 组件并提供 Starter 页面；同路径同名文件可覆盖默认实现。`@powerx-plugin/framework-client` 提供 `usePluginApi` 访问代理后的 REST 接口。
- **路由前缀与代理约束**：所有前端路由以 `/_p/<plugin-id>/admin/` 为基准前缀，后端业务 API 以 `/_p/<plugin-id>/api/v1/` 为基准，确保与宿主路由代理和 `framework-client` 默认行为一致。
- **契约产物**：Phase 1 输出 `docs/contracts/manifest.json`、`docs/contracts/rbac.json`、`docs/contracts/openapi.yaml` 等文件，CI 需生成可阅读文档并提供框架/SDK 校验钩子。
- **示例与模板**：`docs/init-project.md` 中的后端入口、路由、前端页面以及 Manifest/RBAC 示例具有约束力，skeleton 与模板需保持一致，确保 CLI 输出与文档示例匹配。

## 需求 *(必填)*

### 功能性需求

- **FR-001**: 仓库根目录必须提供 `go.work`，显式 `use ./framework` 与 `use ./tools/cli`，并确保这两个模块可单独执行 `go test ./...`。
- **FR-002**: `framework/frontend/nuxt` 必须配置 npm workspace（含 `package.json` 与锁定文件），覆盖 `framework-admin` 与 `framework-client` Layer 并支持分别发布。
- **FR-003**: `skeleton/backend` 与 `skeleton/web-admin` 必须提供最小可运行示例：后端可直接执行 `go run ./skeleton/backend/cmd/plugin` 暴露 `GET /api/v1/ping`，前端在 `skeleton/web-admin` 内执行 `npm run dev` 并默认挂载 `@powerx-plugin/framework-admin` Layer，且 README 记录 Go 1.24+/Node 18+ 依赖与启动指引。
- **FR-004**: Scaffold 模板（`scaffold/templates/backend/go-gin`、`scaffold/templates/web-admin/nuxt`）必须与 skeleton 结构、入口代码和配置保持一致，CLI 渲染后的项目无需重命名或手工补档即可运行上述命令。
- **FR-005**: `px-plugin init` 必须嵌入模板与契约元数据，在无网络环境中仍可生成完整项目，并在生成结果中写入包含 `backend`/`frontend`/`version` 字段的 `plugin.yaml` 以及 README/TODO 提示。
- **FR-006**: `docs/` 必须声明仅支持 Go + Nuxt，并将多语言扩展需求记录在 `docs/backlog/multi-language.md`，随迭代更新状态。
- **FR-007**: Manifest、RBAC、健康检查与 API 契约必须以 JSON Schema/OpenAPI 形式维护在 `docs/contracts/manifest.json`、`docs/contracts/rbac.json`、`docs/contracts/openapi.yaml` 等文件中，并通过 CI 生成可阅读文档。
- **FR-008**: 框架后端与 SDK 必须消费契约 Schema（代码生成或运行时校验），若契约与实现不一致，CI 应失败并给出修复提示。
- **FR-009**: `px-plugin package`、`px-plugin dist`、`px-plugin publish` 必须提供最小可用实现：依次完成后端/前端构建、产物打包、调用 Marketplace API 上传，并在 README/CHANGELOG 中记录命令状态与“experimental”标记，提醒使用者功能仍处试验阶段。
- **FR-010**: `framework/backend/go/middleware` 必须提供默认拒绝请求的 `AuthGuard` stub，并在示例代码中引用以防误用。
- **FR-011**: `examples/starter/` 必须保存用 CLI 生成的最新参考项目，并在每次模板或依赖更新后同步刷新。
- **FR-012**: Release 流程必须生成 Go Module Tag、npm 包版本、CLI 二进制与校验文件，并在文档中提供安装与升级指引。

### 核心实体 *(如涉及数据需说明)*

- **插件契约 Schema**: 描述 Manifest、RBAC、健康检查、API 结构的 JSON Schema/OpenAPI 文件，是仓库与外部插件共享的协议源。
- **框架套件**: `github.com/ArtisanCloud/PowerXPlugin/framework/...` 与 `@powerx-plugin/framework-*` 输出的可复用组件，封装公共装配、端点、中间件与前端 Layer。
- **脚手架模板产物**: `scaffold/templates/**` 中供 CLI 渲染的模板文件集合，产出 `{plugin-skeleton}` 项目的目录、配置、占位代码与版本锁定。

## 假设

- Marketplace API 的鉴权方案与端点已由平台团队提供，CLI 只需按既定接口调用。
- 开发者默认具备 Go 1.24+、Node 18+ 环境，本项目不负责跨版本兼容性保障。
- 多语言、多前端栈支持列为长期规划，本次交付仅输出 Go + Nuxt 相关实现与 TODO。

## 成功标准 *(必填)*

### 可量化结果

- **SC-001**: 仓库克隆后 30 分钟内可完成 skeleton 后端与管理端的启动验证（含依赖安装），无阻塞性缺失。
- **SC-002**: 通过 CLI 初始化的插件，首次执行 `px-plugin init`、安装依赖并运行示例路由/页面的成功率达到 95%+。
- **SC-003**: 契约 Schema 变更后，CI 在 10 分钟内产出最新文档，并阻止与契约不兼容的实现合并。
- **SC-004**: 每次发布后，两周内收集的外部插件接入反馈中，与脚手架结构或契约不符的阻塞问题占比低于 5%。
