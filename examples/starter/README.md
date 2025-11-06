# Powerx Starter Plugin

由 `px-plugin init com.powerx.starter` 生成的 PowerX 插件模板。

## 快速开始

```bash
# 安装后端依赖（使用本地相对路径指向 framework）
cd backend
go mod tidy

# 安装前端依赖（使用 file: 协议指向本地框架包）
cd ../web-admin
npm install
```

```bash
# 启动后端
cd backend
go run ./cmd/plugin

# 启动前端（新终端）
cd web-admin
npm run dev
```

> **注意**：此示例项目用于在 PowerXPlugin 仓库内验证，不依赖已发布的框架包。若要在独立环境中使用，请先发布 `@artisan-cloud/plugin-framework-admin` 与 `@artisan-cloud/plugin-framework-client` 到 npm，然后将 `package.json` 中的依赖版本改为具体版本号。

## 下一步

- 更新 `plugin.yaml` 中的版本与元数据。
- 扩展 `backend/internal/` 与 `web-admin/app/` 以实现业务逻辑。
- 查看 PowerXPlugin 仓库的 `docs/release.md` 获取发布流程概览。
- `package`/`dist`/`publish` 命令仍为实验特性，执行前请阅读 CLI README。
- 多语言扩展暂未开放，关注仓库 `docs/backlog/multi-language.md` 中的 TODO。
