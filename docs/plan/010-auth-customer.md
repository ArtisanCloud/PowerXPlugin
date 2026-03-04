# 010-auth-customer 开发计划

## 背景

PowerX 插件需要在 Mini-App / 2C 场景下识别租户 + Customer 身份。`docs/guides/develop/auth/customer.md` 已给出两套鉴权模式（Skeleton / Delegated），本计划用于梳理开发范围、交付物和时间安排，确保插件在独立运行与宿主代理两种环境下都能完成 Customer 鉴权。

## 目标

1. **租户隔离**：受保护的 `/mini-app/*` 必须能解析到 `tenant_uuid`；tenant 可来自请求显式注入（网关/header/query）或来自 customer token（推荐）。若请求显式 tenant 与 token tenant 不一致，返回 `TENANT_MISMATCH`。
2. **统一上下文**：实现 `CustomerContext` + getter/setter，中间件写入 `gin.Context` 和 `request.Context`，Service 层可直接读取。
3. **双模支持**：Skeleton 模式使用本地 `customer_accounts`（`CustomerAccount`）和 JWT；Delegated 模式全部委托 PowerX 底座校验，不在插件中落库。
4. **统一响应**：mini-app handler 统一使用 `contracts.Response*`，符合 `.codex/skills/crud/api-rest/SKILL.md` 内嵌规则的 envelope。

## 范围与模式

| 模式 | 客户源 | Token 签发者 | 说明 |
| --- | --- | --- | --- |
| Skeleton | 插件 `customer_accounts` 表 | 插件 JWT/STS | 仅在独立运行时使用，支持注册/登录、密码哈希、token 颁发与校验。 |
| Delegated | PowerX 底座 CRM/IAM | 宿主 | 插件不记录客户账号，只调用宿主的校验接口获取 claims。 |

> 注意：`customer_accounts` 及本地登录 API 仅在 Skeleton 模式启用；接入宿主时所有 Customer 数据由 PowerX 底座维护。

## tenant_uuid 逻辑（对齐现有实现）

### 背景：多租户 + “token 穿刺”

- 宿主模式：网关通常会注入 `tenant_uuid`；插件仍会校验 token 中 tenant 是否一致。
- Standalone 模式：同一手机号/邮箱可能在多个租户注册，因此登录阶段允许“未指定租户”，并在 token 内携带 tenant，后续请求可仅带 token。

### 规则汇总

- `/mini-app/auth/register`：必须指定租户（header 优先，否则 body `tenant_uuid`；两者同时存在必须一致）。
- `/mini-app/auth/login`：
  - 未指定 tenant 且 login 只命中 1 个租户：自动选租户并签发 token（claims 含 `tenant_uuid`）。
  - 未指定 tenant 且命中多个租户：返回 `409 TENANT_SELECTION_REQUIRED` + `error.details.tenants[]`。
- `/mini-app/*` 受保护接口：
  - 允许不带 `tenant_uuid`，由 token 注入 tenant。
  - 若显式携带 `tenant_uuid`/query tenant 且与 token tenant 不一致：`403 TENANT_MISMATCH`。

## 架构与组件

1. **CustomerAuthenticator 接口**
   - `LocalCustomerAuthenticator`：解析本地 JWT 或处理邮箱/手机号登录，依赖 `customer_accounts`（`CustomerAccount`）。
   - `DelegateCustomerAuthenticator`：调用宿主 `/api/v1/customer/auth/validate`（具体以宿主开放接口为准），并将结果映射为 `CustomerContext`。
2. **CustomerContext & Middleware**
   - 新增 `customer.Context` struct，字段含 `customer_uuid/customer_id/roles/attributes/tenant_uuid`。
   - `customerhttp.Authenticate(authenticator)` 从 Header 读 token，调用 authenticator，校验 tenant，写入上下文。
3. **Mini-App Router**
   - `/mini-app` 入口统一挂载：`customerhttp.Authenticate()` → `httpmw.EnsureTenant()` → 业务 handler。
   - 原因：Authenticate 成功后会把 token 中的 `tenant_uuid` 注入到上下文，确保 EnsureTenant 在“仅带 token”的场景下也能通过。
4. **Skeleton 模式 API**
   - `/mini-app/auth/register`、`/mini-app/auth/login`、`/mini-app/auth/profile` 等接口。
   - 密码哈希使用 `bcrypt` 或 `argon2`。
   - JWT issuer/audience 支持专用配置（如 `POWERX_CUSTOMER_JWT_ISSUER/SECRET`）。
5. **Delegated 模式流程**
   - 客户端从宿主登录入口获取 token。
   - 插件中间件调用宿主校验端点，验证成功后将 claims 写入 `CustomerContext`。
   - 可选：使用 Redis 缓存校验结果，TTL 受限于 token `exp`。

## 任务拆解

### 1. 规格与配置
- 在 `specs/**` 中补充 mini-app auth 相关 API 定义。
- 更新 `backend/etc/config.*.yaml`，新增 `customerAuth` 配置（mode、delegate endpoint、JWT issuer/secret）。
- `plugin.yaml` 增加 mini-app capability scope 描述。

### 2. 数据与 Repository（Skeleton 模式）
- 新增 `customer_accounts` 表（`CustomerAccount`），用于存储本地 Customer 登录凭证（email/phone/password_hash/status/metadata 等）。
- 新建 `internal/entity/repository/customer`，封装 CRUD / 按租户检索。
- 支持登录前的“跨租户查询”能力（按 email/phone 列出候选 tenant_uuid），仅用于 tenant 选择。

### 3. Service 层
- `internal/services/customer/auth/local_service.go`：注册、登录、token 颁发、密码校验。
- `internal/services/customer/auth/delegate_service.go`：封装宿主 HTTP 校验、错误映射、缓存策略。
- 工厂方法 `NewCustomerAuthenticator(cfg)` 根据配置返回 Local 或 Delegate。

### 4. Middleware 与 Context
- `internal/middleware/customer_context.go`：定义 `CustomerContext`、`FromContext`、`SetContext`。
- `internal/transport/http/middleware/customer_auth.go`：实现 `Authenticate` 中间件，支持 Header/Query 取 token，失败统一返回 `ResponseUnauthorized`。

### 5. HTTP Handler
- `backend/internal/transport/http/mini-app/customer_handler.go`：注册/登录等 API，返回 envelope；login 支持 `login` / `identifier` 字段别名。
- 更新所有 mini-app handler，使用 `customerctx.FromGin(c)` 获取当前客户。

### 6. 测试
- 单元测试：Local authenticator（正确/错误密码、租户 mismatch）、Delegate authenticator（宿主返回 2xx/4xx）。
- 集成测试：mini-app handler，在 Skeleton 模式下完成注册→登录→访问受保护接口。
- 若使用缓存，增加 TTL 和失效测试。

## 风险与缓解

| 风险 | 说明 | 缓解 |
| --- | --- | --- |
| 宿主接口变化 | Delegated 模式依赖宿主提供的 token 校验端点 | 在配置中允许 endpoint/headers 自定义，封装重试与超时 |
| 密码/Token 安全 | Skeleton 模式需自行储存账号密码 | 强制使用标准哈希算法，token 生命周期 ≤ 2h，可选 refresh token |
| 租户隔离 | 忘记校验 tenant_uuid 会导致越权 | 中间件级别保证 tenant 校验，Service 层再兜底 |
| 双模式切换 | 部署环境变化可能忘记切换配置 | 在启动日志输出当前模式，提供健康检查项 |

## 交付清单

1. `CustomerAuthenticator` 及两种实现。
2. `CustomerContext` 中间件及 mini-app router 集成。
3. Skeleton 模式下的注册/登录 API + JWT 颁发。
4. Delegated 模式下的宿主校验客户端。
5. 文档与配置示例，涵盖两种模式的启动方式。

## 观测与告警（Polish）

### 建议关注的日志

- `customer.auth.validation`：token 校验结果（含 `tenant_uuid/customer_uuid/request_id/latency_ms/mode/ok`）
- `customer.auth.login`：Skeleton 登录（含 `tenant_uuid/customer_uuid/request_id/latency_ms/ok`）
- `customer.auth.register`：Skeleton 注册（含 `tenant_uuid/customer_uuid/request_id/latency_ms/ok`）

### 建议关注的指标

以下指标会出现在 Admin Runtime Metrics（Prometheus exposition）中：

- `powerx_customer_auth_validation_total{plugin_id,mode,result}`
- `powerx_customer_auth_validation_latency_ms_sum{plugin_id,mode,result}` / `powerx_customer_auth_validation_latency_ms_count{...}`
- `powerx_customer_auth_login_total{plugin_id,result}`
- `powerx_customer_auth_login_latency_ms_sum{plugin_id,result}` / `powerx_customer_auth_login_latency_ms_count{...}`

### 告警建议（示例）

- Delegated 模式 `result="error"` 或 `503 SERVICE_UNAVAILABLE` 占比异常升高：提示宿主校验端点不可用或网络异常
- `TENANT_MISMATCH` 突增：提示前端 tenant 注入/路由绑定错误或 token 复用

## 时间预估（示例）

| 阶段 | 内容 | 预计 |
| --- | --- | --- |
| 需求澄清 & Spec 补齐 | 补充接口、配置、数据结构 | 1d |
| 数据/Service/Middleware 实现 | 包含 Local & Delegate 两条链路 | 3d |
| HTTP Handler & 集成 | 注册/登录 + mini-app 接入 | 2d |
| 测试与文档 | 单测/集成/操作指南 | 1d |

> *时间仅为规划参考，具体以迭代排期为准。*
