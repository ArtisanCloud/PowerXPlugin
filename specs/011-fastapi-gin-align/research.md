# Research

## Decision 1: 接口权威来源
- Decision: 以 Go Gin 实现作为契约权威来源。
- Rationale: 现有生产与宿主规范以 Gin 为基线，能最大化兼容性。
- Alternatives considered: 以 Nuxt 调用为权威；以文档为权威。

## Decision 2: API 前缀来源
- Decision: 以 `etc/config.yaml` 的 `server.api_prefix` 为准，默认 `/api/v1`。
- Rationale: 与 Gin 保持一致并兼容宿主反代路径。
- Alternatives considered: 固定 `/api/v1` 不可配置。

## Decision 3: ORM 与迁移
- Decision: 使用 SQLAlchemy 2.0 + Alembic。
- Rationale: 生态成熟，迁移能力完善，便于对齐表结构。
- Alternatives considered: Tortoise ORM；无 ORM 直写 SQL。

## Decision 4: 数据库与 Schema
- Decision: 以 PostgreSQL 为主，schema 采用 `powerx_plugin_base`。
- Rationale: 与插件侧既有约定一致，便于宿主与多租户一致性验证。
- Alternatives considered: SQLite 为主；自定义 schema。

## Decision 5: 测试策略
- Decision: 以 pytest 为单测框架，补充轻量集成测试。
- Rationale: Python 标准生态，易于与 CI 集成。
- Alternatives considered: 仅手工联调；unittest 原生框架。

## Decision 6: 性能与可用性目标
- Decision: 关键联调 API 的 P95 < 1s，错误率 < 1%。
- Rationale: 联调阶段需保证稳定性但避免过严指标阻碍迭代。
- Alternatives considered: 不设目标；更严格的性能目标。
