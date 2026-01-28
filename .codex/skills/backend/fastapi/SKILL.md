---
name: backend-fastapi
description: PowerXPlugin FastAPI 后端规范（配置加载、数据库驱动、SQLAlchemy/Alembic 同步链路）
---

# FastAPI 后端规范

## 配置加载（对齐 Gin）

- 使用 `backend/etc/config.yaml` 作为主入口（支持 `CONFIG_PATH` 覆盖）。
- 候选路径与 Gin 对齐：`./backend/etc/config.yaml`、`./skeleton/backend/etc/config.yaml`、`./etc/config.yaml` 等。
- `server.api_prefix` 为 API 前缀来源。
- `database.dsn` 与 `database.schema` 为 DB 连接来源（schema 通过 `search_path` 注入）。

## 数据库驱动与 ORM

- PostgreSQL 驱动：`psycopg2`/`psycopg2-binary`。
- SQLAlchemy 2.x 同步模式；不启用 async ORM。
- Alembic 同步迁移（`alembic upgrade head`）。
- 与 Celery/脚本一致：统一使用同步 DB 访问。

## 约束

- 不引入 asyncpg/SQLAlchemy async。
- 若需异步，必须评估 Celery/脚本链路并同步调整（默认不做）。
