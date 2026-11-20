---
name: crud-http
description: 基于内嵌规则 的 CRUD over HTTP 顶层规范。用于校对 HTTP 路由前缀、响应 envelope、中间件栈、RLS 注入与 Service 复用。
---

# CRUD over HTTP

## 步骤

1) 打开 `本文件内嵌规则`。
2) 按 ruleset 的 requires 联动检查 HTTP/DTO/Service/Repository/Model/Migration/DI/Test。

## 核对点

- 路由前缀可配置，默认 `/api/v1`，并挂载管理端点。
- 响应符合统一 envelope 与分页结构。
- 中间件顺序：request_id → ctx_verify → rbac → tenant_ctx → recovery/logging → throttle。
- Handler 仅做校验/鉴权/调用 Service，不落业务逻辑。
- DB 会话注入 `tenant_uuid`（RLS）。


## 指南

# CRUD over HTTP — Plugin Guide

> 面向插件侧（PowerXPlugin）后端 HTTP 实现的人读说明。  
> 目标：统一路由/响应/中间件/RBAC/RLS 等约定；保证与宿主反代契合（PG-HOST-001/PG-CTX-001/PG-RBAC-001/PG-SVC-001）。

## 1. 路由与前缀

- **业务前缀**：默认 `/api/v1`，可通过 `server.api_prefix` 配置覆盖。  
- **宿主反代**：`/_p/<plugin-id>/api/* → backend <api_prefix>/**`（PG-HOST-001）。  
- **典型资源**（以 `template` 为例）：
  - `GET    /api/v1/templates`（分页/筛选）
  - `GET    /api/v1/templates/:id`
  - `POST   /api/v1/templates`
  - `PUT    /api/v1/templates/:id`
  - `DELETE /api/v1/templates/:id`

> 管理端点另见：`/api/v1/admin/{manifest, rbac}`（合规要求，非业务 CRUD）。

## 2. 统一响应与错误模型

- **Envelope（contracts.APIResponse）**：

  ```json
  {
    "success": true,
    "message": "",
    "data": { ... },
    "error": null,
    "timestamp": "2024-12-09T12:00:00Z",
    "request_id": "rq-123"
  }
  ```

* **分页响应**：`data` 字段内返回 `{ "items": [...], "total": 135, "page": 1, "page_size": 20 }`
* **错误响应**（示例）：

  ```json
  {
    "success": false,
    "error": {
      "code": "PERMISSION_DENIED",
      "message": "permission denied",
      "details": { "resource": "base:template", "action": "read" }
    },
    "timestamp": "2024-12-09T12:00:05Z",
    "request_id": "rq-123"
  }
  ```

> 约定与 `rulesets/crud/api_rest.yaml` 保持一致；如需扩展字段（如 trace_id），保持向后兼容。

## 3. 中间件栈（顺序建议）

1. `request_id` — 生成/透传请求 ID
2. `ctx_verify` — **JWT/HMAC 验签**，抽取 `tenant_uuid/user_id/permissions/...`（PG-CTX-001）
3. `rbac_guard` — **服务端**权限判定（PG-RBAC-001）
4. `tenant_ctx` — 设置 DB 会话变量：`SET LOCAL app.tenant_uuid = ?`（PG-CTX-001/RLS）
5. `recovery/logging` — 统一结构化日志与 panic 保护
6. `throttle/ratelimit` — 可选

> Handler 保持**薄**：入参校验 → 权限检查 → 调用 Service → 组装响应（PG-SVC-001）。

## 4. DTO / 校验 / 查询规范

* **Create/Update**：强校验（必填、长度、格式）
* **List**：仅允许白名单字段排序；`page/page_size` 上限保护
* **过滤**：约定 `q` 为模糊关键字；更多字段请显式列出
* **示例（Template）**：

  ```json
  // Create
  { "name": "Welcome", "description": "Landing snippet", "content": "# Hello" }
  ```

## 5. Service 复用与幂等

* HTTP/gRPC/MCP 等**复用同一 Service**（PG-SVC-001）。
* 幂等建议：以资源自然键或入参关键字段（如 `name`）进行去重与冲突检测；写操作返回完整资源。

## 6. 路由注册与项目结构（示例）

```
backend/
└── internal/
    ├── services/template/
    ├── domain/repository/template/
    └── transport/http/template/
        ├── routes.go         # 注册 /api/v1/templates...（随前缀配置）
        ├── handler_list.go
        ├── handler_get.go
        ├── handler_create.go
        ├── handler_update.go
        └── handler_delete.go
```

## 7. 测试策略

* **Contract Test**：路由/响应码/响应体
* **Integration Test**：多租户隔离（不同 tenant_uuid 不可互访）
* **Migration Smoke**：如涉及表结构变更

## 8. 合规清单（Checklist）

* [ ] 路由全部置于配置的 API 前缀（默认 `/api/v1`）
* [ ] 响应 envelope 与分页结构符合规范
* [ ] 中间件完成验签/RBAC/RLS 注入
* [ ] Handler 薄；业务仅在 Service
* [ ] 写操作具备幂等/冲突检测
* [ ] 有合同/集成/迁移冒烟测试

（相关 Gates：PG-HOST-001 / PG-CTX-001 / PG-RBAC-001 / PG-SVC-001）

## 规则（内嵌）

### crud_http.yaml

```yaml
# CRUD over HTTP — Top Ruleset
# Gates: PG-HOST-001 / PG-CTX-001 / PG-RBAC-001 / PG-SVC-001
kind: ruleset
id: crud_http
version: 1.0.0
summary: "插件后端 HTTP 约束：统一路由、响应、鉴权、RLS 与 Service 复用"
requires:
  - rulesets/crud/api_rest.yaml
  - rulesets/crud/handler_http.yaml
  - rulesets/crud/dto.yaml
  - rulesets/crud/service.yaml
  - rulesets/crud/repository.yaml
  - rulesets/crud/model.yaml
  - rulesets/crud/migration.yaml
  - rulesets/crud/di.yaml
  - rulesets/crud/test.yaml
checks:
  - id: http-prefix-configurable
    desc: API 分组使用可配置前缀，默认 /api/v1，并统一挂载管理端点
    assert:
      type: grep
      target: backend/internal/router/router.go
      pattern:
        - "prefix := r.cfg.Server.APIPrefix"
        - "if strings.TrimSpace(prefix) == \"\" {"
        - "prefix = \"/api/v1\""
        - "gApi := r.engine.Group(prefix)"
  - id: http-envelope
    desc: 响应使用统一 envelope 与分页结构
    assert:
      type: schema
      target: backend/internal/transport/http/**
      schema: refs:rulesets/crud/api_rest.yaml#envelope
  - id: http-mw-stack
    desc: 中间件顺序：request_id→ctx_verify→rbac→tenant_ctx→recovery/logging→throttle
    assert:
      type: grep
      target: backend/internal/transport/http/server.go
      pattern:
        - request_id
        - ctx_verify
        - rbac
        - tenant_ctx
        - recovery
  - id: http-service-slim
    desc: Handler 仅做校验/鉴权/调用 Service；业务逻辑不得落在 handler 中
    assert:
      type: ban-symbol
      target: backend/internal/transport/http/**
      symbols:
        - direct_db_access
        - business_flow_in_handler
  - id: http-rls
    desc: DB 会话必须注入 tenant_uuid（RLS）
    assert:
      type: grep
      target: backend/internal/**
      pattern:
        - "SET LOCAL app.tenant_uuid"
```
