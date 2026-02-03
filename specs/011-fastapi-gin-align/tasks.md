# Tasks: FastAPI 对齐 Go Gin（以 Nuxt 联调为第一目标）

**Input**: Design documents from `/specs/011-fastapi-gin-align/`
**Prerequisites**: plan.md (required), spec.md (required), research.md, data-model.md, contracts/

**Tests**: 未在规格中强制要求，任务内不包含测试项。

**Organization**: 任务按用户故事组织，保证每个故事可独立交付与验证。

## Format: `[ID] [P?] [Story] Description`

## 执行顺序（最小可联通 → 可用）

1) 认证链路（AuthService + `/admin/user/auth/*`）
2) IAM 核心（tenants/roles/permissions/departments/members）
3) 模板 CRUD（TemplateService + `/admin/templates/*`）
4) 能力管理（CapabilityService + `/admin/capabilities/*`）
5) 运行时会话（RuntimeSessionService + `/admin/runtime/sessions/*`）
6) 数据库落地（模型字段对齐 + Alembic 迁移）

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: 项目初始化与基础结构落地

- [x] T001 创建 FastAPI 目录结构骨架于 `skeleton/backend/python-fastapi/app/`
- [x] T002 更新依赖清单于 `skeleton/backend/python-fastapi/requirements.txt`（补齐 SQLAlchemy 2.0 + Alembic）
- [x] T003 [P] 初始化应用入口与应用工厂于 `skeleton/backend/python-fastapi/app/main.py`
- [x] T004 [P] 初始化配置模型与加载逻辑于 `skeleton/backend/python-fastapi/app/config/settings.py`
- [x] T005 [P] 初始化统一响应封装于 `skeleton/backend/python-fastapi/app/contracts/response.py`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: 所有用户故事依赖的基础设施

- [x] T006 初始化数据库会话与基础仓储于 `skeleton/backend/python-fastapi/app/entity/repository/db.py`
- [x] T007 初始化 Alembic 配置与迁移入口于 `skeleton/backend/python-fastapi/alembic.ini`
- [x] T008 初始化迁移环境与版本目录于 `skeleton/backend/python-fastapi/migrations/env.py`
- [x] T009 [P] 增加租户上下文中间件于 `skeleton/backend/python-fastapi/app/middleware/tenant_context.py`
- [x] T010 [P] 增加鉴权与权限校验中间件于 `skeleton/backend/python-fastapi/app/middleware/auth_guard.py`
- [x] T011 [P] 增加日志与请求链路基础实现于 `skeleton/backend/python-fastapi/app/observability/logging.py`
- [x] T012 初始化路由聚合与前缀配置于 `skeleton/backend/python-fastapi/app/router/api.py`
- [x] T013 增加健康检查端点于 `skeleton/backend/python-fastapi/app/transport/http/health.py`

**Checkpoint**: 基础设施完成，可开始用户故事实现

---

## Phase 3: User Story 1 - Nuxt 管理端可直接联调 (Priority: P1) 🎯 MVP

**Goal**: 覆盖 P1 联调范围的管理端接口（认证、IAM 核心、模板、能力管理、运行时会话）。

**Independent Test**: Nuxt 管理端在不改动代码的前提下完成登录、列表展示与核心操作。

### Implementation for User Story 1

- [x] T014 [P] 创建 Tenant 模型于 `skeleton/backend/python-fastapi/app/entity/models/tenant.py`
- [x] T015 [P] 创建 User 模型于 `skeleton/backend/python-fastapi/app/entity/models/user.py`
- [x] T016 [P] 创建 Member 模型于 `skeleton/backend/python-fastapi/app/entity/models/member.py`
- [x] T017 [P] 创建 Role 模型于 `skeleton/backend/python-fastapi/app/entity/models/role.py`
- [x] T018 [P] 创建 Permission 模型于 `skeleton/backend/python-fastapi/app/entity/models/permission.py`
- [x] T019 [P] 创建 Department 模型于 `skeleton/backend/python-fastapi/app/entity/models/department.py`
- [x] T020 [P] 创建 Template 模型于 `skeleton/backend/python-fastapi/app/entity/models/template.py`
- [x] T021 [P] 创建 Capability 模型于 `skeleton/backend/python-fastapi/app/entity/models/capability.py`
- [x] T022 [P] 创建 RuntimeSession 模型于 `skeleton/backend/python-fastapi/app/entity/models/runtime_session.py`
- [x] T023 [US1] 实现认证服务于 `skeleton/backend/python-fastapi/app/services/auth_service.py`
- [x] T024 [US1] 实现 IAM 服务于 `skeleton/backend/python-fastapi/app/services/iam_service.py`
- [x] T025 [US1] 实现模板服务于 `skeleton/backend/python-fastapi/app/services/template_service.py`
- [x] T026 [US1] 实现能力管理服务于 `skeleton/backend/python-fastapi/app/services/capability_service.py`
- [x] T027 [US1] 实现运行时会话服务于 `skeleton/backend/python-fastapi/app/services/runtime_session_service.py`
- [x] T028 [US1] 实现认证路由于 `skeleton/backend/python-fastapi/app/transport/http/admin/auth.py`
- [x] T029 [US1] 实现 IAM 路由于 `skeleton/backend/python-fastapi/app/transport/http/admin/iam.py`
- [x] T030 [US1] 实现模板 CRUD 路由于 `skeleton/backend/python-fastapi/app/transport/http/admin/templates.py`
- [x] T031 [US1] 实现能力管理路由于 `skeleton/backend/python-fastapi/app/transport/http/admin/capabilities.py`
- [x] T032 [US1] 实现运行时会话路由于 `skeleton/backend/python-fastapi/app/transport/http/admin/runtime_sessions.py`
- [x] T033 [US1] 挂载管理端路由至 API 前缀于 `skeleton/backend/python-fastapi/app/router/api.py`

**Checkpoint**: P1 管理端联调范围可用

---

## Phase 4: User Story 2 - 宿主模式接口一致 (Priority: P2)

**Goal**: 宿主反代路径下业务接口与管理端点规范一致。

**Independent Test**: 通过宿主反代路径请求管理端点与业务接口均可返回一致响应。

### Implementation for User Story 2

- [x] T034 [US2] 实现管理端 manifest/rbac 端点于 `skeleton/backend/python-fastapi/app/transport/http/admin/manifest.py`
- [x] T035 [US2] 统一宿主反代路径映射与前缀解析于 `skeleton/backend/python-fastapi/app/router/api.py`
- [x] T036 [US2] 记录宿主模式请求上下文于 `skeleton/backend/python-fastapi/app/observability/host_context.py`

**Checkpoint**: 宿主模式路径规范与 Gin 一致

---

## Phase 5: User Story 3 - 数据结构一致性 (Priority: P3)

**Goal**: 首阶段联调所需实体表结构与 Gin 对齐。

**Independent Test**: 对齐实体的表结构对比一致。

### Implementation for User Story 3

- [x] T037 [US3] 对齐关键实体字段与表名于 `skeleton/backend/python-fastapi/app/entity/models/*.py`
- [x] T038 [US3] 生成首阶段迁移脚本于 `skeleton/backend/python-fastapi/migrations/versions/`
- [x] T039 [US3] 补齐迁移执行入口文档于 `specs/011-fastapi-gin-align/quickstart.md`

**Checkpoint**: 关键实体结构对齐完成

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: 跨故事统一项与交付准备

- [x] T040 [P] 同步 OpenAPI 合同说明于 `specs/011-fastapi-gin-align/contracts/openapi.yaml`
- [x] T041 [P] 更新对齐说明于 `docs/plan/fastapi/plan.md`
- [x] T042 运行 quickstart 验证并记录结果于 `specs/011-fastapi-gin-align/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: 无依赖
- **Foundational (Phase 2)**: 依赖 Phase 1 完成，阻塞所有用户故事
- **User Stories (Phase 3-5)**: 依赖 Phase 2 完成
- **Polish (Phase 6)**: 依赖所有用户故事完成

### User Story Dependencies

- **US1 (P1)**: 可在 Phase 2 后立即开始
- **US2 (P2)**: 依赖 US1 的管理端基础结构，但可并行部分执行
- **US3 (P3)**: 与 US1 并行，依赖基础模型与迁移框架

### Parallel Opportunities

- Phase 1/2 中标记 [P] 的任务可并行
- US1 内各模型任务可并行
- US2 与 US3 可在 US1 基础完成后并行推进

---

## Parallel Example: User Story 1

```bash
# 并行创建模型
T014 & T015 & T016 & T017 & T018 & T019 & T020 & T021 & T022
# 服务与路由分组并行
T023 & T024 & T025 & T026 & T027
T028 & T029 & T030 & T031 & T032
```

---

## Implementation Strategy

- 先完成基础设施，再完成 US1 作为 MVP。
- US2/US3 并行推进，避免阻塞联调。
- 以 Go Gin 契约为权威基线，任何不一致均以 Gin 为准。


## 联通实现任务清单（按顺序执行）

- [x] T043 对齐认证链路：实现 AuthService 实际逻辑（登录/注册/刷新/获取当前用户）在 `skeleton/backend/python-fastapi/app/services/auth_service.py`
- [x] T044 对齐认证链路：补齐 /admin/user/auth/* 路由返回字段与错误码在 `skeleton/backend/python-fastapi/app/transport/http/admin/auth.py`
- [x] T045 对齐 IAM 核心：实现 tenants/roles/permissions/departments/members 读写逻辑在 `skeleton/backend/python-fastapi/app/services/iam_service.py`
- [x] T046 对齐 IAM 核心：补齐 /admin/iam/* 路由返回字段与错误码在 `skeleton/backend/python-fastapi/app/transport/http/admin/iam.py`
- [x] T047 对齐模板 CRUD：实现模板读写逻辑在 `skeleton/backend/python-fastapi/app/services/template_service.py`
- [x] T048 对齐模板 CRUD：补齐 /admin/templates/* 路由返回字段与错误码在 `skeleton/backend/python-fastapi/app/transport/http/admin/templates.py`
- [x] T049 对齐能力管理：实现 capabilities register/lifecycle/exposure/quotas 逻辑在 `skeleton/backend/python-fastapi/app/services/capability_service.py`
- [x] T050 对齐能力管理：补齐 /admin/capabilities/* 路由返回字段与错误码在 `skeleton/backend/python-fastapi/app/transport/http/admin/capabilities.py`
- [x] T051 对齐运行时会话：实现 register/ack/heartbeat/close/invoke 逻辑在 `skeleton/backend/python-fastapi/app/services/runtime_session_service.py`
- [x] T052 对齐运行时会话：补齐 /admin/runtime/sessions/* 路由返回字段与错误码在 `skeleton/backend/python-fastapi/app/transport/http/admin/runtime_sessions.py`
- [x] T053 对齐数据库结构：校对模型字段/类型与 Gin 一致并更新 `skeleton/backend/python-fastapi/app/entity/models/*.py`
- [x] T054 对齐数据库迁移：更新 Alembic 迁移脚本以匹配字段在 `skeleton/backend/python-fastapi/migrations/versions/0001_init.py`

## DDD 基础层对齐任务清单（先 Model → Repository → Service）

> 约束：遵循 Constitution 的 DDD 分层（Model/Repository/Service），先补齐领域模型，再补齐 Repository，再补齐 Service，最后再回到 Handler/API。

- [x] T055 对齐 Model（Privacy/Security/ToolGrant/Integration/Marketplace/Operations/AdminConsole/Customer）：按 Gin 字段与表名补齐 `skeleton/backend/python-fastapi/app/entity/models/*.py`
- [x] T056 对齐 Repository（Privacy/Security/ToolGrant/Integration/Marketplace/Operations/AdminConsole/Customer）：补齐 `skeleton/backend/python-fastapi/app/entity/repository/*` 的数据访问骨架与查询入口
- [x] T057 对齐 Service（Privacy/Security/ToolGrant/Integration/Marketplace/Operations/AdminConsole/Customer）：补齐 `skeleton/backend/python-fastapi/app/services/*` 的领域服务骨架与关键方法

- [x] T058 对齐 Model（Capability/RuntimeOps 细节字段）：按 Gin 字段补齐 `skeleton/backend/python-fastapi/app/entity/models/capability.py` 与 `runtime_session.py`
- [x] T059 对齐 Repository（Capability/RuntimeOps）：补齐 `skeleton/backend/python-fastapi/app/entity/repository/*` 能力/运行时查询入口
- [x] T060 对齐 Service（Capability/RuntimeOps）：补齐 `skeleton/backend/python-fastapi/app/services/*` 能力/运行时核心逻辑骨架

- [x] T061 对齐 Model（Plugin/Tenant 扩展）：补齐 `plugin_tenant_ext`、`plugin_credentials` 相关字段
- [x] T062 对齐 Repository（Plugin/Tenant 扩展）：补齐 `skeleton/backend/python-fastapi/app/entity/repository/*` 扩展表访问入口
- [x] T063 对齐 Service（Plugin/Tenant 扩展）：补齐 `skeleton/backend/python-fastapi/app/services/*` 扩展表服务层

## Handler 对齐任务清单（按 Gin 行为）

> 目标：HTTP Handler 行为/响应与 Gin 对齐，保持路径、参数、分页、错误码与 envelope 一致。

- [x] T064 对齐 Handler（Admin Auth）：/admin/user/auth/* 请求参数与响应字段一致
- [x] T065 对齐 Handler（Admin IAM）：/admin/iam/* 列表/详情/创建/更新/删除与 Gin 一致
- [x] T066 对齐 Handler（Admin Templates）：/admin/templates/* CRUD 与 Gin 一致
- [x] T067 对齐 Handler（Admin Capabilities）：/admin/capabilities/*（register/lifecycle/exposure/quotas）与 Gin 一致
- [x] T068 对齐 Handler（Admin Runtime Sessions）：/admin/runtime/sessions/*（register/ack/heartbeat/close/invoke）与 Gin 一致
- [x] T069 对齐 Handler（Admin Manifest/RBAC）：/admin/manifest & /admin/rbac 与 Gin 一致
- [x] T070 对齐 Handler（Host 反代路径）：宿主前缀解析与 Gin 路由映射一致

## 业务模块对齐任务清单（Gin 现有模块补齐）

> 目标：补齐 Gin 已实现模块在 FastAPI 的 Handler + Service 逻辑，实现可联调与可用。

- [x] T071 对齐 Integration 模块：/admin/integration/* + /integration/* Handler 与 Service 逻辑
- [x] T072 对齐 Operations 模块：/admin/operations/* Handler 与 Service 逻辑
- [x] T073 对齐 Marketplace 模块：/admin/marketplace/* + public marketplace Handler 与 Service 逻辑
- [x] T074 对齐 Security 模块：/admin/security/* Handler 与 Service 逻辑
- [x] T075 对齐 Privacy 模块：/admin/privacy/* Handler 与 Service 逻辑
- [x] T076 对齐 ToolGrant 模块：/admin/tool-grant/* Handler 与 Service 逻辑

## 运行时与中间件一致性任务

- [x] T077 对齐 RBAC 中间件逻辑（权限判定、默认拒绝策略）与 Gin
- [x] T078 对齐 Tenant Context 注入与校验（tenant_uuid/roles/permissions）

## 补齐项：中间件与注册机制缺口

- [x] T079 对齐 RequestTrace 中间件（request_id 生成/透传）与 Gin
- [x] T080 对齐 Host Context 注入（x-powerx-proxy / x-plugin-id）与 Gin
- [x] T081 对齐 DevSwitch 开发态注入（非生产 & 非代理时注入超管上下文）
- [x] T082 对齐 RBAC 路由权限注册机制（Registry → RoutePermissions 注入）
- [x] T083 对齐 TenantGuard（强制 tenant_uuid 校验与注入）

## 补齐项：其它路由与能力对齐

- [x] T084 对齐 Dev Console 路由（/admin/dev-console/*）与 Gin
- [x] T085 对齐 Runtime Ops 扩展路由（bootstrap/quota/metrics/event-bridge）与 Gin
- [x] T086 对齐 Agent 路由（credentials/STS/security）与 Gin
- [x] T087 对齐 Mini-App 路由（auth/ping/templates）与 Gin
- [x] T088 对齐 Dev Assets 路由（/assets/builds/meta）与 Gin
- [x] T089 对齐 Agent 凭证加密（AES-GCM + AAD + server.secret_key）与 Gin
- [x] T090 对齐 Agent 安全端点（privacy/toolgrant）逻辑与 Gin
- [x] T091 对齐 Agent STS Exchange 错误语义（未配置时返回内部错误）
- [x] T092 对齐 Agent STS Exchange 实现（HTTP STS endpoint + client_id/secret）
- [x] T093 对齐 Admin ToolGrant 行为（204 返回/limit/toolgrant_id/usage 记录）
- [x] T094 对齐 Admin Security/Privacy 过滤与响应（status/event_type/limit/204/格式）
- [x] T095 对齐 STS TokenManager（缓存/刷新/单例管理）
