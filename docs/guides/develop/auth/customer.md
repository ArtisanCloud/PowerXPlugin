# Customer 鉴权策略（Mini-App / 2C 场景）

> 适用范围：PowerX 插件在 C 端（mini-app）暴露产品/下单等接口，需要识别租户 + Customer 身份。脚手架在 Skeleton/local 模式内置 `customer.CustomerAccount`（表 `customer_accounts`）用于本地账号体系；部署在宿主（PowerX，Delegated 模式）时则复用宿主 CRM 的 Customer 校验能力。

## 模式划分

| 模式 | 触发条件 | Customer 记录来源 | Token 签发者 | 适配说明 |
| --- | --- | --- | --- | --- |
| **Delegated（宿主）** | `POWERX_PROXY=1` 或明示 `POWERX_CUSTOMER_DELEGATE=true` | PowerX 底座 CRM / IAM | 宿主安全域 | 插件不落地 Credential，仅校验宿主颁发的 customer token，并将 claims 转写到 `TenantContext`。 |
| **Skeleton（Standalone）** | 无宿主反代；`POWERX_CUSTOMER_DELEGATE` 关闭 | 插件维护的 `customer_accounts` 表（`CustomerAccount`） | 插件自身的 JWT/STS 服务 | 本地注册/登录签发 customer token；token 内携带 `tenant_uuid`，后续请求可仅携带 token（不强依赖 `X-Tenant-UUID`）。 |

> **重要**：`customer_accounts`（本地 Customer 账号）及相关注册/登录接口只会在 Skeleton 模式启用，方便插件独立运行或本地调试时拥有客户体系；当插件部署在宿主（Delegated 模式）时，经由 PowerX 底座 CRM/IAM 颁发和验证 customer token，插件侧不会创建/维护任何 Customer 鉴权表结构，所有 Customer 数据均在宿主侧。

### 配置入口（Skeleton / Delegated 共用）

Skeleton 后端配置文件示例位于 `skeleton/backend/go-gin/etc/config.example.yaml`，其中新增 `customer_auth` 段落用于切换模式：

- `customer_auth.mode: local`：启用本地 Customer（Skeleton）注册/登录与 JWT 签发；
- `customer_auth.mode: delegate`：启用宿主委托校验（Delegated），需要配置 `customer_auth.delegate_endpoint`。

## 目标能力

1. **租户隔离**：受保护的 `/mini-app/*` 继续要求存在 tenant 上下文；该上下文可来自 `X-Tenant-UUID`/query，也可由 customer token 的 `tenant_uuid` 注入。若请求已显式携带 tenant 且与 token tenant 不一致，返回 `TENANT_MISMATCH`。
2. **统一上下文**：定义 `CustomerContext`（`customer_uuid/customer_id/roles/attributes`），通过 `authx.SetTenantContext` 或新的 `SetCustomerContext` 挂入 `gin.Context`，Service 层可通过 `customer.ContextFrom(ctx)` 获取。
3. **统一返回**：所有 mini-app handler 使用 `contracts.Response*` 封装（参考 `skeleton/backend/go-gin/internal/transport/http/mini-app/template_handler.go`），以满足 `.codex/skills/crud/api-rest/SKILL.md` 内嵌规则的 envelope 约定。

## tenant_uuid 的来源与规则（对齐现有实现）

### 来源优先级（从高到低）

对 `/mini-app/*` 受保护接口：

1. **Customer token**：`Authorization: Bearer <token>`（或 `X-Customer-Token`）内的 `tenant_uuid`（推荐，适用于 standalone 与宿主网关两种模式）。
2. **请求显式 tenant**：`X-Tenant-UUID` header 或 query `tenant_uuid`（主要用于宿主网关显式注入、或调试场景）。

> `EnsureTenant()` 在最终阶段只要求“能解析到 tenant”，并不强制必须来自 header；因此 **mini-app 请求可以只带 token**（tenant 会被中间件注入）。

### 一致性校验（TENANT_MISMATCH）

- 若请求显式携带 `X-Tenant-UUID`/query `tenant_uuid`，同时 token 里也有 `tenant_uuid`，且两者不一致：返回 `403 TENANT_MISMATCH`。
- 若请求未显式携带 tenant：以 token 的 tenant 为准，并写入 `TenantContext` 供后续 repo/service 使用。

### 登录/注册时的 tenant 规则

- `POST /mini-app/auth/register`：必须明确目标租户（优先 `X-Tenant-UUID`，否则 body `tenant_uuid`；两者同时存在必须一致）。
- `POST /mini-app/auth/login`：
  - 可不传 `tenant_uuid`（header/body 都可以省略）；
  - 若该 `login` 仅存在于一个租户：自动选该租户并签发 token（token 内携带 `tenant_uuid`）；
  - 若该 `login` 在多个租户存在且未指定 `tenant_uuid`：返回 `409 TENANT_SELECTION_REQUIRED`，并在 `error.details.tenants[]` 返回候选租户列表，客户端选择后带 `tenant_uuid` 再次登录即可。

## 关键组件

### CustomerAuthenticator 接口

```go
// backend/internal/services/customer/authenticator.go
// 伪代码
 type CustomerAuthenticator interface {
     Authenticate(ctx context.Context, token string) (*CustomerContext, error)
 }
```

提供两种实现：

- `DelegateCustomerAuthenticator`：调用宿主 `/api/v1/customer/auth/validate`（示例），验证后返回宿主 claims；失败时将宿主错误映射到 `contracts.ResponseUnauthorized`。
- `LocalCustomerAuthenticator`：
  - 解析本地 JWT（与 admin JWT 共用 `middleware.JWTAuth`，但 issuer/audience 不同）。
  - 或者走邮箱/手机号 + 密码登录 -> 生成自定义 token。
  - Token payload 至少包含 `tenant_uuid`、`customer_uuid`、`exp`，并可附加角色、渠道等字段。

### CustomerContext 与中间件

1. 在 `backend/internal/middleware` 下新增 `CustomerContext` 结构和 getter/setter。
2. Mini-app Router：

```go
miniAppGroup := r.engine.Group(prefix)
miniAppGroup.Use(middleware2.RequestTrace())
miniappapi.RegisterRoutes(miniAppGroup, r.deps)
```

在 `miniapp.RegisterRoutes` 内，为 `/mini-app` group 增加 `CustomerAuthMiddleware(authenticator)`：

```go
group := rg.Group("/mini-app",
    customerhttp.Authenticate(authenticator),
    httpmw.EnsureTenant(),
)
```

其中 `customerhttp.Authenticate` 负责：
- 从 Header（如 `Authorization: Bearer <token>` 或 `X-Customer-Token`）提取凭证。
- 调用 `CustomerAuthenticator.Authenticate`。
- 若请求显式携带 tenant，则校验 `request.tenant_uuid == customer.tenant_uuid`；否则把 `customer.tenant_uuid` 注入到请求上下文，供 `EnsureTenant()` 与后续 service/repo 使用。
- 将 `CustomerContext` 写入请求上下文，供 Handler/Service 使用。

### Skeleton 模式数据流

1. **注册/登录 API**：在 `skeleton/backend/go-gin/internal/transport/http/mini-app` 下提供 `/mini-app/auth/register`、`/mini-app/auth/login`，操作 `customer_accounts`（`CustomerAccount`）表。
2. **密码存储**：务必使用 `bcrypt`/`argon2`；不存明文密码。
3. **Token 颁发**：在登录成功后使用 `middleware.JWTAuth` 的 helper：

```go
token := jwt.NewWithClaims(...)
claims := CustomerClaims{TenantUUID, CustomerUUID, Roles, Exp}
```

Issuer 可以使用 `POWERX_SECURITY_JWT_ISSUER` 或单独的 `POWERX_CUSTOMER_JWT_ISSUER`。Mini-app 客户端保存 token，并在每次请求携带。

### Delegated 模式数据流

1. 客户端调用 PowerX 宿主的登录入口，获取宿主颁发的 customer token。
2. mini-app 中间件调用宿主校验端点（可由宿主在 header 中注入 STS context，或暴露 `/api/v1/customer/tokens:inspect`）。
3. 宿主返回的 payload 包含 `tenant_uuid` 和 `customer_uuid`，插件只需要做匹配与续传；无需落库。
4. 如需缓存，可使用 Redis 缓存校验结果，TTL 不超过 token `exp`。

## 步骤建议

1. **接口设计**：在 `specs/**/spec.yaml` 增补 `/mini-app/auth/login` 等端点，定义成功/失败响应（统一 envelope）。
2. **Service**：
   - `internal/services/customer/auth/local_service.go`：注册/登录/密钥生成。
   - `internal/services/customer/auth/delegate_service.go`：HTTP 客户端调用宿主。
3. **Repository**：`internal/entity/repository/customer`（若尚未生成）继承 `BaseRepository`，封装 `FindByCustomerID`, `CreateCustomer` 等，全部通过 `WithTenantTx`。
4. **Middleware**：`internal/transport/http/middleware/customer_auth.go` 新增中间件，将 `CustomerContext` 存储在 Gin context + request context 中。
5. **Handler 更新**：
   - mini-app handler 通过新的 helper `customerctx.FromGin(c)` 获取当前客户。
   - 错误统一用 `contracts.ResponseUnauthorized` 或 `ResponseBadRequest`。
6. **配置**：
   - `backend/etc/config.*.yaml` 新增 `customerAuth` 段落（`mode: delegate|local`, `delegateEndpoint`, `jwtIssuer`, `jwtSecret`）。
   - `plugin.yaml` 可声明 mini-app 接口需要的 scopes（如 `com.powerx.plugin.ecommerce:miniapp.product`）。

## 注意事项

- **安全**：Skeleton 模式下禁用弱密码；token 过期时间建议 ≤ 2 小时，并支持 refresh token 或 STS 交换接口。
- **审计**：在日志中记录 `tenant_uuid`、`customer_uuid`、`request_id`，必要时在 `observability` 域新增 customer 访问日志。
- **兼容**：若宿主/本地都可能存在，使用工厂模式基于配置决定加载哪个 authenticator；保持接口一致即可在两种模式间切换。
- **测试**：
  - 单测 `LocalCustomerAuthenticator`（密码正确/错误、租户 mismatch）。
  - 集成测试 mini-app handler，验证 `contracts.Response*` envelope。
  - 若启用 Redis 缓存，添加 TTL 相关测试。

## 操作指南（推荐）

### Skeleton（local）模式：注册 → 登录 → 调用受保护接口

1. 配置 `customer_auth`（示例见 `skeleton/backend/go-gin/etc/config.example.yaml`）：
   - `customer_auth.mode: local`
   - 开发态可复用 `context.hmac_secret`；生产态必须配置 `customer_auth.jwt_secret`
2. 调用注册接口（必须明确目标租户：优先使用 `X-Tenant-UUID`；若 header 不可用，可在 body 里传 `tenant_uuid`；两者同时存在时必须一致）：
   - `POST /api/v1/mini-app/auth/register`
3. 调用登录接口获取 token：
   - `POST /api/v1/mini-app/auth/login`
   - 多租户提示：当同一 `login`（邮箱/手机号）在多个租户存在且请求未指定 `tenant_uuid` 时，接口返回 `409` + `TENANT_SELECTION_REQUIRED`，并在 `error.details.tenants[]` 给出候选租户列表；客户端选择后再次登录即可。
4. 使用 token 调用受保护接口（示例：`GET /api/v1/mini-app/ping`）：
   - `Authorization: Bearer <token>` 或 `X-Customer-Token: <token>`
   - `X-Tenant-UUID` 头可省略（tenant 会从 token 注入）；若显式携带，则必须与 token 中的 `tenant_uuid` 一致。

更完整的命令与参数请参考 `specs/010-auth-customer/quickstart.md`。

### 示例：给移动端暴露 mini-app Template API

在本仓库 Skeleton 后端中，已新增只读的 Template API（仅返回 `published + approved` 模板），用于移动端/小程序等 C 端读取：

- `GET /api/v1/mini-app/templates`（分页：`page`/`page_size`，可选 `q` 搜索）
- `GET /api/v1/mini-app/templates/:id`

上述接口位于 `/mini-app/*` 保护域内：必须携带 `Authorization: Bearer <customer_token>`（或 `X-Customer-Token`）。`X-Tenant-UUID` 可省略（tenant 会从 token 注入）；若显式携带，则必须与 token tenant 一致。

### Delegated（delegate）模式：宿主校验 → 插件只做转发与租户匹配

1. 配置 `customer_auth`：
   - `customer_auth.mode: delegate`
   - `customer_auth.delegate_endpoint: <PowerX 校验端点 URL>`
   - 可选：`customer_auth.cache_ttl_seconds`（启用内存缓存，TTL 不超过宿主返回的 token 过期时间）
2. mini-app 客户端从宿主获取 customer token 后，请求插件 `/mini-app/*`：
   - `X-Tenant-UUID: <tenant_uuid>`
   - `Authorization: Bearer <host_customer_token>`
3. 插件行为：
   - 调用 `delegate_endpoint` 校验 token（POST JSON + 透传 `Authorization` 与 `X-Tenant-UUID`）
   - 校验响应中的 `tenant_uuid` 与请求一致，否则返回 `TENANT_MISMATCH`
   - 不创建/更新任何 customer 表记录（customer 表仅 Skeleton 使用）

### 可观测性（Metrics & Logs）

- **日志事件**：`customer.auth.validation` / `customer.auth.login` / `customer.auth.register`（字段含 `tenant_uuid`、`customer_uuid`、`request_id`、`latency_ms`）
- **指标**（Admin Runtime Metrics 输出）：
  - `powerx_customer_auth_validation_total{plugin_id,mode,result}`
  - `powerx_customer_auth_validation_latency_ms_sum{...}` / `powerx_customer_auth_validation_latency_ms_count{...}`
  - `powerx_customer_auth_login_total{plugin_id,result}`
  - `powerx_customer_auth_login_latency_ms_sum{...}` / `powerx_customer_auth_login_latency_ms_count{...}`

通过以上策略，插件即可在不同部署形态下同时满足“必须带租户上下文”和“Customer 身份校验”两个目标，而且能与现有 Constitution / Ruleset（统一响应、模块隔离、可观测）保持一致。
