# PowerXPlugin 仓库

本仓库用于沉淀 PowerX 插件的基础框架、可运行骨架以及 `px-plugin` CLI 模板。当前仅支持 **Go + Nuxt** 技术栈，其他语言的规划请关注 `docs/backlog/multi-language.md`。项目结构与交付流程请参照以下文档：

- 功能规格：`specs/001-powerxplugin-foundation/spec.md`
- 实现计划：`specs/001-powerxplugin-foundation/plan.md`
- 快速上手：`docs/quickstart.md`
- 技术设计：`docs/init-project.md`



## 深入指南

- **架构设计**：[docs/plan/001-init-project.md](./docs/plan/001-init-project.md)
- **Standalone 运行指南**：[docs/guide/standalone-mode.md](./docs/guide/standalone-mode.md)
- **迁移实践**：[docs/guide/migration/base-to-skeleton.md](./docs/guide/migration/base-to-skeleton.md)

## 快速开始

1. 安装 Go 1.24+ 与 Node.js 18+。
2. 执行 `go work sync`，并在 `framework/frontend/nuxt` 下的各 Nuxt layer 目录运行 `npm install`。
3. 参照 `specs/001-powerxplugin-foundation/quickstart.md` 启动 skeleton 后端与管理端。

## 模板同步流程

- 所有示例代码的唯一真源是 `skeleton/` 目录，请在此完成修改。
- 修改完成后在仓库根目录运行 `npm run sync:templates`，脚本会依据 `scripts/template-sync-config.yaml` 将最新内容同步到 `scaffold/templates/**` 与 `tools/cli/internal/templates/**`。
- 提交前可执行 `npm run sync:templates -- --check`；CI 也会在 PR 中自动执行该命令，确保 scaffold/CLI 模板与 skeleton 不再漂移。

更多背景、约束与阶段性目标，请阅读 `docs/init-project.md` 以及规范中列出的契约文件。
