# FastAPI 对齐 Go Gin 计划（以 Nuxt 联调为第一目标）

## 背景

Go Gin + Nuxt 已是当前仓库的完整实现与事实标准。FastAPI 目前仅是最小可运行占位骨架，需要逐步对齐 Go Gin 的 DDD 结构与接口契约，以便先行与 Nuxt 完成联调验证。

## 原则

- **不修改**现有 Go Gin 与 Nuxt 代码（含路径、接口、行为与测试）。
- FastAPI 按 Go Gin 的结构与契约对齐，作为新增实现，而非替换。
- 产物仍以 `skeleton/` 为唯一真源，模板同步遵循现有脚本流程。

## 目标

- FastAPI skeleton 具备与 Go Gin 结构一致的 DDD 分层（目录与职责对齐）。
- FastAPI 对外接口最少实现一组可被 Nuxt 调用的基础 API（以 Go Gin 为准）。
- FastAPI 可在 Standalone 下独立启动，并保持与 Nuxt 的联调流程清晰可复用。

## 当前对齐进度（2026-01-29）

- DDD 分层：`models/repository/services/transport/router` 已按 Gin 结构落地。
- Handler 对齐：Admin Auth / IAM / Templates / Capabilities / Runtime Sessions / Manifest / RBAC 已按 Gin 行为对齐。
- 宿主反代：已支持 `/_p/{plugin_id}{api_prefix}` 与 `api_prefix` 同步挂载。

## 非目标

- 不重写 Go Gin 现有业务逻辑。
- 不在本阶段接入新数据库或改变权限模型。
- 不在本阶段替换或迁移 Nuxt 现有页面与 API 代码。

## 现状基线（对齐对象）

- Go Gin skeleton 根结构：`cmd/`、`internal/`、`etc/`、`tests/`、`go.mod`。
- DDD 分层集中在 `internal/`（`bootstrap/config/entity/services/transport/router/observability` 等）。
- 统一 HTTP 前缀、Envelope、RLS、鉴权/中间件顺序均有固定约定。
- 对齐基线来源：`skeleton/backend/go-gin/internal/**`、`skeleton/backend/go-gin/cmd/**`、`skeleton/backend/go-gin/etc/**`。
- FastAPI 现状仅有 `skeleton/backend/python-fastapi/app` 与启动脚本，尚无 DDD 分层。

## 对齐清单（FastAPI 要补齐）

### 1) 目录与分层

对齐 Go Gin 的 DDD 层级，建议 FastAPI 采用如下结构（可根据 Python 生态细化）：

```
skeleton/backend/python-fastapi/
  app/
    bootstrap/      # 配置加载、依赖装配
    config/         # 环境变量与配置模型
    entity/
      models/       # 领域模型/值对象
      repository/   # 仓储接口与默认实现
    services/       # 领域服务与用例编排
    transport/
      http/         # API 路由与 DTO 适配
    manifestx/      # 插件清单描述
    observability/  # 日志、指标、追踪
    middleware/     # 鉴权/RBAC/租户上下文
    router/         # 路由聚合与模块挂载
    server/         # 服务启动与生命周期
    contracts/      # HTTP 响应 Envelope 与错误码模型
    shared/         # 工具与通用组件
    main.py         # FastAPI 入口
  scripts/          # 启动脚本
  README.md
  requirements.txt
```

### 2) 接口契约

- **路径前缀**：以 `etc/config.yaml` 的 `server.api_prefix` 为准，默认 `/api/v1`，与 Go Gin 保持一致。
- **宿主反代**：同时挂载 `/_p/{plugin_id}{api_prefix}`，与 Gin 反代路径一致。
- **健康检查**：保留 `/healthz` 并与 Go Gin 输出结构一致。
- **响应 Envelope**：遵循现有 CRUD/REST 约定（保持字段名与分页结构一致）。
- **鉴权/租户**：沿用 tenant_uuid 注入与最小权限策略（由中间件注入上下文）。
- **RBAC**：前后端一致的权限码与动作集，FastAPI 不另造枚举。
- **错误码**：与 Go Gin 现有错误码保持一致（字段与语义一致）。

### 3) Nuxt 联调最小面

- 明确 Nuxt 现有 API 依赖，FastAPI 先实现对应的最小可用接口集。
- Nuxt 不做改动，FastAPI 适配 Nuxt 现有调用与错误处理格式。

### 4) 运维与配置

- 复用 `skeleton/plugin.yaml` 作为运行时清单来源。
- 配置路径与环境变量命名与 Go Gin 尽量对齐，确保文档一致。
- 运行方式与端口对齐 Go Gin（含本地开发端口、日志格式与输出目录）。
- **ORM**：统一使用 SQLAlchemy 2.0 + Alembic，模型与迁移对齐 Go Gin 表结构。

## 对齐清单（细化到 Go Gin 目录）

> 以 Go Gin 目录为参照，FastAPI 需要逐步补齐对应层级与职责。

- `internal/bootstrap` → `app/bootstrap`：配置加载、依赖装配、生命周期。
- `internal/config` → `app/config`：配置模型、默认值、环境变量映射。
- `internal/contracts` → `app/contracts`：APIResponse/错误码结构与分页模型。
- `internal/db` → `app/entity/repository`：数据库会话与仓储基类。
- `internal/entity/models` → `app/entity/models`：领域模型与表名常量。
- `internal/services/*` → `app/services/*`：领域服务与用例编排。
- `internal/transport/http` → `app/transport/http`：路由、DTO、Handler。
- `internal/router` → `app/router`：路由聚合、挂载前缀。
- `internal/middleware` → `app/middleware`：鉴权/RBAC/RLS 注入。
- `internal/observability` → `app/observability`：日志/指标/审计。
- `internal/manifestx` → `app/manifestx`：清单解析与能力声明。
- `internal/server` → `app/server`：HTTP Server 启动与健康检查。
- `internal/config` 的 `server.api_prefix` → `app/config`：API 前缀统一配置。

## 最小联调 API 范围（待确认）

> 以 Nuxt 现有调用为准，先列“清点路径”，再落地实现。

已从 `skeleton/web-admin/nuxt/app/composables/api/**` 扫描出当前调用路径（按 method+path 去重）：

- `GET /admin/capabilities`
- `GET /admin/capabilities/exposure/${encodeURIComponent(capabilityId)}`
- `PUT /admin/capabilities/exposure/${encodeURIComponent(payload.capability_id)}`
- `GET /admin/capabilities/exposure/template`
- `GET /admin/capabilities/lifecycle`
- `POST /admin/capabilities/lifecycle`
- `POST /admin/capabilities/lifecycle/${encodeURIComponent(planId)}/status`
- `GET /admin/capabilities/lifecycle/template`
- `GET /admin/capabilities/quotas/${encodeURIComponent(capabilityId)}`
- `POST /admin/capabilities/quotas/${encodeURIComponent(capabilityId)}`
- `POST /admin/capabilities/register`
- `GET /admin/capabilities/register/template`
- `POST /admin/capabilities/register/validate`
- `GET /admin/departments`
- `POST /admin/departments`
- `DELETE /admin/departments/${id}`
- `GET /admin/departments/${id}`
- `PUT /admin/departments/${id}`
- `GET /admin/departments/tree`
- `GET /admin/iam/departments`
- `POST /admin/iam/departments`
- `DELETE /admin/iam/departments/${id}`
- `PATCH /admin/iam/departments/${id}`
- `GET /admin/iam/departments/tree`
- `GET /admin/iam/departments?tenant_uuid=${encodeURIComponent(tenantUuid)}`
- `POST /admin/iam/members`
- `PATCH /admin/iam/members/${id}`
- `POST /admin/iam/members/import`
- `GET /admin/iam/members?{query}`
- `GET /admin/iam/permissions`
- `POST /admin/iam/permissions`
- `DELETE /admin/iam/permissions/${id}`
- `PUT /admin/iam/permissions/${id}`
- `GET /admin/iam/permissions/catalog`
- `POST /admin/iam/permissions/sync`
- `POST /admin/iam/roles`
- `DELETE /admin/iam/roles/${id}`
- `GET /admin/iam/roles/${id}`
- `PATCH /admin/iam/roles/${id}`
- `DELETE /admin/iam/roles/${id}/members`
- `POST /admin/iam/roles/${id}/members`
- `PUT /admin/iam/roles/${id}/permissions`
- `GET /admin/iam/roles?{query}`
- `POST /admin/iam/tenants`
- `GET /admin/iam/tenants/${id}`
- `PATCH /admin/iam/tenants/${id}`
- `GET /admin/iam/tenants?{query}`
- `GET /admin/permissions`
- `GET /admin/permissions/groups`
- `GET /admin/roles`
- `POST /admin/roles`
- `DELETE /admin/roles/${id}`
- `GET /admin/roles/${id}`
- `PUT /admin/roles/${id}`
- `POST /admin/roles/${roleId}/permissions`
- `POST /admin/runtime/sessions/${encodeURIComponent(sessionId)}/ack`
- `POST /admin/runtime/sessions/${encodeURIComponent(sessionId)}/close`
- `POST /admin/runtime/sessions/${encodeURIComponent(sessionId)}/heartbeat`
- `POST /admin/runtime/sessions/${encodeURIComponent(sessionId)}/invoke`
- `POST /admin/runtime/sessions/register`
- `POST /admin/user/auth/change-password`
- `POST /admin/user/auth/login`
- `POST /admin/user/auth/logout`
- `GET /admin/user/auth/me`
- `POST /admin/user/auth/me/avatar`
- `POST /admin/user/auth/me/check-permission`
- `GET /admin/user/auth/me/context`
- `GET /admin/user/auth/me/departments`
- `PUT /admin/user/auth/me/profile`
- `GET /admin/user/auth/me/roles`
- `POST /admin/user/auth/me/switch-tenant`
- `GET /admin/user/auth/me/tenants`
- `GET /admin/user/auth/permissions`
- `PUT /admin/user/auth/profile`
- `POST /admin/user/auth/refresh`
- `POST /admin/user/auth/register`
- `POST /admin/user/auth/reset-password`
- `POST /admin/user/auth/reset-password/confirm`
- `GET /admin/user/auth/validate`
- `GET /admin/users`
- `POST /admin/users`
- `DELETE /admin/users/${id}`
- `GET /admin/users/${id}`
- `PUT /admin/users/${id}`
- `PATCH /admin/users/${id}/status`
- `POST /admin/users/batch-delete`
- `GET /templates`
- `POST /templates`
- `DELETE /templates/${id}`
- `GET /templates/${id}`
- `PUT /templates/${id}`

### 已对齐（FastAPI）

- Admin Auth：`/admin/user/auth/*`（含 me/context、login/refresh/logout）已对齐 Gin 响应字段与错误码。
- Admin IAM：`/admin/iam/*`（含 tenants/roles/permissions/departments/members/audit/sts 入口）已对齐 Gin 行为与分页字段。
- Templates：`/templates/*` + `/admin/templates/*` 已补齐 batch-clone/validate 与 Gin 行为。
- Capabilities：`/admin/capabilities/*` 已补齐 reviews 占位与 Gin 路由一致。
- Runtime Sessions：`/admin/runtime/sessions/*` 校验字段与错误信息对齐 Gin。
- Manifest/RBAC：`/admin/manifest` 与 `/admin/rbac` 返回结构与 Gin 对齐。

1) 从 `web-admin` API 调用点清单化（已完成初次扫描）：  
   - 目标：人工校对路径、方法、字段、错误码与权限码映射。
2) 与 Go Gin 路由对照（来源 `skeleton/backend/go-gin/internal/transport/http/**`）。  
3) 生成 FastAPI 路由清单与 DTO 映射表（含字段名/分页结构）。

## 分阶段实施

### Phase A — 结构对齐（不触及业务）

- 建立 FastAPI 的 DDD 目录骨架。
- 迁移最小化启动逻辑与配置加载结构。
- 增加统一的日志、请求链路与基础中间件占位。
- 增加 contracts/envelope 与错误码结构（对齐 Go Gin）。

### Phase B — 契约对齐（面向 Nuxt 联调）

- 实现 `/api/v1` 前缀路由注册。
- 补齐与 Nuxt 联调所需的最小 API（以 Go Gin 为准）。
- 保持 Envelope/错误码格式一致。
- 先实现 `/healthz` 与基础鉴权中间件联调。

### Phase C — 渐进对齐 Go Gin 功能面

- 以领域模块为单位补齐 Service/Repository/Transport。
- 补齐必要的观测与权限检查。
- 引入对应的测试结构（单测 + 轻量集成）。

## 验证清单

- FastAPI 本地启动：`bash scripts/dev.sh` 后 `/healthz` 返回 200。
- Makefile 验证：默认 `BACKEND=gin`，FastAPI 需 `BACKEND=fastapi`；`make BACKEND=fastapi test-smoke` 可通过。
- 与 Nuxt 联调：Nuxt 不修改的前提下可调用 FastAPI 目标接口。
- 目录与分层结构与 Go Gin 对齐，且文档明确。
- routes/DTO/envelope 与 Go Gin 一致（字段与分页结构）。

## 风险与约束

- **严禁修改** Go Gin 与 Nuxt 现有实现。
- FastAPI 必须自行适配既有契约与路径，否则联调将失败。
- 若 Nuxt 依赖未公开的私有接口，需要从 Go Gin 路由与合同中明确清单。
- FastAPI 仅作为新增实现，不替换现有 Go Gin 流程。

## 实施状态（FastAPI 对齐）

- Phase 1（Setup）：已完成
- Phase 2（Foundational）：已完成
- Phase 3（US1 / MVP）：已完成
- Phase 4（US2 / 宿主模式）：已完成
- Phase 5（US3 / 数据结构）：已完成
