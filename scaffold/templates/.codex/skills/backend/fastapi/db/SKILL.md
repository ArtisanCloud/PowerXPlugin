---
name: backend-fastapi-db
description: FastAPI 数据库与同步/异步约定（SQLAlchemy/Alembic 同步链路）
---

# FastAPI 数据库与同步/异步约定

- 本项目采用 **同步** 数据库与迁移链路（SQLAlchemy sync + Alembic sync）。
- FastAPI 路由可用 `async def`，但 DB 操作保持同步调用（不切 async ORM）。
- Celery/脚本/CLI 与 Web 端统一使用同步 DB 访问，避免跨模式混用。
- PostgreSQL 驱动：`psycopg2`/`psycopg2-binary`。
- 禁止引入 asyncpg/SQLAlchemy async，除非全链路评估并统一切换。
