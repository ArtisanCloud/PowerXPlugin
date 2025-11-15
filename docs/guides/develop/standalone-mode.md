
# Standalone 模式运行指南

本文说明如何在宿主之外运行 Skeleton 插件，并阐释后端分层、扩展点与启动流程。

## 1. 目录结构与分层

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

## 2. 启动流程

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

## 3. 本地运行

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
#    如果需要单独执行，可替换为：
#    go run ./cmd/database/main.go migrate
#    go run ./cmd/database/main.go seed

# 3. 启动后端（默认使用 SQLite 文件 powerxplugin.db 保存数据）
go run ./cmd/plugin

# 4. 访问健康检查
curl http://127.0.0.1:8087/healthz

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

- 后端 HTTP: `8087`（可通过 `PORT` 环境变量覆盖）
- 后端 gRPC: `8079`（通过 `POWERX_GRPC_PORT` 覆盖）
- 管理端 Nuxt: 默认 `3031`（冲突时自动寻找可用端口）

> `skeleton/backend/etc/` 目录内包含示例 `config.yaml` 与 `security_baseline.yaml`。默认 DSN 为 `file:../.cache/powerxplugin.db?cache=shared&_fk=1`，Loader 会把它解析成相对于 `config.yaml` 的路径，因此无论在仓库根目录还是 `skeleton/backend` 执行命令，最终都会落在 `skeleton/.cache/` 下；若希望把文件放到仓库根目录，也可以把 DSN 改成 `file:../../.cache/powerxplugin.db?cache=shared&_fk=1` 或通过 `POWERX_DB_DSN` 环境变量覆盖。若改为纯内存 DSN（如 `file::memory:?cache=shared`），请在同一进程内连续执行 `migrate` 与 `seed`。示例配置同时关闭了 Marketplace 推荐和续费提醒的后台任务，避免在空表上触发告警。
>
> Loader 在解析 SQLite DSN 时会自动创建目标目录，无需手动 `mkdir`。如果路径中包含 `../.cache`，会基于配置文件目录进行展开。
>
> 也可以将 `runtime.run_migrate` 设为 `true` 或在启动命令前加 `POWERX_RUN_MIGRATE=true`，这样服务启动时会自动运行迁移。种子数据仍需手动执行 `go run ./cmd/database/main.go seed`。

> 默认导航栏左上角引用 `public/images/logo-s.png`。如果要替换 Logo，可在生成项目的 `public/images` 目录中用同名文件覆盖，或调整 `app/components/AppNavbar.vue` 中的 `<img>` 引用。

## 4. 扩展点示例：新增模板审批接口

1. **领域模型**：在 `internal/entity/models/template` 新增 `approval.go`，描述审批状态。
2. **仓储接口**：在 `internal/entity/repository/template` 添加 `approval_repository.go`，定义 `FindPending`、`Approve` 等方法，并提供最小内存实现。
3. **应用服务**：于 `internal/services/template/approval_service.go` 编排审批流程，处理幂等与事件上报。
4. **传输层 Handler**：在 `internal/transport/http/admin/templates/approval_handler.go` 实现接口，`POST /templates/:id/approve` 调用服务层。
5. **路由注册**：更新 `internal/router/templates.go` 或所在域的路由，将新 Handler 挂载到 `/api/v1/templates` 子路由。
6. **Manifest 更新**：在 `internal/manifestx/manifest.go` 增加菜单或权限项，并同步前端导航。
7. **同步模板**：完成修改后执行 `npm run sync:templates`，确保 Scaffold/CLI 模板保持一致。

## 5. 常见问题

- **403 或 401**：确认 `POWERX_SECURITY_*` 安全上下文配置正确；独立模式下可在配置中关闭严格模式。
- **CORS 报错**：`internal/middleware/common.go` 中的 CORS 中间件需要加入前端 origin。
- **模板漂移**：忘记执行 `npm run sync:templates` 会导致脚手架与 Skeleton 不一致；CI 会在 PR 中执行 `npm run sync:templates -- --check` 给出提示。

## 6. 相关文档

- [架构设计总览](../plan/001-init-project.md)
- [从 Base 插件迁移指南](migration/base-to-skeleton.md)
- [CLI 模板同步脚本](../../scripts/template-sync-config.yaml)
