# 使用 `px-plugin` 初始化、启动并本地安装插件

本文是 `px-plugin` 的主流程文档：从 CLI 构建、`init` 项目、数据库初始化、本地启动，到安装到 PowerX。

## 0) 前置条件

- 命令默认在仓库根目录执行：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin`
- 已安装 Go 1.24+、Node.js 18+、npm 9+、PostgreSQL（或 SQLite）

## 1) 构建并安装 `px-plugin`

推荐（全局可直接调用）。正式发布或团队共享时，使用已确认的 CLI tag：

```bash
make install-px-plugin PX_PLUGIN_CLI_VERSION=v1.0.1
hash -r
px-plugin --version
```

本地验证当前 checkout 时，推荐用 `git describe` 生成可追踪版本戳：

```bash
make install-px-plugin PX_PLUGIN_CLI_VERSION="$(git describe --tags --dirty --always)"
hash -r
px-plugin --version
```

什么时候需要更新 `PX_PLUGIN_CLI_VERSION`：

- `tools/cli` 的命令行为有变化
- `scaffold/templates` 或 `tools/cli/internal/templates/data` 的脚手架模板有变化
- `docs/contracts` 或 `tools/cli/internal/contracts/data` 的契约有变化
- 要把本地安装结果交给团队复用或用于发布链路

只是重复安装同一份代码时不需要升版本；只做本地临时调试时也可以不传，此时 `px-plugin --version` 会显示 `dev` 和当前构建信息。

仅本地构建（不覆盖全局）：

```bash
make build-px-plugin
./bin/px-plugin --version
```

## 2) 初始化插件项目

先区分两个名字，避免把插件运行时标识和本地目录混在一起：

- 项目目录名：本地文件夹名，例如 `com.powerx.plugins.hello-world`
- 插件 ID：写入 `plugin.yaml` 的 `id`，也是运行时注册、路由、能力归属使用的标识，例如 `com.powerx.plugins.hello-world`

推荐用交互模式初始化。命令里的参数会作为 `Plugin ID` 的默认值；如果不修改 `Target directory`，项目目录名也会默认使用同一个值：

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins
px-plugin init com.powerx.plugins.hello-world
```

交互过程示例：

```text
Entering init guide (interactive mode).
Plugin ID [com.powerx.plugins.hello-world]:
Backend:
  1) go-gin (default)
  2) python-fastapi
Choose number [1]:
Admin frontend:
  1) nuxt (default)
  2) next
Choose number [1]:
Backend port [8078]:
Frontend port [3131]:
GitHub org/user [your-org]: ArtisanCloud
Module root [github.com/ArtisanCloud/com.powerx.plugins.hello-world]:
Target directory [com.powerx.plugins.hello-world]:
Install dependencies now (go mod tidy + npm install) [y/n] [y]:
```

这个例子最终得到：

- `plugin.yaml` 里的 `id`: `com.powerx.plugins.hello-world`
- 项目目录：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/com.powerx.plugins.hello-world`

当前 CLI 校验规则如下：

```text
^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$
```

交互向导会确认：
- backend / admin
- module root（默认 `github.com/<org>/<plugin-id>`）
- target directory（项目目录名；默认跟随最终确认的插件 ID）
- 是否安装依赖（`go mod tidy` + `npm install`）
- 是否复制配置（`*.example -> .local/.yaml`）
- 是否 `git init`

## 3) 配置数据库

`init` 选择 `Create local config files` 后会自动生成：

- `backend/etc/config.yaml`
- `backend/.env.local`（go-gin）或 `backend/python-fastapi/.env.local`（python-fastapi）
- `web-admin/.env.local`（nuxt）或 `web-admin/next/.env.local`（next）

以 Postgres 为例，先建库（按你的库名）：

```bash
psql "postgres://<user>:<pass>@127.0.0.1:5432/postgres?sslmode=disable" \
  -c "CREATE DATABASE com_powerx_plugin_hello_world;"
```

然后在 `backend/etc/config.yaml` 对齐：

- `database.dsn`
- `database.schema`

## 4) migrate / seed / setup

在插件项目根目录执行（推荐）：

```bash
make migrate   # 仅迁移
make seed      # 仅种子
make setup-db  # migrate + seed（推荐首跑）
```

Go backend 等价命令：

```bash
cd backend
go run ./cmd/database/main.go migrate
go run ./cmd/database/main.go seed
go run ./cmd/database/main.go setup
```

## 5) 启动插件项目

后端：

```bash
cd backend
go run ./cmd/plugin
```

前端（Nuxt 默认）：

```bash
cd web-admin
npm install
npm run dev
```

Nuxt 常用环境变量（`web-admin/.env.local`）：

- `POWERX_PROXY=0|1`
- `NUXT_PUBLIC_API_BASE`
- `NUXT_PUBLIC_API_PREFIX=/api/v1`
- `NUXT_DEV_API_PROXY`
- `NUXT_DEV_WS_PROXY`
- `NUXT_PUBLIC_POWERX_CORE_BASE`（可选）
- `NUXT_PUBLIC_POWERX_PROVIDER_MODE=local|delegated`（可选）

前端（Next）：

```bash
cd web-admin/next
npm install
npm run dev
```

## 6) 本地安装到 PowerX

当你要验证“真实安装链路”而不是热更新时，直接转到：

- `docs/guides/publish/local-install.md`

`local-install.md` 负责完整的 `dist/zip/.pxp` 与 `POST /admin/plugins/install/local` 流程。

## 7) 权限声明检查

新生成项目的 `plugin.yaml` 必须按 PowerX 权限声明规范维护顶层 `permissions[]`。详细规则见：

- `docs/guides/plugin_release/permission_declaration.md`
- PowerX Core 权威规范：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/PowerX/docs/guides/plugin_release/permission_declaration.md`

本地检查：

```bash
make plugin-permission-declaration-check
make dist
```

`make dist` 会强制执行权限声明检查，页面 binding 没有覆盖菜单路径、binding path 写了 `/_p/<plugin_id>` 或 `/api/v1` 时都会失败。
