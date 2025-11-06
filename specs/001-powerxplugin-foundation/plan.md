# Implementation Plan: PowerXPlugin 仓库基线落地

**Branch**: `001-powerxplugin-foundation` | **Date**: 2025-10-29 | **Spec**: `specs/001-powerxplugin-foundation/spec.md`
**Status**: Phase 7 - Polish & Cross-Cutting  
**Input**: 基线规格来自 `/specs/001-powerxplugin-foundation/spec.md`

## Summary

落实 PowerXPlugin Phase 0~4 的地基建设：提供 `go.work` 管理的多模块 Go 框架、`framework/frontend/nuxt` 下的 Nuxt Layer/npm workspace、可运行的 skeleton 以及与之完全对齐的 CLI 模板与实验性 `package/dist/publish` 工作流；所有实现与契约、目录、示例必须同步 `docs/init-project.md` 的技术设计。

## Technical Context

**Language/Version**: Go 1.24+, TypeScript 5.x (Nuxt 4.2)  
**Primary Dependencies**: Gin 框架、Nuxt 4.2、`@powerx-plugin/framework-admin` Layer、`@powerx-plugin/framework-client`  
**Storage**: 暂不引入持久层（Skeleton/模板以内存或 mock 为主）  
**Testing**: `go test ./...`、`npm run lint && npm run build`  
**Target Platform**: PowerX Core 宿主环境（Linux 服务 + Nuxt SSR）  
**Project Type**: 多模块仓库（Go 后端 + Nuxt 前端 + CLI）  
**Performance Goals**: 本地单实例可稳定运行 `go run` / `npm run dev` / `npm run build`  
**Constraints**: 必须通过 Go lint/test 与 `npm run build`，CLI 离线可运行  
**Scale/Scope**: 支撑单插件团队（约 5-10 名成员）完成 Phase 0~4 交付

## Constitution Check

| 原则 | 状态 | 说明 |
|------|------|------|
| 双重使命仓库 | ✅ | 计划保持 skeleton 与 `framework/`、CLI 模板一致，满足 Principle I |
| 契约优先兼容性 | ✅ | 契约文件落位 `docs/contracts/**` 并纳入 CI，符合 Principle II |
| Go + Nuxt 基线 | ✅ | 使用 `go.work` + npm workspace，沿用 Go(Gin)+Nuxt 栈 |
| 脚手架与 CLI 纪律 | ✅ | CLI 模板与实验性命令均标注状态，输出与 skeleton 对齐 |
| 透明交付与一致性 | ✅ | 计划覆盖 CI、文档同步及 TODO 标记，无额外豁免 |

## Project Structure

### Documentation (this feature)

```text
specs/001-powerxplugin-foundation/
├── plan.md              # 本文件（/speckit.plan 输出）
├── research.md          # Phase 0 研究产物
├── data-model.md        # Phase 1 数据模型
├── quickstart.md        # Phase 1 启动指南
├── contracts/           # Phase 1 契约草稿
└── tasks.md             # /speckit.tasks 生成（本命令不创建）
```

### Source Code (repository root)

```text
PowerXPlugin/
├─ go.work                                 # Phase 0: 管理 framework/ 与 tools/cli/ 多模块
├─ package.json                            # Phase 0: 根级 npm workspaces（可选）
├─ framework/                              # Phase 3: 共享后端框架
│  ├─ go.mod (module github.com/ArtisanCloud/PowerXPlugin/framework)
│  └─ backend/go/
│     ├─ bootstrap/                        # App 初始化（参考 Base/internal/bootstrap）
│     ├─ router/                           # RegisterFrameworkRoutes/RegisterPluginRoutes
│     ├─ middleware/                       # AuthGuard stub、Recovery、Trace
│     ├─ manifest/                         # Manifest 类型与注册
│     ├─ rbac/                             # 权限报告
│     ├─ observability/                    # 指标 & Trace 集成
│     ├─ tenancy/                          # 多租户上下文
│     └─ shared/                           # 通用组件与工具
├─ framework/frontend/nuxt/                # Phase 3: Nuxt Layer + Client npm 包
│  ├─ package.json (workspaces: frontend/nuxt/*)
│  └─ frontend/nuxt/
│     ├─ framework-admin/                  # @powerx-plugin/framework-admin
│     │  ├─ layer/app/{components,middleware,pages,plugins}
│     │  ├─ layer/nuxt.config.ts
│     │  ├─ module.ts
│     │  └─ index.ts (definePowerXAdminConfig)
│     └─ framework-client/                 # @powerx-plugin/framework-client
│        ├─ api.ts / http.ts
│        └─ index.ts
├─ skeleton/                               # Phase 2: 可运行样例
│  ├─ backend/
│  │  ├─ go.mod (require github.com/ArtisanCloud/PowerXPlugin/framework)
│  │  ├─ cmd/plugin/main.go                # 6 步装配流程（参考 Base backend/cmd/plugin）
│  │  └─ internal/
│  │     ├─ routes/                        # `/api/v1/ping`
│  │     ├─ handler/
│  │     ├─ service/
│  │     └─ manifestx/                     # Manifest 定义
│  └─ web-admin/
│     ├─ app/{components,pages,_p/...}
│     ├─ nuxt.config.ts
│     └─ package.json
├─ scaffold/templates/                     # Phase 4: CLI 模板（与 skeleton 一一对应）
│  ├─ backend/go-gin/
│  │  ├─ cmd/plugin/main.go.tmpl
│  │  └─ internal/*.tmpl
│  └─ web-admin/nuxt/
│     ├─ nuxt.config.ts.tmpl
│     └─ app/**/*.tmpl
├─ tools/cli/                              # Phase 4: px-plugin CLI
│  ├─ go.mod (module github.com/powerx-plugin/cli)
│  └─ cmd/
│     ├─ init.go
│     ├─ package.go        # experimental
│     ├─ dist.go           # experimental
│     └─ publish.go        # experimental
├─ docs/
│  ├─ contracts/{manifest.json,rbac.json,openapi.yaml}
│  └─ init-project.md
├─ examples/starter/                       # Phase 5: CLI 生成物快照
└─ config/config.yaml.example              # 配置样例
```

**Structure Decision**: 以 Base 插件实装为基准，强化多模块骨架：`framework/` 拆解 Base/internal 逻辑形成框架层，`skeleton/` 复制最小可运行流程，`scaffold/templates/` 与 skeleton 保持 100% 同步，`tools/cli/` 提供 px-plugin 命令；整体结构与 `docs/init-project.md` 及 `/com.powerx.plugin.base` 一致，确保 CLI、框架与示例互相验证。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
