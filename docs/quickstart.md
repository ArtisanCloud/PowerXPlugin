# PowerXPlugin Quickstart

本手册帮助你在本地完成 PowerXPlugin 仓库的基础验证。仓库当前仅支持 **Go + Nuxt** 技术栈，如需了解多语言路线图请参阅 `docs/backlog/multi-language.md`。

## 0. 环境要求

- Go 1.21+，已启用 `GOWORK=on`
- Node.js 18+ / npm 9+
- （可选）GNU Make，用于脚本化命令

## 1. 安装依赖

```bash
# 同步 go.work 中声明的模块
go work sync

go mod tidy -e ./framework/... ./tools/cli/...

# 安装前端 workspace 依赖
cd sdk/workspace
npm install
```

确认以下文件同步：

- `go.work` 包含 `use ./framework`、`use ./tools/cli`
- `sdk/workspace/package.json` 的 `workspaces` 指向 `frontend/nuxt/*`

## 2. 启动 Skeleton 示例

```bash
# 启动后端（默认 Standalone 模式）
go run ./skeleton/backend/cmd/plugin

# 另开终端启动前端管理端
cd skeleton/web-admin
npm run dev
```

验证：

- `GET http://localhost:8077/api/v1/ping` 返回 `{ "status": "ok" }`
- 浏览器访问 `http://localhost:3000/_p/com.powerx.sample/admin/`，可看到 Starter 页面

## 3. 构建并运行 CLI

```bash
cd tools/cli
# 构建 px-plugin，可替换为 go install
go build -o ../../bin/px-plugin ./cmd/px-plugin

# 在空目录渲染插件骨架
cd ../../examples
../bin/px-plugin init com.powerx.demo
```

检查渲染结果：

- `plugin.yaml` 包含 `backend`、`frontend`、`version` 字段
- `backend/` 与 `web-admin/` 结构与 Skeleton 保持一致
- README 中标注 `package/dist/publish` 为 `experimental`

## 4. 契约校验（持续完善中）

当前仓库在 `docs/contracts/` 中维护 Manifest、RBAC 与 OpenAPI 契约。CLI 会将这些文件嵌入生成产物；更新后请运行 CI 或自定义脚本完成校验。

```bash
# TODO: 替换为正式的契约校验命令
npm run lint:contracts --if-present
```

## 5. 发布流程与后续动作

- 阅读 `docs/release.md` 了解 Release Workflow、产物清单与手动发布步骤
- 查看 `.github/workflows/release.yml` 了解自动化流程（目前 `package/dist/publish` 仍为实验性输出）
- 关注 `CHANGELOG.md` 与 `examples/starter/` 的更新节奏，确保 CLI 模板与示例保持同步
- 多语言扩展与其他框架支持请持续跟进 `docs/backlog/multi-language.md`

> 如遇构建或依赖问题，可先检查 Go/Node 版本是否符合要求，再对照 `docs/init-project.md` 中的环境说明进行排查。
