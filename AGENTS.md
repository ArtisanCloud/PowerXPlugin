# PowerXPlugin Development Guidelines

Auto-generated from all feature plans. Last updated: 2025-10-29

## Active Technologies
- Go 1.24+, TypeScript 5.x (Nuxt 4.2) + Gin 框架、Nuxt 4.2、`@artisan-cloud/plugin-framework-admin` Layer、`@artisan-cloud/plugin-framework-client` (001-powerxplugin-foundation)
- 暂不引入持久层（Skeleton/模板以内存或 mock 为主） (001-powerxplugin-foundation)
- Go 1.24+, Node.js 18+, npm 9+, Bash (POSIX) + Go toolchain (`go test`, `go tool cover`), Playwright 1.48+, Nuxt CLI (`nuxi`), jq/standard UNIX utilities (002-testing-strategy)
- N/A (documentation and scripts only) (002-testing-strategy)
- Go 1.24 (PowerX Core/Dev API), TypeScript 5 + Node.js 18 (px-plugin CLI & tooling), Nuxt 4.2 (Admin) + Gin、`@artisan-cloud/plugin-framework-*`、px-plugin CLI runtime、S3/MinIO 对象存储、Redis/Kafka 链路、Playwright 1.48+ (004-publish-hub-spec)
- Artefacts + integrity 文件放置于对象存储；Marketplace 复用既有 DB 持久化版本/审核记录，本阶段不新增数据库 (004-publish-hub-spec)
- Go 1.24 (backend), TypeScript 5 / Nuxt 4.2 (web admin) + Gin, Gorm, `$fetch`/Nitro、Pinia、`@nuxt/ui`, PowerX framework middleware、`@artisan-cloud/plugin-framework-*` (005-plugin-auth)
- 插件数据库（SQLite/PostgreSQL 由配置决定）中的业务表 + 新增 IAM 表（Local 模式）；Delegated 模式仅读写宿主 API (005-plugin-auth)
- Go 1.24（后端），TypeScript 5 + Nuxt 4.2（前端），Node.js 20 + Gin、Gorm、Pinia、`@nuxt/ui`、PowerX 插件框架、px-plugin CLI (007-standalone-iam-rbac)
- PostgreSQL（生产）/SQLite（本地），schema `powerx_plugin_base` 启用 IAM 表 (007-standalone-iam-rbac)
- Go 1.24 + Gin/Gorm（插件侧既有）；framework runtime 组件（用于 TaskBus 适配，细节见 research） (008-framework-task-bus)
- N/A（本 feature 目标是事件机制与迁移路径；consumer 落地结果时复用既有 DB） (008-framework-task-bus)

## Project Structure

```text
src/
tests/
```

## Commands

npm test && npm run lint

## Code Style

Go 1.24+, TypeScript 5.x (Nuxt 4.2): Follow standard conventions

## Recent Changes
- 008-framework-task-bus: Added Go 1.24 + Gin/Gorm（插件侧既有）；framework runtime 组件（用于 TaskBus 适配，细节见 research）
- 010-auth-customer: Added [if applicable, e.g., PostgreSQL, CoreData, files or N/A]
- 007-standalone-iam-rbac: Added Go 1.24（后端），TypeScript 5 + Nuxt 4.2（前端），Node.js 20 + Gin、Gorm、Pinia、`@nuxt/ui`、PowerX 插件框架、px-plugin CLI

## Manifest 迁移公告（2025-12-08）
- 开发态唯一清单移动到 `skeleton/plugin.yaml`，仓库根目录的 `plugin.yaml` 仅保留 symlink，所有脚本/文档示例已更新。
- 本地执行 `npm test`、`make validate`、`px-plugin capabilities *`、`make dist` 等命令时，请在 `skeleton/` 目录下运行或显式传入 `--manifest ./skeleton/plugin.yaml`。
- `scripts/capabilities/run-from-package.mjs` 与 `validate-capabilities.mjs` 默认回退到 `./skeleton/plugin.yaml`，CI 环节无需额外设定；若在独立插件仓库操作则保持原有路径。
- QA/发布同学在具备 Dev API 凭证时，请使用迁移文档中的命令验证 `px-plugin capabilities submit/quota`，以覆盖新的 manifest 路径。

<!-- MANUAL ADDITIONS START -->
Always respond in Chinese-simplified
<!-- MANUAL ADDITIONS END -->
