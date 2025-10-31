# PowerXPlugin 仓库

本仓库用于沉淀 PowerX 插件的基础框架、可运行骨架以及 `px-plugin` CLI 模板。当前仅支持 **Go + Nuxt** 技术栈，其他语言的规划请关注 `docs/backlog/multi-language.md`。项目结构与交付流程请参照以下文档：

- 功能规格：`specs/001-powerxplugin-foundation/spec.md`
- 实现计划：`specs/001-powerxplugin-foundation/plan.md`
- 快速上手：`docs/quickstart.md`
- 技术设计：`docs/init-project.md`

## 快速开始

1. 安装 Go 1.21+ 与 Node.js 18+。
2. 执行 `go work sync`，并在 `sdk/workspace/` 目录运行 `npm install`。
3. 参照 `specs/001-powerxplugin-foundation/quickstart.md` 启动 skeleton 后端与管理端。

更多背景、约束与阶段性目标，请阅读 `docs/init-project.md` 以及规范中列出的契约文件。
