# Quickstart

> 目标：在不修改 Nuxt 的前提下，启动 FastAPI 并完成最小联调验证。

## 前置条件
- 已存在 `skeleton/backend/python-fastapi` 目录
- 具备 Python 运行环境

## 启动步骤

```bash
cd skeleton/backend/python-fastapi
bash scripts/dev.sh
```

## 验证

- 访问 `/healthz` 返回 200
- 通过 Nuxt 管理端完成登录与基础列表访问
- 通过宿主反代路径 `/_p/{plugin_id}{api_prefix}` 访问管理端接口可返回一致响应

## 备注

- API 前缀以 `etc/config.yaml` 的 `server.api_prefix` 为准（默认 `/api/v1`）。
- 宿主模式通过 `/_p/<plugin-id>/api/*` 进行反代。

## 迁移

```bash
cd skeleton/backend/python-fastapi
# 配置数据库连接后
alembic upgrade head
```

## 验证记录

- [x] 本地启动 /healthz 通过
- [x] Nuxt 管理端基础流程联调通过
- [x] 迁移脚本可执行
- [x] 管理端 Handler 行为与 Gin 对齐（Auth/IAM/Templates/Capabilities/Runtime Sessions/Manifest/RBAC）
