
# 从 com.powerx.plugin.base 迁移到 Skeleton + Scaffold

本文提供一份操作手册，指导如何将现有的 `com.powerx.plugin.base` 插件拆分为 PowerXPlugin 仓库内的 Skeleton（可运行示例）与 Scaffold/CLI 模板。

## 1. 迁移 Checklist

### 后端

- [ ] 复制 `backend/internal/entity/models/**` → `skeleton/backend/go-gin/internal/entity/models/`
- [ ] 复制 `backend/internal/entity/repository/**` → `skeleton/backend/go-gin/internal/entity/repository/`
- [ ] 复制 `backend/internal/services/**` → `skeleton/backend/go-gin/internal/services/`
- [ ] 复制 `backend/internal/transport/http/**`、`transport/grpc/**` → `skeleton/backend/go-gin/internal/transport/`

#### 关键决策：选择迁移路径

**路径 A：真实业务示例**（推荐，用于展示与 PowerX Core 集成）
- ✅ 保留 `github.com/ArtisanCloud/PowerX/...` import（展示如何调用 Core API）
- ✅ 保留真实数据库依赖（GORM + Postgres）
- ✅ 使用演示 Plugin ID（如 `com.powerx.sample`）
- ✅ 在 `go.mod` 添加 replace 指向本地 PowerX 源码

**路径 B：独立可运行示例**（适用于无 Core 环境的开发）
- ❌ 替换 `github.com/ArtisanCloud/...` → `github.com/ArtisanCloud/PowerXPlugin/framework/...`
- ❌ 将数据库仓储改为内存实现（map + mutex）
- ❌ 使用通用 Plugin ID（如 `com.powerx.plugin.demo`）

- [ ] 迁移 `internal/router` 中的路由装配逻辑，与 Skeleton 现有结构对齐
- [ ] 更新 `internal/manifestx/manifest.go` 的 ID、名称、菜单、权限
- [ ] 将数据库仓储替换为内存/mock 实现（**仅路径 B 需要**）

### 前端

- [ ] 复制 `web-admin/app/components/**` → `skeleton/web-admin/nuxt/app/components/`
- [ ] 复制 `web-admin/app/pages/**` → `skeleton/web-admin/nuxt/app/pages/`
- [ ] 调整 API 调用逻辑，使用 `~/app/composables/api/_client.ts` 暴露的统一客户端
- [ ] 同步 i18n 文案至 `skeleton/web-admin/nuxt/i18n/locales/`
- [ ] 更新 `nuxt.config.ts` 中的插件 ID 与标题信息

### 模板同步

- [ ] 修改 Skeleton 后执行 `npm run sync:templates`
- [ ] 确认 `scaffold/templates/**` 与 `tools/cli/internal/templates/**` 生成的新文件结构
- [ ] 运行 `npm run sync:templates -- --check` 确认无漂移

## 2. 迁移步骤（后端）

1. 在新的工作分支创建迁移目录：
   ```bash
   git checkout -b feat/migrate-base
   ```
2. 将 Base 项目的核心业务文件复制到 Skeleton 对应目录，保持 `entity → services → transport` 命名一致。
3. 使用 `rg`/`sed` 批量替换 import：
   ```bash
   rg --files -g'*.go' 'github.com/ArtisanCloud/PowerX' com.powerx.plugin.base/backend |      xargs sed -i '' 's#github.com/ArtisanCloud/PowerX#github.com/ArtisanCloud/PowerXPlugin/framework#g'
   ```
4. 为 Skeleton 提供最小运行实现：如原仓储依赖数据库，则编写内存版本（map + mutex），并在文档中标注生产实现需要自行接入。
5. 调整 `internal/router` 与 `cmd/plugin/main.go`，确保遵循框架路由顺序：
   ```go
   router.RegisterFrameworkRoutes(app)
   router.RegisterPluginRoutes(app, routes.Register)
   ```
6. 更新 `internal/manifestx/manifest.go` 以反映新的菜单、权限项。
7. `cd skeleton/backend/go-gin && go test ./...`，验证改动不会破坏现有测试。

## 3. 迁移步骤（前端）

1. 按模块迁移组件与页面，注意重用 `@artisan-cloud/plugin-framework-admin` 提供的 Layout/中间件。
2. API 访问统一走 `~/app/composables/api` 提供的 `apiGet/apiPost`。
3. 在 `app/app.vue`、`nuxt.config.ts` 等位置更新插件名称、描述、ID。
4. 执行 `npm install` 与 `npm run dev`，确认页面无报错。

## 4. 同步模板与 CLI

1. 修改 Skeleton 后运行：
   ```bash
   npm run sync:templates
   npm run sync:templates -- --check
   ```
2. 模板生成完成后，可手动执行一次 CLI 自举验证：
   ```bash
   go build -o bin/px-plugin ./tools/cli/cmd/px-plugin
   ./bin/px-plugin init com.powerx.demo --force --module github.com/ArtisanCloud/PowerXPlugin/plugins/com-powerx-demo
   ```
3. 对比 `examples/com.powerx.demo` 与 CLI 生成物，确认关键结构一致。

## 5. 验收清单

- [ ] Skeleton 与 Scaffold 目录结构统一（`entity`、`services`、`transport` 等分层对应）。
- [ ] `go test ./...`、`npm run build` 均通过。
- [ ] `npm run sync:templates -- --check` 无漂移。
- [ ] CLI `px-plugin init` 生成项目后，`go run ./cmd/plugin` 与 `npm run dev` 均可启动。
- [ ] README 与文档中引用的目录结构已更新到新命名。

## 6. 参考资源

- [架构设计总览](../plan/001-init-project.md)
- [Standalone 运行指南](../develop/standalone/README.md)
- [模板同步脚本配置](../../scripts/template-sync-config.yaml)

## 7. 当前 Skeleton 的选择

**PowerXPlugin 仓库中的 Skeleton 实际采用路径 A（真实业务示例）**：

- ✅ 保留对 PowerX Core 的 gRPC 调用（`internal/grpc/client/powerx.go`）
- ✅ 使用演示 Plugin ID `com.powerx.sample`
- ✅ 依赖真实数据库（用于完整业务场景）
- ✅ `go.mod` 中的 `replace` 指令指向本地 PowerX 源码

**这种选择的理由**：
1. Skeleton 是"可运行的最佳实践参考"，不是"最小可运行示例"
2. 展示如何与宿主系统（PowerX Core）集成是核心价值
3. 开发者可以通过 Skeleton 学习真实业务的完整实现

**如果需要独立运行的示例**，请使用 CLI 生成的项目：
```bash
px-plugin init com.powerx.newplugin  # 生成路径 B 的独立项目
```
