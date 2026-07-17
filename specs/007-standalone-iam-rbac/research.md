# Research: Standalone 模式 IAM & RBAC

## 决策 1：本地 IAM 运行模式
- **Decision**: Standalone 默认启用 local provider（`POWERX_PROVIDER_MODE=local`），`POWERX_PROXY` 只控制链路；delegated provider 下正式 IAM 菜单保持可见，页面展示 provider/read-only 状态并按 RBAC 控制操作。
- **Rationale**: 需要离线/独立演示能力，同时保持与宿主兼容；文档将强调切换变量与隐藏逻辑。
- **Alternatives considered**: 全量依赖宿主 IAM（Delegated Only）会阻断本地调试；混合模式（同时暴露本地+宿主）增加安全风险。

## 决策 2：RBAC 三元模型
- **Decision**: 完全复用 PowerX `plugin/resource/action` 三元模型，路由自动推导与 Manifest scope 保持一致。
- **Rationale**: 降低迁移到宿主或与 Marketplace 对接的风险；现有工具链（OpenAPI → scope）可沿用。
- **Alternatives considered**: 单独设计本地 `resource.action` 前缀或 JSON Policy，但会破坏宿主一致性并增加维护成本。

## 决策 3：审计与指标分权
- **Decision**: 审计日志全量保留；系统管理员可查看所有租户，租户管理员仅能查看自身租户条目；指标暴露 `plugin_iam_member_total`、`plugin_rbac_denied_total`、`plugin_iam_seed_duration_seconds` 等。
- **Rationale**: 满足租户隔离与平台治理；指标可支撑运行维护。
- **Alternatives considered**: 将日志按租户单独存储或提供跨租户搜索，但现阶段数据量有限，不需要复杂日志路由。

## 决策 4：数据存储与迁移
- **Decision**: 正式环境要求 PostgreSQL（schema `powerx_plugin_base`），本地允许 SQLite 但输出警告；迁移 `iam_*` 表拆分受模式控制。
- **Rationale**: Postgres 支持 RLS/约束；SQLite 仅用于快速演示并可自动迁移。
- **Alternatives considered**: 引入外部 IAM 服务或自建 NoSQL，将显著增加依赖与维护成本。

## 决策 5：CLI 与文档
- **Decision**: CLI 新增 `iam export`/`iam seed`；Quickstart 覆盖初始化、Playwright、Delegated 切换。
- **Rationale**: 方便备份/迁移本地租户与权限，且 QA/运维可按照 Quickstart 快速验证。
- **Alternatives considered**: 仅提供手动 SQL 脚本或 README 说明，难以保证一致性。
