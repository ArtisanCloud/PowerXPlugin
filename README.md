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
- **框架发布指南**：[docs/guide/framework-release.md](./docs/guide/framework-release.md)

## 快速开始

1. 安装 Go 1.24+ 与 Node.js 18+。
2. 执行 `go work sync`，并在 `framework/frontend/nuxt` 下的各 Nuxt layer 目录运行 `npm install`。
3. 参照 `specs/001-powerxplugin-foundation/quickstart.md` 启动 skeleton 后端与管理端。

## Publish Hub 链路（CLI → 审核 → 安装）

1. **CLI 构建/发布**：阅读 `specs/004-publish-hub-spec/spec.md` 与 `specs/004-publish-hub-spec/quickstart.md`，依照 `px-plugin dev/publish/dist` 命令准备 manifest、签名与 `.pxp` artefact；Node 18 + TypeScript 5 依赖位于 `tools/cli/package.json`。
2. **Marketplace 审核**：`docs/guides/publish/marketplace-review.md` 描述在线/离线审核 checklist、SLA 监控、告警与 reviewer 控制台；配套任务见 `specs/004-publish-hub-spec/tasks.md` Phase 5 & 6。
3. **租户安装/回滚**：按照 `specs/004-publish-hub-spec/tasks.md` Phase 7 以及 quickstart 中的 “Install / Rollback” 步骤，通过 PowerX Admin `install/url` 或 `install/local` 完成安装、灰度与 5 分钟内回滚验证。

## 模板同步流程

- 所有示例代码的唯一真源是 `skeleton/` 目录，请在此完成修改。
- 修改完成后在仓库根目录运行 `npm run sync:templates`，脚本会依据 `scripts/template-sync-config.yaml` 将最新内容同步到 `scaffold/templates/**` 与 `tools/cli/internal/templates/**`。
- 提交前可执行 `npm run sync:templates -- --check`；CI 也会在 PR 中自动执行该命令，确保 scaffold/CLI 模板与 skeleton 不再漂移。

更多背景、约束与阶段性目标，请阅读 `docs/init-project.md` 以及规范中列出的契约文件。
