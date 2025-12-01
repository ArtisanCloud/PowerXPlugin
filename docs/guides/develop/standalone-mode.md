
# 运行模式指南（Standalone / Delegated）

本指南统筹说明 PowerX 插件在 **独立运行（Standalone）** 与 **宿主委派（Delegated）** 场景下的配置、启动顺序与常见排障。前半部分聚焦本地独立运行，后半部分整合原《delegated-mode》文档，帮助你在宿主内核中调试 iframe/代理相关问题。

## 1. Standalone 模式

### 1.1 目录结构与分层

```
backend/
└─ internal/
   ├─ bootstrap/         # 配置、依赖注入、进程生命周期
   ├─ config/            # YAML/环境变量解析
   ├─ entity/
   │  ├─ models/{domain} # 领域实体、值对象
   │  └─ repository/{domain} # 仓储接口与默认实现
   ├─ services/{domain}  # 应用服务，编排业务用例
   ├─ transport/
   │  ├─ http/admin/{domain}   # Web Handler，DTO 到实体转换
   │  └─ grpc/...              # gRPC Handler（可选）
   ├─ manifestx/         # 插件清单（菜单、权限）
   ├─ router/            # 注册 Gin 引擎、PowerX 框架路由
   ├─ observability/     # 指标、日志、链路追踪
   ├─ middleware/        # 跨域、RBAC、审计中间件
   └─ shared/            # 共用工具、常量
```

分层职责：

- **传输层** (`transport/http|grpc`)：只做协议转换，调用同域的服务层。
- **服务层** (`services/{domain}`)：封装业务流程，依赖实体与仓储接口。
- **领域层** (`entity/models` + `entity/repository`)：定义聚合根、值对象及仓储抽象。
- **基础设施**（仓储默认实现、DB、外部 API）：位于 `entity/repository/{domain}` 与 `transport` 的适配层。

Manifest 通过 `internal/manifestx/manifest.go` 暴露插件 ID、菜单、权限，供宿主读取。

### 1.2 启动流程

Skeleton 入口位于 `cmd/plugin/main.go`，关键步骤如下：

1. 读取配置：`config.Load()` 支持 `CONFIG_PATH` / `POWERX_PLUGIN_CONFIG_DIR`。
2. 初始化依赖：`bootstrap.BootstrapPlugin` 注入数据库、缓存、PowerX gRPC 客户端等。
3. 构建 Gin 引擎：`internal/router.NewRouter` 装配中间件与业务路由。
4. 挂载框架路由：
   ```go
   fwrouter.AttachHTTPServer(app)
   fwrouter.RegisterFrameworkRoutes(app)
   fwrouter.RegisterPluginRoutes(app, func(r bootstrap.Router) {
     httpserver.RegisterGinRoutes(r, engine)
   })
   ```
5. 注册 Manifest：`manifest.Register(app, manifestx.Plugin())`。
6. 启动 HTTP/gRPC、周期任务等，并监听退出信号安全关闭。

### 1.3 本地运行

```bash
# 1. （可选）复制示例配置
cp skeleton/backend/etc/config.example.yaml skeleton/backend/etc/config.yaml
#    指向默认的 skeleton/.cache/powerxplugin.db（需提前创建目录）
#    若放在其他目录，请通过 CONFIG_PATH 指向该目录

# 2. 初始化数据库（setup = migrate + seed）
cd skeleton/backend
export POWERX_PROXY=0
export POWERX_RBAC_DELEGATE=false
export PLUGIN_IAM_TENANT_KEY=px_local
export PLUGIN_IAM_TENANT_NAME="Local Tenant"
export PLUGIN_IAM_ADMIN_EMAIL=admin@local.test
export PLUGIN_IAM_ADMIN_PASSWORD='S3cret!!'
go run ./cmd/database/main.go setup
#    上述环境变量可选；若未设置，系统会使用 admin@local.test / S3cret!! 等默认值（仅限本地环境，生产务必覆盖）
#    本地接口也会强制校验 Authorization。若要临时跳过，可设置 POWERX_AUTH_OPTIONAL=true（仅限调试）。
#    如果需要单独执行，可替换为：
#    go run ./cmd/database/main.go migrate
#    go run ./cmd/database/main.go seed

# 3. 启动后端（默认使用 SQLite 文件 powerxplugin.db 保存数据）
go run ./cmd/plugin

# 4. 访问健康检查
curl http://127.0.0.1:8078/healthz

# 5. 启动前端管理端（使用本地管理员登录）
cd ../web-admin && npm install && npm run dev

# 6. （可选）运行本地 IAM E2E
cd ../web-admin
PLAYWRIGHT_LOCAL_IAM=1 \\
PLAYWRIGHT_LOCAL_EMAIL=admin@local.test \\
PLAYWRIGHT_LOCAL_PASSWORD='S3cret!!' \\
npm run test:e2e -- auth-local
```

常用调试端口：

- 后端 HTTP: `8078`（可通过 `PORT` 环境变量覆盖）
- 后端 gRPC: `8079`（通过 `POWERX_GRPC_PORT` 覆盖）
- 管理端 Nuxt: 默认 `3031`（冲突时自动寻找可用端口）

#### 1.3.1 模拟 `_p/<plugin-id>/admin` 访问

要在本地复刻宿主 iframe（`/_p/<plugin-id>/admin/**`），目前推荐直接让 Nuxt 以 `_p` 为基准运行。Skeleton 的 Go 进程不会自动把 `/_p/**` 代理到前端，因此如果没有外部反代，就必须在 Nuxt 启动参数里切换到“宿主模式”。

1. **前端开启宿主模式**：启动 dev server 前设置 `POWERX_PROXY=1`（或 `NUXT_PUBLIC_INSIDE_POWERX=1`）。Nuxt 会将 `app.baseURL` 与 `runtimeConfig.public.pluginAdminBase` 设为 `/_p/<plugin-id>/admin/`。为了避免本地 Dev Server 将 `_p/<plugin-id>/api` 当成静态路由，`runtimeConfig.public.apiBaseUrl` 会在开发模式下自动退回到 `/_p/<plugin-id>/api/v1`，从而始终命中 Vite 的代理。
2. **补齐 Vite 代理（已在 `skeleton/web-admin/nuxt.config.ts` 内置）**：配置依赖以下环境变量，便于在需要时修改目标地址：
   - `NUXT_DEV_API_PROXY`：HTTP 代理目标（默认 `http://localhost:8078`）
   - `NUXT_DEV_WS_PROXY`：WebSocket 代理目标（默认 `ws://127.0.0.1:4000`）

   对应配置代码如下，`/_p/${pluginId}/api` 的条目仅在 `POWERX_PROXY=1` 时注入：
   ```ts
   const pluginId = 'com.powerx.plugin.base'
   const devApiProxyTarget = process.env.NUXT_DEV_API_PROXY || 'http://localhost:8078'
   const devWsProxyTarget = process.env.NUXT_DEV_WS_PROXY || 'ws://127.0.0.1:4000'

   const INSIDE_POWERX = process.env.POWERX_PROXY === '1'
   const devProxy: Record<string, any> = {
     '/api': { target: devApiProxyTarget, changeOrigin: true, ws: true },
     '/ws': { target: devWsProxyTarget, changeOrigin: true, ws: true }
   }
   if (INSIDE_POWERX) {
     devProxy[`/_p/${pluginId}/api`] = { target: devApiProxyTarget, changeOrigin: true }
   }
   ```
   如此前端直接访问 `http://localhost:3031/_p/com.powerx.plugin.base/admin/templates/crud` 时，所有 API 请求都会被转发到本地 8078 实例，不会 404 或触发 CORS。
   同时在宿主模式下会自动关闭 Nuxt `appManifest`，避免 `manifest-route-rule` 在 `_p` 前缀下无法匹配路由而抛错。
3. **后端保持默认**：`POWERX_PROXY=0 go run ./cmd/plugin`，只需暴露 `/api/v1/**`。由于前端代理会把 `/_p/...` 请求映射回本地接口，后端无需额外改动。

配置完上述环境后，直接访问 `http://localhost:3031/_p/<plugin-id>/admin/...` 即可模拟宿主 iframe，Bridge/CTX/主题等行为与宿主一致。

> `skeleton/backend/etc/` 目录内包含示例 `config.yaml` 与 `security_baseline.yaml`。默认 DSN 为 `file:../.cache/powerxplugin.db?cache=shared&_fk=1`，Loader 会把它解析成相对于 `config.yaml` 的路径，因此无论在仓库根目录还是 `skeleton/backend` 执行命令，最终都会落在 `skeleton/.cache/` 下；若希望把文件放到仓库根目录，也可以把 DSN 改成 `file:../../.cache/powerxplugin.db?cache=shared&_fk=1` 或通过 `POWERX_DB_DSN` 环境变量覆盖。若改为纯内存 DSN（如 `file::memory:?cache=shared`），请在同一进程内连续执行 `migrate` 与 `seed`。示例配置同时关闭了 Marketplace 推荐和续费提醒的后台任务，避免在空表上触发告警。
>
> Loader 在解析 SQLite DSN 时会自动创建目标目录，无需手动 `mkdir`。如果路径中包含 `../.cache`，会基于配置文件目录进行展开。
>
> 也可以将 `runtime.run_migrate` 设为 `true` 或在启动命令前加 `POWERX_RUN_MIGRATE=true`，这样服务启动时会自动运行迁移。种子数据仍需手动执行 `go run ./cmd/database/main.go seed`。

> SQLite 下默认只迁移“核心/安全”表，目的是避免功能依赖与 FK/RLS 兼容性问题，方便做 IAM/启动层的最小验证。已经将 RSS 表加入白名单，本地可跑 RSS。若要开发完整业务，建议切到 Postgres（默认迁所有表，更接近上线），或者按需扩白名单再在 SQLite 上重跑 `go run ./cmd/database/main.go setup`。

> 默认导航栏左上角引用 `public/images/logo-s.png`。如果要替换 Logo，可在生成项目的 `public/images` 目录中用同名文件覆盖，或调整 `app/components/AppNavbar.vue` 中的 `<img>` 引用。

### 1.4 扩展点示例：新增模板审批接口

1. **领域模型**：在 `internal/entity/models/template` 新增 `approval.go`，描述审批状态。
2. **仓储接口**：在 `internal/entity/repository/template` 添加 `approval_repository.go`，定义 `FindPending`、`Approve` 等方法，并提供最小内存实现。
3. **应用服务**：于 `internal/services/template/approval_service.go` 编排审批流程，处理幂等与事件上报。
4. **传输层 Handler**：在 `internal/transport/http/admin/templates/approval_handler.go` 实现接口，`POST /templates/:id/approve` 调用服务层。
5. **路由注册**：更新 `internal/router/templates.go` 或所在域的路由，将新 Handler 挂载到 `/api/v1/templates` 子路由。
6. **Manifest 更新**：在 `internal/manifestx/manifest.go` 增加菜单或权限项，并同步前端导航。
7. **同步模板**：完成修改后执行 `npm run sync:templates`，确保 Scaffold/CLI 模板保持一致。

### 1.5 常见问题

- **403 或 401**：确认 `POWERX_SECURITY_*` 安全上下文配置正确；独立模式下可在配置中关闭严格模式。
- **CORS 报错**：`internal/middleware/common.go` 中的 CORS 中间件需要加入前端 origin。
- **模板漂移**：忘记执行 `npm run sync:templates` 会导致脚手架与 Skeleton 不一致；CI 会在 PR 中执行 `npm run sync:templates -- --check` 给出提示。

### 1.6 相关文档

- [架构设计总览](../plan/001-init-project.md)
- [从 Base 插件迁移指南](migration/base-to-skeleton.md)
- [CLI 模板同步脚本](../../scripts/template-sync-config.yaml)

## 2. Delegated（宿主）模式

Delegated 模式指插件被 PowerX 宿主拉起后运行在 iframe + process 沙箱内，前端与宿主共享登录态、后端通过宿主代理访问 Core API。本节保留原《delegated-mode》文档的关键内容，方便与 Standalone 场景对照。

### 2.1 角色与进程

- **宿主 PowerX Core**：监听 `8077`（默认），提供 `/api/v1/**`。宿主登录后会把 token 写入浏览器 `localStorage` 供插件复用。
- **插件后端（process 模式）**：由宿主进程拉起，暴露 `/_p/<pluginId>/api/v1/**` 接口，但不会直接处理宿主登录。
- **插件前端（iframe 模式）**：宿主在 Admin Router 中以 iframe 渲染 `/_p/<pluginId>/admin/**`，静态资源来自插件包 `web-admin/.output`。
- **CLI 构建/打包**：`px-plugin package` 默认注入 `POWERX_PROXY=1`、`baseURL=/_p/<pluginId>/admin/`、`apiBaseUrl=/api/v1` 等宿主配置。

### 2.2 关键配置

- **前端 baseURL / assets**
  - `app.baseURL = /_p/<pluginId>/admin/`
  - `buildAssetsDir = 'assets/'`，避免被宿主解析为 `/assets`。
- **前端 API 基址**
  - 当 `INSIDE_POWERX=true`：`runtimeConfig.public.apiBaseUrl = /api/v1`，并复用宿主 token。
  - 本地/独立：回退到 `/_p/<pluginId>/api/v1` 或 `NUXT_PUBLIC_API_BASE`。
- **模式标记**
  - `POWERX_PROXY=1`：构建/后端判断是否委托宿主。
  - `NUXT_PUBLIC_POWERX_PROXY=1`：前端 runtimeConfig 可见，用于桥接/脚本调整。
- **安装路径**
  - 宿主将包解压到 `backend/plugins/installed/<pluginId>/<version>/`，其中包含 `payload/web-admin/.output` 与 `payload/backend/bin/*`。

### 2.3 运行顺序

1. 宿主登录并写入 token。
2. 管理员安装插件包。
3. 宿主拉起插件后端进程，并在 Admin Router 中加载插件前端。
4. 前端初次访问会请求：
   - 静态资源：`/_p/<pluginId>/admin/assets/...`
   - 会话检测：`/api/v1/admin/auth/me/context`（复用宿主 token）
   - 业务接口：依据实现决定走 `/api/v1/**`（宿主）或 `/_p/<pluginId>/api/v1/**`（插件自带）

### 2.4 常见症状与排查

- **仍看到登录页**：确认 `auth/me/context` 请求命中了 `/api/v1/...` 且附带宿主 token。若请求落在 `/_p/...`，说明前端仍携带本地构建。
- **静态资源 404**：核对包内 `web-admin/.output/public/assets/*` 是否存在，并确保 baseURL 以 `/_p/<pluginId>/admin` 开头。
- **i18n baseDir not found**：构建时未包含 `web-admin/i18n/locales`，需要重新 `npm run build`。
- **插件接口返回 404**：记得区分宿主与插件 API。需要访问插件自有 API 时，请求 `/_p/<pluginId>/api/v1/**`。

### 2.5 构建/打包要点

- 在插件根目录执行 `px-plugin package --entry .`（内部会自动 `npm --prefix web-admin run build`）。
- 打包后检查：
  - `web-admin/.output/server/index.mjs` 与 `public/assets` 是否存在。
  - `web-admin/.output/server/chunks/nitro/nitro.mjs` 中 `baseURL=/_p/<pluginId>/admin/`、`apiBaseUrl=/api/v1`、`insidePowerX=true`。
  - `backend/bin/plugin`、`backend/bin/migrate` 等二进制是否被打入。

### 2.6 宿主/插件接口版本

- 宿主 API 版本由 PowerX 控制（默认 `/api/v1`），宿主模式下固定指向该前缀。
- 插件自有 API 可以使用 `/_p/<pluginId>/api/<version>` 独立演进，升级时记得同步后端路由与前端配置。

### 2.7 快速自检清单

- 浏览器 Network：`/api/v1/admin/auth/me/context` 200 且携带宿主 token。
- 静态资源请求形如 `/_p/<pluginId>/admin/assets/...`。
- `nitro.mjs` 中 `insidePowerX=true`、`apiBaseUrl=/api/v1`。
- 宿主日志出现 `/_p/:id/admin/*filepath` 命中记录。

若上述检查均正常但依旧提示登录，多半是浏览器缓存旧前端，可清缓存后重新安装。

### 2.8 跨域会话传递

当宿主与插件不在同一 host（例如 `localhost:3030` vs `127.0.0.1:8077`）时，浏览器不会共享 Cookie。PowerX Admin 可通过 `postMessage` 注入 token：

```ts
iframe.contentWindow?.postMessage(
  {
    source: 'powerx',
    type: 'auth-token',
    accessToken: '<宿主 access_token>',
    refreshToken: '<可选>',
    tokenType: 'Bearer',
    expiresIn: 3600, // 或 expiresAt
    pluginId: '<可选>'
  },
  'http://127.0.0.1:8077'
)
```

插件前端只接受来自可信 `origin` 且 `source === 'powerx'` 的消息，并把 token 写入自身 localStorage，再调用 `/api/v1/admin/auth/me/context` 即可共享登录态。

### 2.9 常见参考链接

- [Auth 集成说明](auth.md)
- [CLI 发布/热加载指南](go-cli-dev-watch.md)
- [CLI 入门教程](cli-plugin-tutorial.md)

## 3. 打包与环境变量注意事项

- 给宿主发布的资产必须保持 `POWERX_PROXY=1` 且不要附带任何 `NUXT_PUBLIC_API_BASE=http://localhost:8078`、`NUXT_DEV_*` 之类的本地值，否则构建产物会把 API 指向你的本地端口，安装到宿主后无法加载。
- 本地/Standalone 调试可以在 `.env` 中覆盖这些变量，但在执行 `npm run build` 之前请确认环境已清理。推荐将宿主构建命令放在脚本中显式设置：`POWERX_PROXY=1 npm run build`。
- `px-plugin package` 默认注入正确的宿主配置；若你在构建前修改 `.env`，CLI 不会覆盖这些值，因此务必在包发布前检查 `web-admin/.output/server/chunks/nitro/nitro.mjs` 中的 `baseURL` 与 `apiBaseUrl`。
