**PowerXPlugin 仓库的目标**是同时承载可运行的 Skeleton 以及可复用的“框架层”能力（供 PowerXPluginNote 等外部插件引用），并通过 `github.com/powerx-plugin/framework/...`（Go/Gin 默认实现）与 `@powerx-plugin/framework-admin` / `@powerx-plugin/framework-client` 输出统一组件生态。

> ⚠️ 当前仓库仍处于文档与设计阶段，尚未提供完整的 Skeleton、框架层代码或可用 CLI。本文档在各章节中明确“已落地”“进行中”“待规划”状态，请在推进实现前先校对对应章节的约束与 TODO。

> ✅ 当前实现范围仅涵盖 **Go + Nuxt**。多语言或其他框架的扩展路线请参见 `docs/backlog/multi-language.md`。

## 落地路线图（聚焦脚手架生成同级别插件）

- 当前仓库仅包含文档，完整实现可参考已有的 Base 插件项目。目标是在 `PowerXPlugin` 中沉淀框架与模板，使 `px-plugin init` 能一键生成该级别的插件。

### Phase 0 · 仓库地基
- 建立多模块结构：根目录添加 `go.work`，将 `framework/` 与 `tools/cli/` 注册为可独立构建的 Go Module。
- 前端使用 `npm` workspaces：在 `framework/frontend/nuxt/` 下维护 Layer 包的 `package.json` 与锁定文件，统一管理依赖版本。
- 配置基础 CI（Go Lint/Test、npm run build）和发布脚本，为后续版本化做准备。
- 在 `docs/` 中写明“当前仅支持 Go + Nuxt 实现”，并将多语言扩展的需求与约束记录在单独的 backlog（如 `docs/backlog/multi-language.md`），待协议稳定后再启动。

### Phase 1 · 协议沉淀（语言无关）
- （进行中）梳理插件与宿主的通信契约：Manifest、RBAC、鉴权上下文、健康检查、日志/指标约定，并沉淀到 `docs/contracts/manifest.json`、`docs/contracts/rbac.json` 等文件中。
- （待完成）输出对应的 OpenAPI/JSON Schema，通过 `docs/contracts/openapi.yaml` 统一描述 `/api/v1/**`、`/healthz`、环境变量要求，并由 CI 生成可阅读文档。
- （待接入）在 `framework/` 与 `sdk/` 中以代码生成或运行时校验的方式消费这些 Schema，保证契约与实现的映射关系可追踪。

### Phase 2 · Skeleton 抽取（当前 Go + Nuxt 实现）
- 以现有 Base 插件为蓝本，筛出最小可运行逻辑搬运到仓库内的 `skeleton/backend/` 与 `skeleton/web-admin/`，要求：`go run ./cmd/plugin` 以及 `npm run dev` 可直接启动。
- 在此基础上整理 `{plugin-skeleton}/backend/`、`{plugin-skeleton}/web-admin/` 模板（脚手架输出目录），默认生成 `GET /api/v1/ping` 等示例 API。
- 模板内不强制业务分层，只保留 routes/handler/service/repo 示例，鼓励团队按 DDD 自行划分。

### Phase 3 · 框架层拆分（以 Go + Nuxt 为首个实现）
- 将 skeleton 中的公共装配、系统端点、Manifest/RBAC 逻辑沉淀到 `framework/backend/go`，通过根模块 `github.com/powerx-plugin/framework/...` 暴露为默认 Go 实现。
- 将共享的前端 Layer 与客户端抽出到 `framework/frontend/nuxt/framework-admin|client`。
- 在文档中标注“未来可新增 `framework/backend/<lang>`、`framework/frontend/<stack>`”。

### Phase 4 · Scaffold 模板与 CLI 扩展
- （进行中）研究 Base 插件的目录命名、配置文件，编写 `scaffold/templates/backend/go-gin` 与 `scaffold/templates/web/nuxt`，确保能渲染出 Phase 2 中定义的 Skeleton。
- （规划中）CLI 首阶段只暴露 Go + Nuxt 模板，保留 `--backend`、`--frontend` 参数但标记为 `experimental`，待多语言模板准备就绪后再正式开放。
- （规划中）`plugin.yaml` 中记录框架类型（`backend: go-gin`, `frontend: nuxt`），供宿主或 CLI 在构建、打包阶段读取。
- （待开发）在 `px-plugin` CLI 中补齐 `package` / `dist` / `publish` 子命令的最小实现：`package` 负责编译后端（`go build`）与前端（`npm run build`），`dist` 将构建产物与 `plugin.yaml`、CHANGELOG 等打包，`publish` 则调用 Marketplace API 上传。当前文档中涉及这些命令的流程仅是设计稿，执行前需确认对应命令已实现并在 `CHANGELOG` 记录。
- （待补充）在 `framework/backend/go/middleware` 中放置 `AuthGuard` stub：默认返回 `501 Not Implemented` 或直接拒绝请求，并在文档、示例代码中引用该 stub，避免开发者误以为权限校验已经可用。

### Phase 5 · 验收与示例
- 使用全新目录执行 `px-plugin init <plugin-id>`（默认生成 Starter UI），与现有 Base 插件做差异对比，确认模板覆盖度。
- 在 `examples/starter/` 保存 CLI 生成物，并记录多语言扩展的待办清单（而非完整指引），等实际实现到位后再回填案例。
- 完成版本化发布：Go Module 打 Tag，npm 包发布；同时在 `docs/` 记录扩展协议、语言适配指南。CLI 在 Release 中提供编译好的 `px-plugin` 二进制（macOS/Linux/Windows），并生成安装脚本或 Homebrew/apt 仓库元数据。
- （待自动化）提供脚本或 CI 任务，同步更新 `framework/frontend/nuxt` 中各 npm 包的版本号、生成 `CHANGELOG`，并回写脚手架模板的依赖锁定，降低多包发布的人工成本。

### 阶段性检查项
- [ ] skeleton 可独立运行，提供健康检查与示例 API。
- [ ] 外部插件通过 import `github.com/powerx-plugin/framework` 编译通过。
- [ ] CLI 生成的项目（后端 + 前端）即取即用，默认即可运行。
- [ ] 文档覆盖初始化流程、目录约定、扩展点说明。

下面给你两个层面的**目录结构 + 说明**：

---

# 一、PowerXPlugin（Skeleton + Framework + Scaffold 的单仓 Monorepo）

> 对外品牌：**PowerXPlugin**
> 技术命名：`powerx-plugin`（连字符）

```
PowerXPlugin/                               # ← 仓库根（品牌 PowerXPlugin）
├─ framework/                                # [后端框架层] 多语言入口
│  ├─ backend/
│  │  ├─ go/                                 # 首个实现：Go/Gin
│  │  │  ├─ bootstrap/                       # App 装配、配置加载、Run()
│  │  │  ├─ router/                          # RegisterFrameworkRoutes / RegisterPluginRoutes
│  │  │  ├─ manifest/                        # 插件元数据契约（ID、Menus、Permissions…）
│  │  │  ├─ rbac/                            # 权限声明与上报
│  │  │  ├─ tenancy/                         # 多租户上下文
│  │  │  ├─ middleware/                      # auth、recovery、trace、audit
│  │  │  ├─ observability/                   # metrics、tracing
│  │  │  ├─ shared/                          # 错误/常量/工具
│  │  │  └─ README.md                        # 说明 Go 实现结构
│  ├─ go.mod                                 # module github.com/powerx-plugin/framework（默认 Go 实现的 import path）
│  └─ README.md                              # 协议文档索引（Manifest/RBAC/Health 等），记录多语言扩展指引
│
├─ framework/frontend/nuxt/                  # 前端运行时 Layer（admin/client）
│  ├─ package.json                           # "workspaces": ["frontend/*"]
│  └─ frontend/
│     ├─ nuxt/
│     │  ├─ framework-admin/                 # npm: @powerx-plugin/framework-admin
│     │  │  ├─ layer/                        # Nuxt Layer：默认布局/中间件/Starter页
│     │  │  │  ├─ app/
│     │  │  │  │  ├─ components/powerx/      # PX* 组件（可被插件同路径重载）
│     │  │  │  │  │  ├─ PXAdminLayout.vue
│     │  │  │  │  │  ├─ PXNav.vue
│     │  │  │  │  │  └─ ...
│     │  │  │  │  ├─ middleware/
│     │  │  │  │  │  ├─ auth-guard.global.ts
│     │  │  │  │  │  └─ tenant-context.global.ts
│     │  │  │  │  ├─ pages/_p/[pluginId]/admin/index.vue   # 可选：Starter 首页（存在则默认展示）
│     │  │  │  │  └─ plugins/permissions.ts                # v-permission 注册
│     │  │  │  └─ nuxt.config.ts                           # Layer 自身配置（components/imports）
│     │  │  ├─ module.ts                     # Nuxt Module（注入 baseURL、中间件、auto-import）
│     │  │  ├─ index.ts                      # definePowerXAdminConfig({ pluginId, starterPages })
│     │  │  └─ package.json
│     │  └─ framework-client/                # npm: @powerx-plugin/framework-client
│     │     ├─ http.ts                       # $fetch/axios 包装，401/403 统一处理
│     │     ├─ api.ts                        # createPluginApi / usePluginApi（自动拼 /_p/<id>/api/v1）
│     │     ├─ index.ts
│     │     └─ package.json
│     └─ README.md                           # 引导未来新增 React/Next 等实现
│
│
├─ scaffold/                                 # [脚手架模板] 仅渲染，不被 import
│  ├─ templates/
│  │  ├─ backend/
│  │  │  └─ go-gin/
│  │  │     ├─ cmd/plugin/main.go.tmpl
│  │  │     ├─ internal/routes.go.tmpl
│  │  │     ├─ internal/service/example_service.go.tmpl
│  │  │     └─ internal/manifestx/manifest.go.tmpl
│  │  └─ web/
│  │     └─ nuxt/
│  │        ├─ nuxt.config.ts.tmpl
│  │        └─ app/pages/_p/{{ .PluginID }}/admin/index.vue.tmpl
│  └─ README.md                              # 说明如何新增其他栈模板
│
├─ tools/cli/                                # [CLI] 二进制 px-plugin
│  ├─ cmd/init.go                            # 渲染 scaffold/templates，钉住框架版本
│  ├─ cmd/package.go                         # 计划：编译后端/前端，输出构建产物
│  ├─ cmd/dist.go                            # 计划：打包 bundle（后端 + 前端 + plugin.yaml）
│  ├─ cmd/publish.go                         # 计划：上传 Marketplace
│  ├─ cmd/selfupdate.go
│  ├─ main.go
│  └─ go.mod
│
├─ skeleton/                                 # [可运行 Skeleton 示例]（演示最小 CRUD）
│  ├─ backend/ ...                           # 复用 framework；可内存/SQLite
│  └─ web-admin/ ...                         # 复用 framework-admin/client
│
├─ examples/                                 # 用 Scaffold 生成的实战样例
│  └─ starter/
│
├─ docs/                                     # 指南、约定、升级说明
└─ README.md
```

### 关键说明

* **PowerXPlugin 仓库本身可直接运行（skeleton 目录）**，也是“最佳实践参考”。
* **其他插件项目**（如 PowerXPluginNote）只**import 框架**：

  * 后端：`github.com/powerx-plugin/framework/...`（当前默认 Go 实现）
  * 前端：`@powerx-plugin/framework-admin` / `@powerx-plugin/framework-client`
* **Scaffold** 只负责把“入口 + 钩子 + 占位文件 + 固定依赖版本”渲染到目标项目；**不会被 import**。
* `@powerx-plugin/framework-*` 以 npm workspace 形式维护源码，但发布时需分别执行 `npm publish --workspace <pkg>`，并在每次 release 更新脚手架模板中的版本锁定（例如 `scaffold/templates/web/nuxt/package.json.tmpl`），否则外部插件无法锁定稳定依赖。
* 多语言与多前端栈仍在调研中：仓库结构中预留了目录，但没有任何非 Go/Nuxt 的可执行实现。开启新语言前需先在 `docs/backlog/multi-language.md` 完成设计评审。

---

# CLI 分发与安装策略（px-plugin）

> 当前实现状态：`px-plugin` 仅提供 `init` 子命令的雏形，其余命令仍在开发路线图中。以下流程描述的是目标状态，执行前需要确认对应命令已经合入并发布。

- CLI 源码位于 `tools/cli/`，通过 Go 构建为单个可执行文件 `px-plugin`，并计划使用 `go:embed`（或同类机制）将 `scaffold/templates/**/*` 以及协议元数据打包进二进制；因此即便单独分发 `px-plugin`，也能离线渲染 `{plugin-skeleton}` 所需的模板。
- 推荐安装方式：
  * **Go 安装**：`go install github.com/powerx-plugin/PowerXPlugin/tools/cli@latest`，便于开发阶段获取最新版本；
  * **预编译发行包**：在 Release CI 中生成 `px-plugin-<os>-<arch>.tar.gz` / `.zip` 与校验文件，供生产环境下载；可选提供 Homebrew Tap、APT/YUM 仓库或 NPM wrapper，简化安装体验。
- `{plugin-skeleton}` 项目不内置 CLI 二进制，仅在 README/Makefile 中提醒开发者安装或更新 `px-plugin`；CLI 会读取项目内 `plugin.yaml`、`package.json`、`go.mod` 等信息完成后续操作。
- 计划支持的核心子命令（除 `init` 外均处于待实现阶段）：
  * `px-plugin init <plugin-id> [--backend=go-gin] [--frontend=nuxt]`：生成脚手架（当前仅支持 Go + Nuxt 模板，其他参数会忽略并打印提示）；
  * `px-plugin package`：在插件项目中执行，调用 `go build`、`npm run build` 等命令，统一输出构建产物（默认放置于 `build/` 或 `.px/dist`）；
  * `px-plugin dist`：将 package 结果与 `plugin.yaml`、CHANGELOG、校验信息打包成标准 bundle（如 `dist/<plugin-id>-<version>.zip`）；
  * `px-plugin publish --token=<api-token>`：调用 PowerX Marketplace API 上传 bundle，支持版本校验、发布说明、回滚策略；
  * `px-plugin selfupdate`：从官方 Release 检查并下载最新 CLI。
- Marketplace 需提供对应 REST/GraphQL API（上传、版本列表、状态查询），并约定鉴权方式（PAT、OAuth、签名链接等）；CLI 与宿主的部署流水线将围绕这些接口构建。

---

# 二、被生成的插件项目示例：{plugin-skeleton}

- 默认通过 `px-plugin init com.powerx.note` 生成（可选 `--ui=starter`/`--ui=blank`）

```
com.powerx.note/
├─ plugin.yaml
├─ backend/
│  ├─ go.mod                         # require github.com/powerx-plugin/framework vX.Y.Z
│  └─ cmd/plugin/main.go             # 入口：只做装配与挂载
│     internal/
│       ├─ routes.go                 # 业务路由（/_p/<id>/api/v1/**）
│       ├─ handler/                  # HTTP Handler（可自定义）
│       ├─ service/                  # 应用服务（可自定义）
│       ├─ repo/                     # 仓储层（可自定义）
│       └─ manifestx/manifest.go     # Menus/Permissions（与前端菜单对齐）
│
└─ web-admin/
   ├─ package.json                   # 依赖 @powerx-plugin/framework-admin/client
   ├─ nuxt.config.ts                 # definePowerXAdminConfig({ pluginId: 'com.powerx.note' })
   └─ app/
      ├─ components/                 # 可定义（或覆盖 PX* 组件）
      │  └─ powerx/
      │     └─ PXNav.vue             # 同路径同名 → 覆盖框架默认
      ├─ pages/
      │  └─ _p/com.powerx.note/admin/
      │     ├─ index.vue             # 可选：覆盖 Layer 默认首页（blank 模式下会生成占位文件）
      │     └─ notes/index.vue       # 自定业务页
      ├─ plugins/
      │  ├─ powerx.client.ts         # 可选：提供 $pluginApi
      │  └─ menus.ts                 # 可选：registerMenu([...])
      └─ middleware/                 # 业务中间件（系统守卫已由 Layer 注入）
```

> **DDD 提示**：模板默认提供基础 `routes/handler/service/repo` 目录，仅供参考；团队可根据 DDD/六边形等实践在 `internal/` 下自行调整分层。

### 后端入口（生成）

```go
package main

import (
  "github.com/powerx-plugin/framework/bootstrap"
  "github.com/powerx-plugin/framework/router"
  "github.com/powerx-plugin/framework/manifest"
  "github.com/powerx-plugin/framework/rbac"

  "github.com/acme/powerx-plugin-note/internal"
  "github.com/acme/powerx-plugin-note/internal/manifestx"
)

func main() {
  app := bootstrap.NewAppFromEnv()

  router.RegisterFrameworkRoutes(app)              // 固定：健康检查 /healthz + 平台系统端点
  router.RegisterPluginRoutes(app, internal.Routes) // 只开放业务挂载点

  manifest.Register(app, manifestx.Plugin())       // 上报 ID、菜单、权限
  rbac.Report(app)                                 // 可选：RBAC 同步
  app.Run()
}
```

### 业务路由（生成）

```go
func Routes(rg *gin.RouterGroup) {
  api := rg.Group("/api/v1")
  // 例如：/api/v1/notes/** ...
}
```

> 宿主代理：插件自身暴露 `/api/v1/**` 与 `/healthz` 等端点；部署到 PowerX Core 后，宿主会透过动态路由将请求转发到 `/_p/<plugin-id>/api/**`（前端 Admin 静态资源同理映射为 `/_p/<plugin-id>/admin/**`）。

---

# 三、前端的“默认可用 + 自定义重载”机制

**框架包 `@powerx-plugin/framework-admin` 是 Nuxt Layer + Module**：

* `definePowerXAdminConfig({ pluginId, starterPages })`

  * 设置 `app.baseURL = '/_p/<plugin-id>/admin'`
  * 自动注册中间件：`auth-guard`、`tenant-context`、统一错误页
  * 自动导入组件 `PX*` 与 composable（通过 `imports`/`components`）
  * 当 `starterPages: true` 时，**提供默认 Starter 页面**（Dashboard 等）
* **重载规则（Nuxt Layers）**：插件项目内若存在**同路径同名文件**，**立即覆盖框架默认**

  * 覆盖默认首页：创建 `app/pages/_p/com.powerx.note/admin/index.vue`
  * 覆盖默认侧边栏：创建 `app/components/powerx/PXNav.vue`

> ⚠️ 文件级覆盖属于“最后手段”。优先通过 `PXAdminLayout` 的插槽、`registerMenu`、可配置选项或计划中的扩展 API 来定制界面。若必须覆盖文件，请在 PR/变更说明中注明覆盖意图，并为覆盖文件补充最小的组件测试或可视化回归检查，避免无意覆盖核心能力。

### 使用示例（业务页）

```vue
<script setup lang="ts">
import { PXAdminLayout } from '@powerx-plugin/framework-admin'
import { usePluginApi } from '@powerx-plugin/framework-client'

const api = usePluginApi({ pluginId: 'com.powerx.note' })
const { data: notes } = await api.get('/api/v1/notes')
</script>

<template>
  <PXAdminLayout>
    <template #toolbar>
      <div class="flex justify-between">
        <h1>Notes</h1>
        <button class="btn">New</button>
      </div>
    </template>

    <ul class="mt-4">
      <li v-for="n in notes" :key="n.id">{{ n.title }}</li>
    </ul>
  </PXAdminLayout>
</template>
```

### 权限与菜单

* 全局指令：`v-permission="'com.powerx.note.admin.view'"`
* 页面元信息：`definePageMeta({ permission: 'com.powerx.note.note.read', breadcrumb: ['Notes'] })`
* 可选菜单注册：`registerMenu([{ path: '/_p/com.powerx.note/admin/notes', title: 'Notes', perm: '...' }])`

后端权限校验需要与前端保持同一命名约定：在 handler 中通过即将提供的 `middleware.AuthGuard()` 读取请求上下文的权限集合，并对照 Manifest/RBAC Schema 进行校验。当前仓库仅有接口定义，具体实现需在 `framework/backend/go/middleware` 与宿主鉴权服务对接后方可启用；在此之前，请确保关键接口仍进行手动校验或返回明确的 `501 Not Implemented`。

---

# 四、契约与不变量（宿主对齐）

* **前端页面前缀**：`/_p/<plugin-id>/admin/**`（框架自动设置 baseURL）
* **业务 API 前缀**：`/_p/<plugin-id>/api/v1/**`（framework-client 自动拼接）
* **系统端点**（不可改）：`/healthz`、`/api/v1/admin/manifest` 等由框架统一注册；部署到宿主后由 PowerX 反向代理到 `/_p/<plugin-id>/api/**`。
* **RBAC Key 命名**：`<plugin-id>.<domain>.<action>`（示例：`com.powerx.note.note.read`）

---

# 五、初始化选项（CLI）

```bash
# 默认（Starter）：使用框架内置 Starter 页
px-plugin init com.powerx.note

# 显式指定 Starter（可选）
px-plugin init com.powerx.note --ui=starter

# Blank：只生成 nuxt.config.ts 和目录，页面完全自建
px-plugin init com.powerx.note --ui=blank
```

---

## 总结

* **PowerXPlugin 仓库**：一处维护，既能**自己跑**（skeleton 目录），又能**输出框架与模板**给所有插件用。
* **插件项目（如 PowerXPluginNote）**：只 import **framework**，**不 import scaffold**；前端通过 **Layer 覆盖**实现“默认即用 + 自定义重载”。
* 不变量（路由前缀、系统端点、RBAC、Manifest）由 **framework** 统一约束，业务方只填充 **/api/v1** 与页面。

---

# 后端（Go）— Framework 导出签名

## 1) Go 后端（bootstrap）

> 位置：`framework/backend/go/bootstrap`（对外 import 为 `github.com/powerx-plugin/framework/bootstrap`）

**`framework/backend/go/bootstrap/app.go`**

```go
package bootstrap

import (
	"context"
	"database/sql"
	"log/slog"
)

type App struct {
	Ctx     context.Context
	Config  *Config
	Logger  *slog.Logger
	DB      *sql.DB
	Router  Router // 由 router 包注入（抽象接口，实际可用 gin.Engine 包装）
	closeFn func() error
}

type Config struct {
	Listen   string // ":8078"
	Env      string // "dev|prod"
	Standalone bool // STANDALONE=true 时启用 Demo/本地策略
	// DB、缓存、追踪等...
}

type Option func(*Config)

func WithListen(addr string) Option
func WithEnv(env string) Option
func WithStandaloneDefaults() Option

func NewApp(cfg *Config) *App
func NewAppFromEnv(opts ...Option) *App
func (a *App) Run() error           // 启动 HTTP 监听（在 router 初始化后）
func (a *App) Shutdown() error      // 优雅关闭
```

**运行模式说明**

- 当 `Standalone=true` 时，后端直接监听 `Config.Listen` 并暴露 `/api/v1/**` 与 `/healthz`。该模式用于本地开发与 CI self-test，不依赖宿主代理。
- 当 `Standalone=false` 时，服务应部署在宿主侧，由宿主反向代理到 `/_p/<plugin-id>/api/**`。构建产物与配置文件需要由 CLI `package`/`dist`（待实现）生成，以确保路径一致。
- 文档与脚手架需在 README 中提示如何在两种模式间切换，例如通过 `.env` 中的 `STANDALONE` 与 `POWERX_BASE_URL`。未显式配置时默认走 Standalone，本地调试后再结合部署 pipeline 手动调整。

**`framework/backend/go/bootstrap/router_port.go`**

```go
package bootstrap

// Router 抽象，避免直接暴露具体框架；router 包会提供注入器。
type Router interface {
	Group(rel string) Router
	Handle(method, path string, h Handler)
	Use(mw ...Middleware)
}

type Handler func(Context)
type Middleware func(Handler) Handler

type Context interface {
	Param(name string) string
	Query(name string) string
	BindJSON(v any) error
	JSON(code int, v any)
	Status(code int)
	// ...按需扩展
}
```

---

## 2) Go 后端（router）

> 位置：`framework/backend/go/router`

**`framework/backend/go/router/router.go`**

```go
package router

import "github.com/powerx-plugin/framework/bootstrap"

const (
	HealthzPath = "/healthz" // 健康检查端点
	APIPrefix   = "/api/v1"  // 业务路由前缀（宿主会代理为 /_p/<id>/api/v1/**）
)

// 将具体的 HTTP 框架（如 gin）适配为 bootstrap.Router，并注入到 app。
func AttachHTTPServer(app *bootstrap.App /* 可变参: 端口/MW 等 */) error

// 框架内置“不可改”的系统端点：健康、manifest、rbac、metrics...
func RegisterFrameworkRoutes(app *bootstrap.App)

// 开放插件挂载点：业务只能在这里注册；最终路径为 /api/v1/**
func RegisterPluginRoutes(app *bootstrap.App, reg func(rg bootstrap.Router))
```

---

## 3) Go 后端（manifest）

> 位置：`framework/backend/go/manifest`

**`framework/backend/go/manifest/manifest.go`**

```go
package manifest

type Menu struct {
	Path  string `json:"path"`
	Title string `json:"title"`
	Icon  string `json:"icon,omitempty"`
	Order int    `json:"order,omitempty"`
}

type Plugin struct {
	ID          string   `json:"id"`   // com.powerx.xxx
	Name        string   `json:"name"`
	Version     string   `json:"version,omitempty"`
	Menus       []Menu   `json:"menus,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	// 可扩展：Capabilities、Webhooks、Quotas...
}

type App interface {
	RegisterManifest(p Plugin)
}

func Register(app App, p Plugin)  // 简化辅助；router.RegisterFrameworkRoutes 会读取
```

---

## 4) Go 后端（rbac）

> 位置：`framework/backend/go/rbac`

**`framework/backend/go/rbac/rbac.go`**

```go
package rbac

type Permission struct {
	Key   string `json:"key"`   // com.powerx.note.note.read
	Scope string `json:"scope"` // tenant/system...
	Desc  string `json:"desc,omitempty"`
}

func Report(app any /* or a dedicated interface */, perms []Permission) error
```

---

## 5) 其他（按需）

* Go 实现目录（`framework/backend/go/middleware`，对外 import: `github.com/powerx-plugin/framework/middleware`）：`Recovery()`, `Trace()`, `Audit()`, `AuthGuard()`
* Go 实现目录（`framework/backend/go/observability`，import: `github.com/powerx-plugin/framework/observability`）：`InitMetrics(app)`, `InitTracing(app)`
* Go 实现目录（`framework/backend/go/tenancy`，import: `github.com/powerx-plugin/framework/tenancy`）：`WithTenant(ctx, tenantID string) context.Context`, `TenantFrom(ctx)`

---

# 前端（TS/Nuxt）

## 1) `@powerx-plugin/framework-admin`

> 位置：`framework/frontend/nuxt/framework-admin`

**目录**

```
frontend/nuxt/framework-admin/
├─ layer/
│  ├─ app/components/powerx/PXAdminLayout.vue
│  ├─ app/components/powerx/PXNav.vue
│  ├─ app/middleware/auth-guard.global.ts
│  ├─ app/middleware/tenant-context.global.ts
│  ├─ app/plugins/permissions.ts
│  ├─ app/pages/_p/[pluginId]/admin/index.vue  # Starter（可关）
│  └─ nuxt.config.ts                           # 注册 components/imports
├─ module.ts
└─ index.ts
```

**`index.ts`**

```ts
import { defineNuxtConfig } from 'nuxt/config'
import FrameworkAdmin from './module'

export interface PowerXAdminOptions {
  pluginId: string
  starterPages?: boolean   // 默认 true：使用 Layer Starter
}

export function definePowerXAdminConfig(opts: PowerXAdminOptions) {
  const { pluginId, starterPages = true } = opts
  return defineNuxtConfig({
    extends: ['@powerx-plugin/framework-admin/layer'],
    modules: [[FrameworkAdmin, { pluginId, starterPages }]],
    app: { baseURL: `/_p/${pluginId}/admin` },
  })
}

// 菜单注册（可选）
export type MenuItem = { path: string; title: string; icon?: string; perm?: string; order?: number }
const _menus: MenuItem[] = []
export function registerMenu(items: MenuItem[]) { _menus.push(...items) }
export function getRegisteredMenus() { return _menus }
```

**`module.ts`**

```ts
import { defineNuxtModule, addImportsDir, addComponentsDir, createResolver } from '@nuxt/kit'

export interface ModuleOptions { pluginId: string; starterPages?: boolean }
export default defineNuxtModule<ModuleOptions>({
  meta: { name: '@powerx-plugin/framework-admin' },
  defaults: { starterPages: true },
  setup(opts, nuxt) {
    const r = createResolver(import.meta.url)
    nuxt.options.runtimeConfig.public.powerx = {
      pluginId: opts.pluginId,
      baseApi: `/_p/${opts.pluginId}` // client 会自动补 /api/v1
    }
    // 自动导入组件/组合式
    addComponentsDir({ path: r.resolve('layer/app/components'), pathPrefix: false })
    addImportsDir(r.resolve('layer/app/composables'))
    // 允许按需关掉 Starter 页
    if (!opts.starterPages) {
      nuxt.options.ignore = nuxt.options.ignore || []
      nuxt.options.ignore.push('**/layer/app/pages/_p/**/admin/index.vue')
    }
  }
})
```

**`layer/app/components/powerx/PXAdminLayout.vue`（核心插槽）**

```vue
<script setup lang="ts">
defineProps<{ title?: string }>()
</script>

<template>
  <div class="min-h-screen flex">
    <PXNav />
    <main class="flex-1 p-6">
      <header class="mb-4">
        <slot name="toolbar" />
      </header>
      <slot />
    </main>
  </div>
</template>
```

**`layer/app/middleware/auth-guard.global.ts`**

```ts
export default defineNuxtRouteMiddleware((to) => {
  const perm = to.meta?.permission as string | undefined
  // TODO: 从会话/后端拉取权限集合，若缺失则导航到禁止页
  if (perm) {/* check & block */}
})
```

---

## 2) `@powerx-plugin/framework-client`

> 位置：`framework/frontend/nuxt/framework-client`

**目录**

```
frontend/nuxt/framework-client/
├─ api.ts
└─ index.ts
```

**`api.ts`**

```ts
export interface ApiOptions {
  pluginId: string
  baseApiPath?: string // 默认: /_p/<pluginId>
  getToken?: () => Promise<string> | string
}

export function createPluginApi(opts: ApiOptions) {
  const base = (opts.baseApiPath || `/_p/${opts.pluginId}`) + '/api/v1'
  async function request<T>(method: string, url: string, body?: any): Promise<T> {
    const token = (await opts.getToken?.()) || ''
    return await $fetch<T>(base + url, {
      method,
      headers: token ? { Authorization: `Bearer ${token}` } : undefined,
      body
    })
  }
  return {
    get:  <T>(u: string, q?: any) => request<T>('GET',    u, q),
    post: <T>(u: string, b?: any) => request<T>('POST',   u, b),
    put:  <T>(u: string, b?: any) => request<T>('PUT',    u, b),
    del:  <T>(u: string)          => request<T>('DELETE', u),
  }
}

export function usePluginApi(opts: ApiOptions) {
  return createPluginApi(opts)
}
```

**`index.ts`**

```ts
export * from './api'
```

---

# 插件项目中的最小使用（PowerXPluginNote）

**后端入口**

```go
package main

import (
  "github.com/powerx-plugin/framework/bootstrap"
  "github.com/powerx-plugin/framework/router"

  "github.com/acme/powerx-plugin-note/internal"
  "github.com/acme/powerx-plugin-note/internal/manifestx"
)

func main() {
  app := bootstrap.NewAppFromEnv()
  router.RegisterFrameworkRoutes(app)
  router.RegisterPluginRoutes(app, internal.Routes)
  manifestx.Register(app) // 你的业务 Manifest()
  app.Run()
}
```

**前端 `nuxt.config.ts`**

```ts
import { definePowerXAdminConfig } from '@powerx-plugin/framework-admin'
export default definePowerXAdminConfig({ pluginId: 'com.powerx.note', starterPages: true })
```

**业务页（存在即覆盖 Layer 默认）**

```vue
<script setup lang="ts">
import { PXAdminLayout } from '@powerx-plugin/framework-admin'
import { usePluginApi } from '@powerx-plugin/framework-client'
const api = usePluginApi({ pluginId: 'com.powerx.note' })
const { data } = await api.get('/api/v1/ping')
</script>

<template>
  <PXAdminLayout>
    <template #toolbar><h1>Notes</h1></template>
    <pre>{{ data }}</pre>
  </PXAdminLayout>
</template>
```

---

这些签名已经把“**后端装配/路由边界/Manifest**”和“**前端 Layer/覆盖机制/API 客户端**”固定好了。
你可以直接按这个骨架把现有 Base 的实现迁过去：先保证编译通过与最小跑通，再逐步丰富中间件、观测、RBAC 等实现细节。
