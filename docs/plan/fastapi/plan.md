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

## 非目标

- 不重写 Go Gin 现有业务逻辑。
- 不在本阶段接入新数据库或改变权限模型。
- 不在本阶段替换或迁移 Nuxt 现有页面与 API 代码。

## 现状基线（对齐对象）

- Go Gin skeleton 根结构：`cmd/`、`internal/`、`etc/`、`tests/`、`go.mod`。
- DDD 分层集中在 `internal/`（`bootstrap/config/entity/services/transport/router/observability` 等）。
- 统一 HTTP 前缀、Envelope、RLS、鉴权/中间件顺序均有固定约定。

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
    shared/         # 工具与通用组件
    main.py         # FastAPI 入口
  scripts/          # 启动脚本
  README.md
  requirements.txt
```

### 2) 接口契约

- **路径前缀**：统一 `/api/v1`，与 Go Gin 保持一致。
- **健康检查**：保留 `/healthz` 并与 Go Gin 输出结构一致。
- **响应 Envelope**：遵循现有 CRUD/REST 约定（保持字段名与分页结构一致）。
- **鉴权/租户**：沿用 tenant_uuid 注入与最小权限策略（由中间件注入上下文）。

### 3) Nuxt 联调最小面

- 明确 Nuxt 现有 API 依赖，FastAPI 先实现对应的最小可用接口集。
- Nuxt 不做改动，FastAPI 适配 Nuxt 现有调用与错误处理格式。

### 4) 运维与配置

- 复用 `skeleton/plugin.yaml` 作为运行时清单来源。
- 配置路径与环境变量命名与 Go Gin 尽量对齐，确保文档一致。

## 分阶段实施

### Phase A — 结构对齐（不触及业务）

- 建立 FastAPI 的 DDD 目录骨架。
- 迁移最小化启动逻辑与配置加载结构。
- 增加统一的日志、请求链路与基础中间件占位。

### Phase B — 契约对齐（面向 Nuxt 联调）

- 实现 `/api/v1` 前缀路由注册。
- 补齐与 Nuxt 联调所需的最小 API（以 Go Gin 为准）。
- 保持 Envelope/错误码格式一致。

### Phase C — 渐进对齐 Go Gin 功能面

- 以领域模块为单位补齐 Service/Repository/Transport。
- 补齐必要的观测与权限检查。
- 引入对应的测试结构（单测 + 轻量集成）。

## 验证清单

- FastAPI 本地启动：`bash scripts/dev.sh` 后 `/healthz` 返回 200。
- 与 Nuxt 联调：Nuxt 不修改的前提下可调用 FastAPI 目标接口。
- 目录与分层结构与 Go Gin 对齐，且文档明确。

## 风险与约束

- **严禁修改** Go Gin 与 Nuxt 现有实现。
- FastAPI 必须自行适配既有契约与路径，否则联调将失败。
- 若 Nuxt 依赖未公开的私有接口，需要从 Go Gin 路由与合同中明确清单。
