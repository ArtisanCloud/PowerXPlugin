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
- Python 3.11 + FastAPI, SQLAlchemy 2.0, Alembic (011-fastapi-gin-align)
- PostgreSQL（schema: `powerx_plugin_base`） (011-fastapi-gin-align)
- Go 1.24, TypeScript 5.x (Nuxt 4.2) + Gin/Gorm (backend), PowerXPlugin Framework, Nuxt UI (015-framework-websocket)
- PostgreSQL/SQLite (schema: powerx_plugin_base) (015-framework-websocket)
- TypeScript 5.x, React 18, Next.js 14.2.5, Go 1.24（联调基线） + Next App Router, Playwright（E2E）, 既有 Go-Gin 管理端 API 契约 (012-next-nuxt-align)
- 前端本地会话存储（token/expires）+ 后端既有数据库（由 Gin 管理） (012-next-nuxt-align)
- Go 1.24（backend runtime），TypeScript 5.x（文档/工具链侧验证） + framework runtime（taskbus/wsbus/common middleware）、Gin skeleton、`log/slog`、`logrus` (016-runtime-log-unification)
- N/A（本特性不新增持久化模型） (016-runtime-log-unification)
- Go 1.24（backend runtime），TypeScript 5.x（文档与联调脚本） + Gin skeleton、framework eventbridge/taskbus/wsbus 抽象、runtime ops 管理端点、logrus/slog 结构化日志 (017-async-runtime-scheduler-switch)
- 复用现有插件数据库（PostgreSQL/SQLite），本特性不新增业务持久化模型 (017-async-runtime-scheduler-switch)
- Go 1.24 + framework middleware/context/rbac；skeleton IAM local store；delegated auth proxy (018-framework-iam-unification)
- PostgreSQL/SQLite（local 模式复用现有 IAM 表）；delegated 模式不新增插件侧组织写入 (018-framework-iam-unification)
- Go 1.24, TypeScript 5.x (Nuxt 4.2) + framework IAM contracts/context/errors, skeleton IAM/auth service, RBAC, observability/audi (019-iam-federated-channel-login)
- PostgreSQL/SQLite（复用 IAM 表并新增 external_identity/binding/challenge/risk_event） (019-iam-federated-channel-login)
- Go 1.24（backend runtime/framework）, TypeScript 4.x（web-admin 验证） + framework runtime/common logging, slog/logrus adapter, skeleton logger bridge, observability hooks (020-framework-logger)
- N/A（不新增业务持久化表；仅复用现有配置来源与日志后端） (020-framework-logger)
- Go 1.24 + Gin middleware/context, existing skeleton customer auth, PowerX STS/delegated client patterns, framework runtime common logging/errors (023-framework-customer-auth)
- Framework 不新增生产 customer 持久化；skeleton local/dev 可复用现有 customer 表；生产 membership/customer 数据由 PowerX Core 或平台身份源权威维护 (023-framework-customer-auth)

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
- 023-framework-customer-auth: Added Go 1.24 + Gin middleware/context, existing skeleton customer auth, PowerX STS/delegated client patterns, framework runtime common logging/errors
- 020-framework-logger: Added Go 1.24（backend runtime/framework）, TypeScript 4.x（web-admin 验证） + framework runtime/common logging, slog/logrus adapter, skeleton logger bridge, observability hooks
- 019-iam-federated-channel-login: Added Go 1.24, TypeScript 5.x (Nuxt 4.2) + framework IAM contracts/context/errors, skeleton IAM/auth service, RBAC, observability/audi

## Manifest 迁移公告（2025-12-08）
- 开发态唯一清单移动到 `skeleton/plugin.yaml`，仓库根目录的 `plugin.yaml` 仅保留 symlink，所有脚本/文档示例已更新。
- 本地执行 `npm test`、`make validate`、`px-plugin capabilities *`、`make dist` 等命令时，请在 `skeleton/` 目录下运行或显式传入 `--manifest ./skeleton/plugin.yaml`。
- `scripts/capabilities/run-from-package.mjs` 与 `validate-capabilities.mjs` 默认回退到 `./skeleton/plugin.yaml`，CI 环节无需额外设定；若在独立插件仓库操作则保持原有路径。
- QA/发布同学在具备 Dev API 凭证时，请使用迁移文档中的命令验证 `px-plugin capabilities submit/quota`，以覆盖新的 manifest 路径。

<!-- MANUAL ADDITIONS START -->
Always respond in Chinese-simplified

当出现新的策略、协议、接口形态或数据格式时，默认只实现和维护新方式；不要主动兼容废弃方式、旧协议或旧格式。只有在用户明确要求兼容时，才添加向后兼容逻辑，并在实现和说明中标注兼容边界。

默认不添加通用 fallback 或静默降级，除非用户明确要求。前端实时更新如果设计为 WebSocket/SSE，不得偷偷增加轮询兜底；结构化字段不得用自由文本解析兜底；渠道、传输、鉴权、数据契约、构建运行时等边界同样不得静默降级。更合理的处理策略是明确失败、显示错误状态、记录日志、提供恢复动作，而不是隐藏式绕过问题。

所有人类可读文本都必须走 i18n/locale，不得硬编码在业务代码、组件或测试断言里。覆盖范围包括前端按钮、提示、错误信息、空状态、确认弹窗、后端对用户可见回复、邮件模板、agent 对用户可见提示、业务角色显示名、角色别名/屏蔽词表，以及测试里依赖文案的断言。例外：协议常量、枚举值、JSON 字段名、路由、日志 key、状态码、能力名、数据库字段、i18n key 等机器语义可以保留在代码里。

UI 默认不得直接把对象 UUID 作为用户可见显示文本，除非用户明确要求展示 UUID。列表、详情、表单、selector option、确认弹窗、提示文案等用户界面应优先显示对象名称或业务可读标识，例如显示租户名称而不是租户 UUID；UUID 仅作为内部 value、路由参数、请求字段、日志字段、调试诊断或明确的技术详情展示。

业务对象表必须具备稳定的 `uuid` 字段。跨表、跨服务、API、事件、审计等所有对象引用统一使用对象 `uuid`，不得新增 numeric id 作为对外或跨边界引用。

中间表可以没有自己的 `uuid`，但关联字段必须使用两端业务对象的 `uuid`。如果中间表演进为可审计、可引用或具备独立状态的业务对象，也必须补充自己的稳定 `uuid`。

新增迁移、GORM 模型、Proto、OpenAPI、前端类型与测试必须按 UUID 规则设计和校验。修正旧 numeric id 引用时不做兼容兜底；缺少必要 UUID 必须明确失败，并提供迁移说明。
<!-- MANUAL ADDITIONS END -->
