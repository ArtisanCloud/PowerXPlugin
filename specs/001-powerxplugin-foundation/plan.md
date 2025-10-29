# Implementation Plan: PowerXPlugin 仓库基线落地

**Branch**: `001-powerxplugin-foundation` | **Date**: 2025-10-29 | **Spec**: `specs/001-powerxplugin-foundation/spec.md`
**Input**: 基线规格来自 `/specs/001-powerxplugin-foundation/spec.md`

## Summary

落实 PowerXPlugin Phase 0~4 的地基建设：提供 `go.work` 管理的多模块 Go 框架、`sdk/workspace` 下的 Nuxt Layer/npm workspace、可运行的 skeleton 以及与之完全对齐的 CLI 模板与实验性 `package/dist/publish` 工作流；所有实现与契约、目录、示例必须同步 `docs/init-project.md` 的技术设计。

## Technical Context

**Language/Version**: Go 1.21+, TypeScript 5.x (Nuxt 3)  
**Primary Dependencies**: Gin 框架、Nuxt 3、`@powerx-plugin/framework-admin` Layer、`@powerx-plugin/framework-client`  
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
framework/
├── backend/go/
│   ├── bootstrap/
│   ├── router/
│   ├── middleware/
│   └── manifest/
├── sdk/workspace/
│   └── frontend/nuxt/
│       ├── framework-admin/
│       └── framework-client/

skeleton/
├── backend/
│   ├── cmd/plugin/
│   └── internal/{routes,handler,service,repo,manifestx}
└── web-admin/
    ├── app/
    ├── nuxt.config.ts
    └── package.json

scaffold/templates/
├── backend/go-gin/
└── web/nuxt/

tools/cli/
└── px-plugin (Go CLI 源码及嵌入模板)
```

**Structure Decision**: 维持仓库的多模块骨架：`framework/` 输出共享框架层、`skeleton/` 提供可运行样例、`scaffold/templates/` 供 CLI 渲染，`tools/cli/` 则负责 px-plugin 命令；所有目录需与 `docs/init-project.md` 指定结构一一对应。

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
