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

> 重要：不要只执行 `--check` 就直接去 `go install`。`px-plugin` 的 Go CLI 会把 `tools/cli/internal/templates/data/**` 作为内嵌模板打进二进制；如果你没有先执行同步（或同步后没有重新编译 CLI），就可能出现 “CLI 内嵌模板与 skeleton 不一致” 的问题（典型表现是新项目 `npm install` 直接 `MODULE_NOT_FOUND`）。

> 说明：只有在需要直接引用仓库内的前端框架源码时，才执行 `npm install --workspaces`。若使用发布版本，可跳过此命令，并在生成后的项目中将 `@artisan-cloud/plugin-framework-*` 调整为目标版本号。

执行 `go work sync` 能确保 `framework/backend/go` 模块使用最新的 `module github.com/ArtisanCloud/PowerXPlugin/framework/backend/go` 声明，避免由于缓存旧模块路径而导致的 `module declares its path` 报错。如果你使用全新 GOPROXY 版本或外部仓库发布版，可视情况省略。

完成同步后再继续构建 CLI／生成插件，可确保 Skeleton、scaffold 与 CLI 模板保持一致。

> 如果你计划发布或复用 Go 框架模块，请使用 `git tag framework/backend/go/v0.0.1-alpha && git push origin framework/backend/go/v0.0.1-alpha`（或更高版本号）在仓库根目录打 tag，供外部 `go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.1-alpha` 直接引用。

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
> 这样再次运行 `px-plugin --version` 时就会显示 `px-plugin version v0.3.0 (commit 1a2b3c4)`。注意该版本号仅用于展示，不代表模板一定已同步；模板是否生效以 “生成项目后文件是否存在” 为准。

### Step 2.1（推荐）CLI 模板 Smoke Check

在继续生成正式项目之前，先做一次最小验证，确保内嵌模板包含 Nuxt Web Admin 的脚本文件：

```bash
TMP_DIR="$(mktemp -d)"
px-plugin init --force --directory "$TMP_DIR" com.powerx.smoke.template
test -f "$TMP_DIR/web-admin/scripts/postinstall-lightningcss.mjs"
echo "OK: CLI templates look good"
```

## Step 2.5  获取 Skeleton Gateway 凭证（可选）

在本仓库的 Skeleton 或你生成的插件项目内调试 PowerX 通用能力时，需要先获取 Dev Gateway 的 Tool Token。CLI 提供 `login` 命令，默认把凭证写入 `~/.powerx/credentials`，你可再同步到 `.env.local`：

```bash
cd /path/to/PowerXPlugin/skeleton
px-plugin login --manifest ./plugin.yaml \
  --base-url https://gateway.powerx.dev \
  --tenant demo-tenant-uuid

# 将凭证写入 Skeleton 的 .env.local（示例脚本）
cat <<'EOF' > .env.local
PX_GATEWAY_BASE_URL=https://gateway.powerx.dev/_tenant
PX_TOOL_TOKEN=$(jq -r '.credentials.toolToken' ~/.powerx/credentials)
EOF
```

说明：

1. **宿主模式**无需执行 `login`，运维会在部署清单中注入 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`；租户由 token `tid` 自动推导。
2. `px-plugin login` 会根据 manifest 中的插件 ID、所需能力向 Dev Gateway 发起 STS 交换，请确保你拥有对应环境的 Token/Key（若需要，请联系平台团队）。
3. `.env.local` 会被 `skeleton/web-admin/nuxt` 与 `skeleton/backend/go-gin` 自动载入；若切换到其他栈（如 `python-fastapi` / `next`），需要在对应目录内确保加载逻辑一致。你也可以使用 `scripts/capabilities/run-from-package.mjs --mode skeleton`，它会优先读取 `.env.local` 的 `PX_*` 变量。
4. 当 Token 即将过期或失效时，重新执行 `px-plugin login` 并覆盖 `.env.local` 即可。

## Step 3. 生成插件骨架

选择一个新的插件 ID（推荐反向域名），并指定模板/组织信息。以下示例使用 `com.powerx.helloworld`，CLI 会读取 `packages/template-registry/index.yaml`，验证模板版本、运行时要求与依赖锁定：

```bash
cd {your}/{customer}/{path}
mkdir -p plugins
cd plugins
px-plugin init --force \
  --backend go-gin \
  --admin nuxt \
  --module github.com/demo/acme-plugin \
  com.powerx.helloworld
```

命令执行期间会触发：

1. 模板渲染：复制 `scaffold/templates` 中的 backend/frontend/manifest 模板。
2. 写入 `docs/contracts/`、`publish.yml`、`reports/sbom.json` 等基础产物。

> 模板选择：
> - `--backend go-gin` + `--admin nuxt`（默认）包含 Gin + Nuxt Web Admin。
> - `--backend python-fastapi` + `--admin nuxt`：FastAPI + Nuxt（后端为最小空壳，前端为完整 Nuxt 管理端）。
> - `--backend go-gin` + `--admin next`：Gin + Next（当前为占位模板，功能不完整）。
> - `--backend python-fastapi` + `--admin next`：FastAPI + Next（当前为占位模板，功能不完整）。
> - `--install-deps` 可选：自动执行 `go mod tidy` 与 `npm install`（联网环境下更方便，离线/内网请手动安装）。

> 说明：
> - 当前 CLI `--admin` 支持 `nuxt` 与 `next`，其中 `next` 为占位模板；`react` 尚未对外开放。
> - 如果你希望默认栈变为 **Gin + Next**，需要同步调整 CLI 默认值与模板注册表，而不仅是文档。

CLI 会在 `plugins/com.powerx.helloworld` 下生成完整项目，并输出创建的文件列表。常见目录包括：

- `backend/`：Go 后端骨架，引用 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`
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

默认依赖 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go {{ .FrameworkVersion }}`（当前为 `v0.0.1-alpha`）。如需直接引用本仓库源码，可在 `backend/go.mod` 添加：

```
replace github.com/ArtisanCloud/PowerXPlugin/framework/backend/go => /path/to/PowerXPlugin/framework/backend/go
```

将路径替换为你本地 `framework/` 的绝对目录，再执行 `go mod tidy`。

### 4.3 管理端依赖

```bash
cd web-admin
npm install
cd ..
```

> CLI 默认写入 `@artisan-cloud/plugin-framework-admin` / `@artisan-cloud/plugin-framework-client` 的已发布版本（当前建议 `^0.0.1-alpha`）。若你在 monorepo 中调试 Layer，需要引用本地源码，可在执行 `px-plugin init` 前设置 `POWERXPLUGIN_USE_LOCAL_FRONTEND=1`，随后重新运行 `npm install`。
>
> ⚠️ **构建宿主包前务必清理本地联调环境变量**：`npm run build` 会把 `.env`（或 shell 环境）中的 `POWERX_PROXY`、`NUXT_PUBLIC_API_BASE=http://localhost:8078`、`NUXT_DEV_API_PROXY` 等值烘焙进产物。入驻 PowerX 宿主时，保持默认（仅 `POWERX_PROXY=1`）即可让前端自动使用 `/_p/<pluginId>/api/v1` 与宿主注入的 API。若仍保留指向本地的变量，部署后会直接访问你的 8078 实例或触发 CSP，导致宿主环境无法登录。

### 4.4 数据库配置

- 默认开发建议使用 Postgres（更接近生产；SQLite 仅保留最小可跑通子集）。
- 若要使用 Postgres（推荐）或 SQLite：
  1. 复制配置文件（注意路径取决于你当前所在目录）：
     - 在插件项目根目录执行：`cp backend/etc/config.example.yaml backend/etc/config.yaml`
     - 若你已 `cd backend`：`cp etc/config.example.yaml etc/config.yaml`
  2. 修改 `database.driver` 与 `dsn/schema`
     - **Postgres（推荐）**：确保 DSN 对应的数据库存在（例如 `com_powerx_plugin_base`）；`schema` 会自动创建（常用 `public`）。
     - **SQLite（仅最小能力集）**：可用于快速验证启动/IAM/MCP，但不保证 marketplace/ops 等全量表结构可用。
     - 若你尚未配置 Dev Gateway（`px-plugin login` / `PX_*` Token），请保持 `gateway.base_url` / `gateway.tool_token` / `gateway.tenant_uuid` 为空（否则会触发配置校验并阻塞启动）。
  3. 初始化本地开发所需数据（推荐，包含 migrate + seed 本地管理员账号）：
     ```bash
     cd backend
     go run ./cmd/database/main.go setup
     cd ..
     ```
     默认本地管理员账号（可用环境变量覆盖）：
     - Email：`admin@local.test`
     - Password：`S3cret!!`
     - 覆盖：`PLUGIN_IAM_ADMIN_EMAIL` / `PLUGIN_IAM_ADMIN_PASSWORD`
  4. 仅初始化表结构（不灌入默认管理员/权限）时，再使用 `go run ./cmd/database/main.go migrate`。
  5. 其它常用命令：`seed`（灌入示例数据）、`refresh`（重置并重建）。

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

生成的 `nuxt.config.ts` 同样默认使用 `port: 3131`（HMR 显式设为 `ws://localhost:24731`），无须额外参数即可与 Skeleton 对齐。若本地端口被占用，可通过 `npm run dev -- --port <custom> --hmr-port <custom-hmr>` 覆盖。

访问：

```
http://localhost:3131/_p/com.powerx.helloworld/admin/
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

## Step 9. 下一步：发布与热加载

本教程完成后，通常会继续：

- 使用 `px-plugin package`/`publish` 生成并上传 artefacts。
- 通过 `px-plugin dev --watch` 进行 Dev API 热加载，或配合 `host`/`sandbox` 管道验证宿主模拟器。
- 借助 `px-plugin import` 审核第三方源码。

这些命令的完整示例、参数说明与排障建议已汇总在《docs/guides/develop/go-cli-dev-watch.md》，请以该文档为准，以免在多处重复维护。

## Step 10. 清理或复用工程

- 不再需要时，可删除 `plugins/com.powerx.helloworld` 目录。
- 若计划长期开发，请更新 `backend/go.mod` 的 module 名称并提交到独立仓库。
- 关注 `docs/release.md` 获取后续 `package/dist/publish` 命令的最新状态。

---

现在你已经通过 CLI 生成并运行了一个新的插件项目。下一步可以拓展业务路由、添加前端页面，或结合测试脚本执行 `make test-smoke` 验证端到端流程。
