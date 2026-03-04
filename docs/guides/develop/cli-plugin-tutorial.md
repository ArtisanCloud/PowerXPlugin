# 使用 `px-plugin` 初始化、启动并本地安装插件

本文是 `px-plugin` 的主流程文档：从 CLI 构建、`init` 项目、数据库初始化、本地启动，到安装到 PowerX。

## 0) 前置条件

- 命令默认在仓库根目录执行：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin`
- 已安装 Go 1.24+、Node.js 18+、npm 9+、PostgreSQL（或 SQLite）

## 1) 构建并安装 `px-plugin`

推荐（全局可直接调用）：

```bash
make install-px-plugin PX_PLUGIN_CLI_VERSION=v0.0.3.3-alpha
hash -r
px-plugin --version
```

仅本地构建（不覆盖全局）：

```bash
make build-px-plugin
./bin/px-plugin --version
```

## 2) 初始化插件项目

命名规则（`plugin-id`）：

```text
^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$
```

示例（推荐交互模式）：

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins
px-plugin init com.powerx.plugin.hello-world
```

交互向导会确认：
- backend / admin
- module root（默认 `github.com/<org>/<plugin-id>`）
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
