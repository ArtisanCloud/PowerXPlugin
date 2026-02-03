# 011 - Python FastAPI + React NextJS Scaffold Plan

## 背景

当前仓库的脚手架唯一真源是 `skeleton/`，通过 `npm run sync:templates` 同步到
`scaffold/templates/**` 与 `tools/cli/internal/templates/**`。现状仅覆盖
Go Gin + Nuxt，CLI `px-plugin init` 也只支持该组合。

## 目标

- 在不破坏现有 Go Gin + Nuxt 的前提下，新增 Python FastAPI + React NextJS 技术栈。
- `px-plugin init` 生成的项目结构仍然只包含 `backend/` 与 `web-admin/`，不出现
  `go-gin/` 或 `next/` 等中间目录。
- 模板同步与 CLI 选项一致，避免 Skeleton / scaffold / CLI 模板漂移。

## 非目标

- 不在本阶段引入新持久层或更改插件运行时模型。
- 不改变现有 Go Gin + Nuxt 的产物结构和启动方式。

## 现状与问题

- Skeleton 只有单一目录：`skeleton/backend/go-gin` 与 `skeleton/web-admin/nuxt`。
- 同步配置固定为 `backend/go-gin` 与 `web-admin/nuxt` 模板。
- CLI 仅允许 `go-gin` 与 `nuxt`，其他类型在常量中处于占位状态。

## 方案概述

将 `skeleton/` 扩展为多栈真源目录，通过同步脚本分别映射到不同模板目录：

```
skeleton/
  backend/
    go-gin/
    python-fastapi/
  web-admin/
    nuxt/
    next/
  Makefile
  plugin.yaml
  make-files/
  ...
```

CLI 渲染逻辑保持不变：在模板目录中保留 `backend/<type>` 与
`web-admin/<type>`，输出时自动映射到 `backend/` 与 `web-admin/`。

## 生成路径约束（必须满足）

- 输出目录仍是 `backend/` 与 `web-admin/`。
- `px-plugin init` 生成的项目中不可出现 `go-gin/`、`python-fastapi/`、
  `nuxt/`、`next/` 等中间层。

## 变更明细（按阶段）

### Phase 1 - Skeleton 结构调整

1. 将现有 `skeleton/backend/go-gin` 移动为 `skeleton/backend/go-gin`。
2. 将现有 `skeleton/web-admin/nuxt` 移动为 `skeleton/web-admin/nuxt`。
3. 新增 `skeleton/backend/go-gin/python-fastapi` 与 `skeleton/web-admin/nuxt/next`：
   - 先放置最小可运行骨架（FastAPI + NextJS）。
   - 目录结构与现有产物一致（例如 `backend/bin`、`web-admin/public`）。

### Phase 2 - 模板同步配置

更新 `scripts/template-sync-config.yaml`：

- 新增 backend mapping：
  - `skeleton/backend/go-gin/python-fastapi` → `scaffold/templates/backend/python-fastapi`
  - `skeleton/backend/go-gin/python-fastapi` → `tools/cli/internal/templates/data/backend/python-fastapi`
- 新增 web-admin mapping：
  - `skeleton/web-admin/nuxt/next` → `scaffold/templates/web-admin/next`
  - `skeleton/web-admin/nuxt/next` → `tools/cli/internal/templates/data/web-admin/next`
- 保留 go-gin/nuxt 的既有 mapping，但源路径改为 `skeleton/backend/go-gin` 与
  `skeleton/web-admin/nuxt`。

### Phase 3 - CLI 与模板类型

- `internal/templates/constants.go`：
  - 开启 `python-fastapi` 与 `next` 为 Supported 列表。
- `cmd/root.go` 与 `cmd/init.go`：
  - 无需修改输出结构，但帮助文案要新增支持项与示例。
- `internal/templates/renderer.go`：
  - 保持现有 `normalizeTargetPath` 逻辑，确保输出到 `backend/` 与 `web-admin/`。

### Phase 4 - 模板注册与文档

- `packages/template-registry/index.yaml`：
  - 新增 `fullstack-python-next` 模板项。
  - 显式写明 templatePaths 与依赖版本。
- 文档更新（README/plan/quickstart）：
  - 同步新模板与 `px-plugin init` 示例。

### Phase 5 - plugin.yaml 适配

FastAPI/NextJS 的运行入口可能与 Go Gin + Nuxt 不同，需在模板层处理：

方案 A：拆分模板
- 为不同栈提供不同的 `plugin.yaml.tmpl`（例如 `plugin-python-next.yaml.tmpl`）。
- 通过模板同步配置与 CLI 选项选择合适的模板文件。

方案 B：单模板条件渲染
- 在 `plugin.yaml.tmpl` 内部用条件渲染控制 `runtime` 与 `frontend` 启动命令。
- 需扩展模板渲染上下文以支持条件判断。

推荐：方案 A（更清晰，减少条件分支复杂度）。

## 影响范围（文件级）

- `skeleton/backend/go-gin/**`、`skeleton/web-admin/nuxt/**`（结构拆分）
- `scripts/template-sync-config.yaml`（新增 mapping）
- `tools/cli/internal/templates/**`（同步产物）
- `scaffold/templates/**`（同步产物）
- `tools/cli/internal/templates/constants.go`（支持列表）
- `tools/cli/cmd/root.go`（帮助文案）
- `packages/template-registry/index.yaml`（模板注册）
- `docs/**`（开发与使用说明）

## 验证清单

- `npm run sync:templates -- --check` 无漂移。
- `px-plugin init`：
  - `--backend go-gin --admin nuxt` 生成现有结构。
  - `--backend python-fastapi --admin next` 生成新结构。
- 生成目录中仅存在 `backend/` 与 `web-admin/`（不含中间框架目录）。

## 风险与缓解

- **风险**：修改 skeleton 结构会影响现有同步脚本。
  - 缓解：分步迁移，先重定向 mapping，再迁移目录。
- **风险**：plugin.yaml 启动逻辑不兼容 FastAPI/NextJS。
  - 缓解：采用多模板或条件渲染方案，明确不同运行入口。

## 迁移建议

1. 先调整 skeleton 目录结构并更新同步配置。
2. 验证 go-gin/nuxt 产物无变化。
3. 再引入 python/next 的最小模板，确保 `px-plugin init` 可生成。
4. 更新文档与模板注册。
