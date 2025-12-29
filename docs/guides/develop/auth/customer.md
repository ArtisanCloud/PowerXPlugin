# Customer 鉴权策略（Mini-App / 2C 场景）

> 适用范围：PowerX 插件在 C 端（mini-app）暴露产品/下单等接口，需要识别租户 + Customer 身份。脚手架已内置 `customer.Customer` 模型，可用于 Skeleton 模式下的本地账号体系；部署在宿主（PowerX）时则复用宿主 CRM 的 Customer 校验能力。

## 模式划分

| 模式 | 触发条件 | Customer 记录来源 | Token 签发者 | 适配说明 |
| --- | --- | --- | --- | --- |
| **Delegated（宿主）** | `POWERX_PROXY=1` 或明示 `POWERX_CUSTOMER_DELEGATE=true` | PowerX 底座 CRM / IAM | 宿主安全域 | 插件不落地 Credential，仅校验宿主颁发的 customer token，并将 claims 转写到 `TenantContext`。 |
| **Skeleton（Standalone）** | 无宿主反代；`POWERX_CUSTOMER_DELEGATE` 关闭 | 插件维护的 `customer.Customer` 表 | 插件自身的 JWT/STS 服务 | 复用已有 Go JWT 中间件，新增 Customer login API + token 颁发逻辑，数据按 `tenant_uuid` 隔离。 |

## 目标能力

1. **租户隔离**：`httpmw.EnsureTenant()` 继续作为必选前置，Customer auth 成功后要验证 `customer.TenantUUID == request.TenantUUID`。
2. **统一上下文**：定义 `CustomerContext`（`customer_uuid/customer_id/roles/attributes`），通过 `authx.SetTenantContext` 或新的 `SetCustomerContext` 挂入 `gin.Context`，Service 层可通过 `customer.ContextFrom(ctx)` 获取。
3. **统一返回**：所有 mini-app handler 使用 `contracts.Response*` 封装（参考 `backend/internal/transport/http/miniapp/product/handler.go`），以满足 `.specify/memory/rulesets/crud/api_rest.yaml` envelope 约定。

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
    httpmw.EnsureTenant(),
    customerhttp.Authenticate(authenticator),
)
```

其中 `customerhttp.Authenticate` 负责：
- 从 Header（如 `Authorization: Bearer <token>` 或 `X-Customer-Token`）提取凭证。
- 调用 `CustomerAuthenticator.Authenticate`。
- 校验 `ctx.TenantUUID == customer.TenantUUID`，否则返回 `contracts.ResponseUnauthorized`。
- 将 `CustomerContext` 写入请求上下文，供 Handler/Service 使用。

### Skeleton 模式数据流

1. **注册/登录 API**：在 `backend/internal/transport/http/miniapp/customer` 下提供 `/mini-app/auth/register`、`/mini-app/auth/login`，操作 `customer.Customer` 表。
2. **密码存储**：新增 `customer_account` 表或在 `customer.Customer` 中加入 `PasswordHash`、`Salt` 字段；务必使用 `bcrypt`/`argon2`。
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

通过以上策略，插件即可在不同部署形态下同时满足“必须带租户上下文”和“Customer 身份校验”两个目标，而且能与现有 Constitution / Ruleset（统一响应、模块隔离、可观测）保持一致。
