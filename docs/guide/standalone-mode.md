# PowerXPlugin Standalone 启动教程

本教程演示如何在本地直接启动 PowerXPlugin 仓库自带的 Skeleton 示例（后端 + 管理端），用于验证框架与模板的基础功能。全程默认使用仓库内置的配置，避免依赖尚未实现的发布或安装流程。

## 适用场景

- 验证 `framework/backend/go` 与 Skeleton 示例是否能够在本机运行。
- 熟悉框架提供的 `App` 装配流程与 Standalone 环境变量。
- 为后续编写自定义插件或运行测试脚本打基础。

## 前置条件

- Go 1.21+，且启用 `GOWORK=on`
- Node.js 18+ 与 npm 9+
- （可选）GNU Make 或 `tmux`，便于同时运行前后端

建议首次运行前执行 `go version` 与 `node -v` 确认版本。仓库当前仅支持 **Go + Nuxt** 技术栈，其他语言路线仍在规划中。

## Step 1. 同步依赖

在仓库根目录执行以下命令，确保 Go Workspace 与前端依赖就绪：

```bash
go work sync

go -C framework mod tidy -e
go -C tools/cli mod tidy -e

cd sdk/workspace
npm install
```

执行完成后，请确认：

- `go.work` 中包含 `use ./framework` 与 `use ./tools/cli`
- `sdk/workspace/package.json` 的 `workspaces` 指向 `frontend/nuxt/*`
- 两个 Go 模块（`framework/` 与 `tools/cli/`）的依赖已整理完成

## Step 2. 启动后端（Standalone）

Skeleton 后端入口位于 `skeleton/backend/cmd/plugin/main.go`，默认加载 `framework/backend/go/bootstrap` 的 Standalone 配置。

```bash
go run ./skeleton/backend/cmd/plugin
```

首次启动会监听 `:8078`。若需要自定义端口或环境，可在启动前设置：

```bash
export POWERX_LISTEN=":18078"
export POWERX_ENV="development"
# STANDALONE 默认为 true，可按需覆盖
```

### 验证 API

服务器启动后，另开终端执行：

```bash
curl http://localhost:8078/api/v1/ping
```

预期返回：

```json
{ "status": "ok" }
```

若收到 200 但响应为空，确认 `routes.Register` 已挂载；若命令报错，请检查端口是否被占用。

## Step 3. 启动前端管理端

Skeleton 管理端位于 `skeleton/web-admin`，基于 Nuxt 4.2 和 `@powerx-plugin/framework-admin` Layer。

```bash
cd skeleton/web-admin
npm run dev
```

Skeleton 已在 `nuxt.config.ts` 中固定 `devServer.port = 3031`，并将 Vite HMR 显式配置为 `ws://localhost:24731`。默认页面地址为 `http://localhost:3031/`。若你需要使用其他端口，可通过：

```bash
npm run dev -- --port <custom-port> --hmr-port <custom-hmr-port>
```

确保后端保持运行，否则页面请求会因 API 不可用而失败。

### 验证界面

在浏览器访问：

```
http://localhost:3031/_p/com.powerx.sample/admin/
```

若你自定义了端口，请将 `3031` 替换为对应值。可看到 “PowerX Sample” Starter 页面。若界面缺失 Layout 或 HMR 无法建立，请确认 `node_modules` 完整、端口未被占用，并根据需要调整 `--hmr-port`。

## Step 4. 停止服务与清理

- 后端：在运行终端按 `Ctrl+C`，`bootstrap.App` 会触发优雅关闭。
- 前端：在运行终端按 `Ctrl+C` 停止 Nuxt Dev Server。

如需重置依赖，可删除 `skeleton/web-admin/node_modules` 后重新安装。

---

完成以上步骤，即可在本地以 Standalone 模式运行 PowerXPlugin Skeleton，实现后端 API 与管理端页面的联调验证。建议继续阅读《使用 CLI 生成并运行插件骨架》以了解如何构建独立插件项目。
