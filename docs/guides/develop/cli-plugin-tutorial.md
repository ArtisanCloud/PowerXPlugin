# 使用 CLI 生成并运行插件骨架

本教程演示如何构建 `px-plugin` CLI、生成新的插件脚手架，并启动生成项目的后端与管理端。目标是快速验证 CLI 模板是否与仓库 Skeleton 保持一致。

## 适用场景

- 在独立目录内创建新的插件工程（后端 + 管理端）。
- 验证 CLI 是否正确写入模板代码、契约文件与 `plugin.yaml`。
- 为后续开发自定义插件或演练发布流程做准备。

## 前置条件

- 完成《PowerXPlugin Standalone 启动教程》中的依赖准备。
- `go` 与 `npm` 命令均可在当前终端使用。
- `$(go env GOPATH)/bin` 已加入 `PATH`（或你自定义的安装目录已在 PATH 中）。

除 Step 3 需要切换到你自定义的插件目录外，其余命令默认在仓库根目录执行。

## Step 1. 同步模板与工作区依赖

```bash
cd /path/to/PowerXPlugin             # 仓库根目录
npm run sync:templates -- --check    # 可选：仅检查差异
npm run sync:templates -- --verbose  # 同步模板并输出写入文件
# npm install --workspaces           # 可选：安装 framework-admin / framework-client
# go work sync                       # 让 go.work 中的本地模块依赖立即生效
```

> 说明：只有在需要直接引用仓库内的前端框架源码时，才执行 `npm install --workspaces`。若使用发布版本，可跳过此命令，并在生成后的项目中将 `@artisan-cloud/plugin-framework-*` 调整为目标版本号。

执行 `go work sync` 能确保 `framework/` 模块使用最新的 `module github.com/ArtisanCloud/PowerXPlugin/framework` 声明，避免由于缓存旧模块路径而导致的 `module declares its path` 报错。如果你使用全新 GOPROXY 版本或外部仓库发布版，可视情况省略。

完成同步后再继续构建 CLI／生成插件，可确保 Skeleton、scaffold 与 CLI 模板保持一致。

> 如果你计划发布或复用 Go 框架模块，请使用 `git tag framework/v0.0.1-alpha && git push origin framework/v0.0.1-alpha`（或更高版本号）在仓库根目录打 tag，供外部 `go get github.com/ArtisanCloud/PowerXPlugin/framework@v0.0.1-alpha` 直接引用。

### 发布 `@artisan-cloud/plugin-framework-*` 到 npm

框架前端包需要先发布到 npm，再由 CLI 生成的脚手架引用：

```bash
# 1. 在 framework/frontend/nuxt/framework-admin
npm version 0.0.1-alpha
npm publish --access public

# 2. 在 framework/frontend/nuxt/framework-client
npm version 0.0.1-alpha
npm publish --access public
```

> 版本号需与 CLI 模板中的默认值（当前 `^0.0.1-alpha`）保持一致。发布前请确认 `package.json` 的 `files` 列表完整可用，并已通过 `npm run lint && npm run build`。

## Step 2. 构建 px-plugin CLI

```bash
cd tools/cli
go install ./cmd/px-plugin
px-plugin --version  # 验证可执行文件已安装并输出版本信息
```

执行完成后，`px-plugin` 可执行文件将安装到 `$(go env GOPATH)/bin`。请确保该目录已加入 `PATH`，之后即可在任意路径直接调用 `px-plugin`。

> 默认输出形如 `px-plugin version dev (commit 1a2b3c4)`，其中 `dev` 表示未设置自定义版本。若想在发布时带上明确版本号，可在安装命令中附加：
>
> ```bash
> go install -ldflags "-X main.version=v0.3.0" ./cmd/px-plugin
> ```
>
> 这样再次运行 `px-plugin --version` 时就会显示 `px-plugin version v0.3.0 (commit 1a2b3c4)`。

## Step 3. 生成插件骨架

选择一个新的插件 ID（推荐反向域名），并指定模板/组织信息。以下示例使用 `com.powerx.helloworld`，同时通过 `--template` 选择 `fullstack-go-nuxt`，CLI 会读取 `packages/template-registry/index.yaml`，验证模板版本、运行时要求与依赖锁定：

```bash
cd {your}/{customer}/{path}
mkdir -p plugins
cd plugins
px-plugin init com.powerx.helloworld \
  --template fullstack-go-nuxt \
  --module github.com/demo/acme-plugin \
  --org demo-team
```

命令执行期间会触发：

1. 模板渲染：复制 `scaffold/templates` 中的 backend/frontend/manifest 模板。
2. Bootstrap 校验：调用 `POST /internal/plugins/bootstrap/validate`（`framework/backend/go/runtime/bootstrap/service/bootstrap_handler.go`）生成 Git 注册建议、`publish.yml`、`reports/sbom.json`。
3. 合规扫描：产出 `publish.yml`、`reports/sbom.json` 并记录 `validationId`，CLI 会在输出中显示。

> 模板选择：
> - `--template fullstack-go-nuxt`（默认）包含 Gin + Nuxt Web Admin。
> - `--template backend-go-lite` 仅生成后端骨架，适合纯 API 插件。
> - 可在 `packages/template-registry/index.yaml` 查看最小运行时和 Hook 列表。

CLI 会在 `plugins/com.powerx.helloworld` 下生成完整项目，并输出创建的文件列表。常见目录包括：

- `backend/`：Go 后端骨架，引用 `github.com/ArtisanCloud/PowerXPlugin/framework`
- `web-admin/`：Nuxt 管理端骨架，引用 `@artisan-cloud/plugin-framework-admin`
- `public/images/logo-s.png`：导航左上角默认 Logo，`AppNavbar` 会引用此文件
- `docs/contracts/`：嵌入的 Manifest/RBAC/OpenAPI 契约
- `plugin.yaml`：插件基础元数据（ID、版本、前后端堆栈）

若目录已存在且不为空，可使用 `--force` 覆盖（谨慎操作）。完成后请在仓库根目录执行 `git init`/`git remote add`，或者根据 bootstrap 响应中的 `gitRepository.url` 直接创建远程仓库。

## Step 4. 安装生成项目的依赖

进入生成的工程目录，依次完成后端、前端及数据库准备。以下步骤可根据需要选择。

### 4.1 后端依赖

```bash
cd com.powerx.helloworld/backend
go mod tidy
cd ..
```

### 4.2 框架依赖（可选）

默认依赖 `github.com/ArtisanCloud/PowerXPlugin/framework {{ .FrameworkVersion }}`（当前为 `v0.0.1-alpha`）。如需直接引用本仓库源码，可在 `backend/go.mod` 添加：

```
replace github.com/ArtisanCloud/PowerXPlugin/framework => /path/to/PowerXPlugin/framework
```

将路径替换为你本地 `framework/` 的绝对目录，再执行 `go mod tidy`。

### 4.3 管理端依赖

```bash
cd web-admin
npm install
cd ..
```

> CLI 默认写入 `@artisan-cloud/plugin-framework-admin` / `@artisan-cloud/plugin-framework-client` 的已发布版本（当前建议 `^0.0.1-alpha`）。若你在 monorepo 中调试 Layer，需要引用本地源码，可在执行 `px-plugin init` 前设置 `POWERXPLUGIN_USE_LOCAL_FRONTEND=1`，随后重新运行 `npm install`。

### 4.4 数据库配置

- 默认使用内存 SQLite，无需额外配置。
- 若要使用文件 SQLite 或 Postgres：
  1. `cp backend/etc/config.example.yaml backend/etc/config.yaml`
  2. 修改 `database.driver`（`sqlite` 或 `postgres`）与 `dsn/schema`
  3. 初始化表结构：
     ```bash
     cd backend
     go run ./cmd/database/main.go migrate
     cd ..
     ```
  4. 如需重置或灌入示例数据，可执行 `go run ./cmd/database/main.go setup` / `seed` / `refresh`。

脚手架会在 `backend/` 下生成 `etc/` 与 `.gitignore`，默认忽略本地配置文件。
## Step 5. 启动生成项目的后端

```bash
cd backend
go run ./cmd/plugin # 注意包含 ./ 指向当前目录下的 main 包
```

CLI 模板与 Skeleton 一致，默认监听 `:8078`。可以重复使用 Step 3 的 `curl http://localhost:8078/api/v1/ping` 验证输出。

### 常见问题

- 若提示 `module declares its path`，说明 `backend/go.mod` 中的 module 名称需要与你的仓库路径一致，可根据实际情况调整。
- 如需自定义端口或环境，与 Skeleton 相同设置 `POWERX_LISTEN` / `POWERX_ENV`。

## Step 6. 启动生成项目的管理端

在另一个终端窗口执行：

```bash
cd {your}/{customer}/{path}/plugins/com.powerx.helloworld/web-admin
npm run dev
```

生成的 `nuxt.config.ts` 同样默认使用 `port: 3031`（HMR 显式设为 `ws://localhost:24731`），无须额外参数即可与 Skeleton 对齐。若本地端口被占用，可通过 `npm run dev -- --port <custom> --hmr-port <custom-hmr>` 覆盖。

访问：

```
http://localhost:3031/_p/com.powerx.helloworld/admin/
```

应看到 Starter 页面，标题会根据你的插件 ID 自动生成，例如 “Powerx Helloworld Plugin”。

> 默认首页 `/` 没有路由，会在终端输出 `No match found for location with path "/"` 的警告。请直接访问上述 plugin 路径。

> Nuxt 导航栏左上角默认展示 `public/images/logo-s.png`。你可以用自己的图替换该文件（保持文件名不变），或在 `app/components/AppNavbar.vue` 中调整引用。

## Step 7. 验证契约与元数据

CLI 会自动写入契约文件与 `plugin.yaml`。建议检查以下内容：

- `plugin.yaml` 中的 `backend`, `frontend`, `version` 字段是否存在。
- `docs/contracts/manifest.json`、`rbac.json` 与仓库 `docs/contracts` 结构一致。
- `README.md` 包含下一步提示与 CLI 实验命令说明。

必要时，可将生成项目与 `examples/starter/` 做差异对比，确认模板未意外 drift。

## Step 8. 运行 `px-plugin doctor`

在插件根目录执行：

```bash
px-plugin doctor --fix
```

该命令会输出 `.doctor/report.json`，校验以下内容：

1. Node/Go 版本是否满足 `>=18` / `>=1.24`。
2. `backend/go.mod`、`web-admin/package.json`、`web-admin/node_modules/` 是否齐全（`--fix` 会自动执行 `go mod tidy` / `npm install`）。
3. 必需 Feature Flag（`PX_PLUGIN_SCAFFOLD_V2`, `plugin-import-audit`, `gitops-bootstrap` 等）是否已配置。

报告建议随同 `publish.yml`、`reports/sbom.json` 一并提交到代码评审或合规工单。

## Step 9. 使用 dev --watch 联调 PowerX

在进入宿主模拟器或发布流程之前，建议先用 `px-plugin dev --watch` 与 PowerX Dev API 建立热加载会话，直接验证后端、前端与数据库行为。最基本的流程如下：

```bash
cd plugins/com.powerx.helloworld
px-plugin dev --watch \
  --entry . \
  --tenant demo-tenant \
  --dev-api https://dev-api.powerx.local \
  --logs-level info
```

该命令会在 Dev API 注册 `sessionId` 并监听本地文件变更，约 250ms 内完成增量打包与热更新，可通过 `px-plugin dev --list-sessions` / `--resume` / `--stop` 管理各个会话。完整的前置条件、证书配置与常见问题请查看《docs/guides/develop/dev-watch.md》，若需要更深入的原理说明可参考发布指南中的《docs/guides/publish/go-cli-dev-watch.md》。

联调完成后再继续执行宿主模拟器、沙箱或发布操作，可以显著减少环境问题带来的干扰。

## Step 10. 宿主模拟器与沙箱验证

完成基础开发后，可以利用 Phase 11 的链路在本地模拟宿主、执行沙箱测试并生成调试报告：

1. **启动宿主模拟器**
   ```bash
   px-plugin host start --mock \
     --plugin com.powerx.helloworld \
     --runtime-version latest \
     --tenant demo-tenant
   ```
   命令会返回 `sessionId`、`endpoint` 与日志地址；可通过 `px-plugin host status --session <id>` 与 `px-plugin host logs --session <id>` 查看运行情况，底层对应 API 为 `POST/GET /internal/dev/hosts/sessions`。

2. **执行沙箱验证**
   ```bash
   px-plugin sandbox deploy \
     --host-session host-123 \
     --dataset demo \
     --test-plan hotload-suite
   ```
   CLI 会调用 `POST /internal/dev/sandbox/deploy`，输出 `validationId` 供 Marketplace 审核或自测记录。

3. **上传调试报告**
   ```bash
   px-plugin debug report \
     --session host-123 \
     --input ./reports/debug.json
   ```
   触发 `POST /internal/dev/debug/report`，记录 `debug.report.generate_ms` 指标并将脱敏报告同步到工单系统。

调试完成后，可使用 `px-plugin host stop --session <id>` 释放资源。

## Step 11. 第三方源码导入（可选）

若需要将外部模板或客户源码导入到插件仓库，请在执行 `px-plugin init` 之后运行：

```bash
px-plugin import --source ./vendor/crm.tar.gz \
  --type tarball \
  --provider github.com \
  --license MIT
```

CLI 会读取 `config/compliance/external_source_policy.yaml`，根据域名、包体大小、许可证以及校验和要求进行评估，并生成 `./.compliance/import-report.json`。若命中 denylist 或超过阈值，命令会提示需要人工审批的联系人，同时把事件写入 `plugin-import-audit` Webhook（由后端 `bootstrap_handler` 记录）。

> 建议将 import/doctor 报告附到变更工单，方便 Marketplace 审核人员追踪第三方源码的来源、许可证和扫描结果。

## Step 12. 清理或复用工程

- 不再需要时，可删除 `plugins/com.powerx.helloworld` 目录。
- 若计划长期开发，请更新 `backend/go.mod` 的 module 名称并提交到独立仓库。
- 关注 `docs/release.md` 获取后续 `package/dist/publish` 命令的最新状态。

---

现在你已经通过 CLI 生成并运行了一个新的插件项目。下一步可以拓展业务路由、添加前端页面，或结合测试脚本执行 `make test-smoke` 验证端到端流程。
