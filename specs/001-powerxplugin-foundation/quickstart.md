# Quickstart - PowerXPlugin 仓库基线落地

> ℹ️ 本文档保留生成历史，最新快速上手指南请参阅 `docs/quickstart.md`。

## 必备环境

- Go 1.24+，启用 `GOWORK=on`
- Node.js 18+ / npm 9+
- GNU make（可选，用于脚本化任务）

## 1. 初始化依赖

```bash
# 安装 Go 模块依赖
go work sync
go mod tidy -e ./framework/... ./tools/cli/...

# 安装前端 workspace 依赖
cd framework/frontend/nuxt/framework-admin
npm install
cd ../framework-client
npm install
```

确认以下结构存在并与 `docs/init-project.md` 一致：

- `go.work` → `use ./framework`、`use ./tools/cli`
- `framework/frontend/nuxt/framework-admin`、`framework-client` 完成依赖安装

## 2. 运行 Skeleton

```bash
# 后端（Standalone 模式）
go run ./skeleton/backend/cmd/plugin

# 另开终端启动管理端
cd skeleton/web-admin
npm run dev
```

- 验证 `GET http://localhost:8077/api/v1/ping` 返回 `{ "status": "ok" }`
- 在浏览器访问 `http://localhost:3000/_p/<plugin-id>/admin/`，Starter 页面应可加载

## 3. 渲染脚手架模板

```bash
# 构建 CLI（或 go install）
cd tools/cli
go build -o ../../bin/px-plugin ./cmd/px-plugin

# 在空目录生成插件
../../bin/px-plugin init com.powerx.demo
```

- 检查生成项目中的 `plugin.yaml`、`backend/`、`web-admin/` 是否与 skeleton 对齐
- `README` 内应标记 `package/dist/publish` 为 `experimental`

## 4. 验证契约

```bash
# 运行契约校验脚本（待实现，可通过 JSON Schema 工具替代）
npm run lint:contracts   # 示例命令，占位
```

- 确认 `docs/contracts/manifest.json`、`docs/contracts/rbac.json`、`docs/contracts/openapi.yaml` 无 schema 错误
- 更新 Manifest 或 RBAC 后需触发 CI，再由框架/SDK 消费最新 Schema

## 5. 准备发布流程（实验性）

```bash
px-plugin package   # 构建后端 + 前端
px-plugin dist      # 打包产物
px-plugin publish   # 调用 Marketplace API 上传（需配置 token）
```

> 当前命令仍处实验阶段，CLI 应打印 “experimental” 提示；执行结果需写入 CHANGELOG/TODO。
