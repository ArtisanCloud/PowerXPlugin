---
name: backend-fastapi-config
description: FastAPI 后端配置加载规范（与 Gin 对齐）
---

# FastAPI 配置加载规范

- 使用 `backend/etc/config.yaml` 作为主入口（支持 `CONFIG_PATH` 覆盖）。
- 候选路径与 Gin 对齐：`./backend/etc/config.yaml`、`./etc/config.yaml` 等。
- `server.api_prefix` 为 API 前缀来源。
- `database.dsn` 与 `database.schema` 为 DB 连接来源（schema 通过 `search_path` 注入）。
