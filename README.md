# PowerXPlugin 仓库

本仓库用于沉淀 PowerX 插件的基础框架、可运行骨架以及 `px-plugin` CLI 模板。当前仅支持 **Go + Nuxt** 技术栈，其他语言的规划请关注 `docs/backlog/multi-language.md`。项目结构与交付流程请参照以下文档：

- 功能规格：`specs/001-powerxplugin-foundation/spec.md`
- 实现计划：`specs/001-powerxplugin-foundation/plan.md`
- 快速上手：`docs/quickstart.md`
- 技术设计：`docs/init-project.md`
- PowerX 通用能力消费方案：[docs/plan/009-consume-powerx-capability.md](./docs/plan/009-consume-powerx-capability.md)（含 Skeleton/宿主示例与观测策略）



## 深入指南

- **架构设计**：[docs/plan/001-init-project.md](./docs/plan/001-init-project.md)
- **Standalone 运行指南**：[docs/guides/develop/standalone-mode.md](./docs/guides/develop/standalone-mode.md)
- **迁移实践**：[docs/guide/migration/base-to-skeleton.md](./docs/guide/migration/base-to-skeleton.md)
- **框架发布指南**：[docs/guides/develop/framework-release.md](./docs/guides/develop/framework-release.md)
- **Mini-App Customer 鉴权**：[docs/guides/develop/auth/customer.md](./docs/guides/develop/auth/customer.md)（含 Skeleton/Delegated 与观测指标）
- **EventBridge / TaskBus 测试指南**：[docs/guides/async_runtime/event_fabric/integration_playbook.md](./docs/guides/async_runtime/event_fabric/integration_playbook.md)

## 快速开始

1. 安装 Go 1.24+、Node.js 20+（npm 9+）、Buf CLI 与 OpenAPI Generator，确保 `PATH` 中可直接调用。
2. 执行 `go work sync`，并在 `framework/frontend/nuxt` 下的各 Nuxt layer 目录运行 `npm install`。
3. 初始化能力工具链：如需 CLI 校验/导出，请运行 `npm --prefix scripts/capabilities install`（若尚未安装依赖），并使用 `make capabilities-lint`、`make capabilities-export` 驱动 `scripts/capabilities` 工具。
4. 参照 `specs/001-powerxplugin-foundation/quickstart.md` 启动 skeleton 后端与管理端。
5. 体验 Go CLI 热加载：请按照 `docs/guides/quickstart.md#dev-api-热更新与-doctor-诊断` 构建 `px-plugin`、运行 `px-plugin dev --watch` / `dev --logs`，并通过 `px-plugin doctor` 生成 `.doctor/report.json` 以验证 Toolchain、mTLS、Dev API、Watcher 状态。
6. 验证 Standalone IAM：按照 `specs/007-standalone-iam-rbac/quickstart.md` 导出 `PLUGIN_IAM_*` 环境变量运行 `go run ./cmd/database/main.go setup`，再使用 `PLAYWRIGHT_LOCAL_IAM=1 npm --prefix skeleton/web-admin/nuxt run test:e2e -- auth-local` 验证本地管理员登录；若要确认 Delegated 模式入口隐藏，可设置 `PLAYWRIGHT_LOCAL_IAM=0`。

### 在本仓库直接输出可安装包（无需先 `px-plugin init`）

当你要联调 PowerX 宿主安装链路时，可直接在本仓库根目录执行：

```bash
# 一条命令校验 plugin.yaml（ID + capabilities + events topics）
make plugin-yaml-check

# 生成 skeleton 安装产物（输出到 skeleton/dist/<version>）
make skeleton-dist

# 直接调用 PowerX 本地安装接口（会先执行 dist）
make skeleton-install API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=<ADMIN_BEARER_TOKEN>

# 一键重装并切换版本（disable -> force install -> switch_version enable）
make skeleton-reinstall VERSION=0.7.1 API_BASE=http://127.0.0.1:8077/api/v1 TOKEN=<ADMIN_BEARER_TOKEN>
```

等价命令分别是 `make -C skeleton dist` 和 `make -C skeleton local-install ...`；`make dist` 在仓库根目录也已透传到 `skeleton`。

如需输出 Linux 安装包（例如在 macOS 上打包部署到 Linux）：

```bash
make dist PLATFORM=linux TARGET_ARCH=amd64 DIST_DIR=dist/0.1.1-linux
```

## Manifest 位置说明

- 仓库真实的开发态 manifest 仅存放在 `skeleton/plugin.yaml`。
- 当前 skeleton 默认使用「索引清单」：`skeleton/plugin.yaml` 仅保留基础元信息，能力/暴露/事件/RBAC 映射统一放在 `skeleton/plugin.d/*.yaml`。
- 支持 `catalogs` 引用模式：`skeleton/plugin.yaml` 通过 `catalogs.*` 指向同级 `skeleton/plugin.d/*.yaml`（capabilities / exposure / agent_tools / events / rbac）。
- 事件声明规范源为 `skeleton/plugin.d/events.yaml` 的 `events.topics[]`；过渡期执行层文件为 `skeleton/config/event_fabric.yaml`（供底座扫描 topic/ACL）。
- 运行 `npm test`、`make validate`、`px-plugin capabilities ...` 等命令时，请在 `skeleton/` 目录中执行（或显式传入 `--manifest skeleton/plugin.yaml` / `CAP_MANIFEST=./skeleton/plugin.yaml`），以免引用到不存在的文件。
- 当你使用 `px-plugin init` 生成独立插件仓库时，`plugin.yaml` 位于其根目录，命令可继续按常规相对路径执行。

## Publish Hub 链路（CLI → 审核 → 安装）

1. **CLI 构建/发布**：阅读 `specs/004-publish-hub-spec/spec.md` 与 `specs/004-publish-hub-spec/quickstart.md`，依照 `px-plugin dev/publish/dist` 命令准备 manifest、签名与 `.pxp` artefact；Node 18 + TypeScript 5 依赖位于 `tools/cli/package.json`。
2. **Marketplace 审核**：`docs/guides/publish/marketplace-review.md` 描述在线/离线审核 checklist、SLA 监控、告警与 reviewer 控制台；配套任务见 `specs/004-publish-hub-spec/tasks.md` Phase 5 & 6。
3. **租户安装/回滚**：按照 `specs/004-publish-hub-spec/tasks.md` Phase 7 以及 quickstart 中的 “Install / Rollback” 步骤，通过 PowerX Admin `install/url` 或 `install/local` 完成安装、灰度与 5 分钟内回滚验证。

## 模板同步流程

- 所有示例代码的唯一真源是 `skeleton/` 目录，请在此完成修改。
- 修改完成后在仓库根目录运行 `npm run sync:templates`，脚本会依据 `scripts/template-sync-config.yaml` 将最新内容同步到 `scaffold/templates/**` 与 `tools/cli/internal/templates/**`。
- 提交前可执行 `npm run sync:templates -- --check`；CI 也会在 PR 中自动执行该命令，确保 scaffold/CLI 模板与 skeleton 不再漂移。

更多背景、约束与阶段性目标，请阅读 `docs/init-project.md` 以及规范中列出的契约文件。
