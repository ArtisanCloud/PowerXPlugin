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
```

> 说明：只有在需要直接引用仓库内的前端框架源码时，才执行 `npm install --workspaces`。若使用发布版本，可跳过此命令，并在生成后的项目中将 `@powerx-plugin/framework-*` 调整为目标版本号。

完成同步后再继续构建 CLI／生成插件，可确保 Skeleton、scaffold 与 CLI 模板保持一致。

> 如果你计划发布或复用 Go 框架模块，请使用 `git tag framework/v0.0.0-alpha && git push origin framework/v0.0.0-alpha`（或更高版本号）在仓库根目录打 tag，供外部 `go get github.com/ArtisanCloud/PowerXPlugin/framework@v0.0.0-alpha` 直接引用。

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

选择一个新的插件 ID（推荐反向域名）。以下示例使用 `com.powerx.helloworld`，并在自定义的 `plugins/` 目录中创建项目（请将 `{your}/{customer}/{path}` 替换为你实际的工作目录）：

```bash
cd {your}/{customer}/{path}
mkdir -p plugins
cd plugins
px-plugin init com.powerx.helloworld
```

CLI 会在 `plugins/com.powerx.helloworld` 下生成完整项目，并输出创建的文件列表。常见目录包括：

- `backend/`：Go 后端骨架，引用 `github.com/ArtisanCloud/PowerXPlugin/framework`
- `web-admin/`：Nuxt 管理端骨架，引用 `@powerx-plugin/framework-admin`
- `docs/contracts/`：嵌入的 Manifest/RBAC/OpenAPI 契约
- `plugin.yaml`：插件基础元数据（ID、版本、前后端堆栈）

若目录已存在且不为空，可使用 `--force` 覆盖（谨慎操作）。

## Step 4. 安装生成项目的依赖

进入生成的工程目录，依次完成后端、前端及数据库准备。以下步骤可根据需要选择。

### 4.1 后端依赖

```bash
cd com.powerx.helloworld/backend
go mod tidy
cd ..
```

### 4.2 框架依赖（可选）

默认依赖 `github.com/ArtisanCloud/PowerXPlugin/framework {{ .FrameworkVersion }}`（当前为 `v0.0.0-alpha`）。如需直接引用本仓库源码，可在 `backend/go.mod` 添加：

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

> 建议在生成插件前，先在仓库根目录执行 `npm install --workspaces`，确保 `framework/frontend/nuxt/framework-admin|client` 已安装。CLI 会自动引用这些目录；若要独立发布，可改为正式版本号或私有 registry。

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

## Step 6. 验证契约与元数据

CLI 会自动写入契约文件与 `plugin.yaml`。建议检查以下内容：

- `plugin.yaml` 中的 `backend`, `frontend`, `version` 字段是否存在。
- `docs/contracts/manifest.json`、`rbac.json` 与仓库 `docs/contracts` 结构一致。
- `README.md` 包含下一步提示与 CLI 实验命令说明。

必要时，可将生成项目与 `examples/starter/` 做差异对比，确认模板未意外 drift。

## Step 7. 清理或复用工程

- 不再需要时，可删除 `plugins/com.powerx.helloworld` 目录。
- 若计划长期开发，请更新 `backend/go.mod` 的 module 名称并提交到独立仓库。
- 关注 `docs/release.md` 获取后续 `package/dist/publish` 命令的最新状态。

---

现在你已经通过 CLI 生成并运行了一个新的插件项目。下一步可以拓展业务路由、添加前端页面，或结合测试脚本执行 `make test-smoke` 验证端到端流程。
