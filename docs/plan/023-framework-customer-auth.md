# PowerXPlugin Framework Customer Identity/Auth 统一规划

本文定义 PowerXPlugin framework 中面向 C 端外部用户的通用 Customer Identity/Auth 能力。该能力解决插件如何识别、校验、获取一个外部 customer 身份，并承载 PowerX Core customer 主账号的基础展示属性；它不承载 SCRM 或行业业务模型。

## 1. 背景

插件在 mini-app、mobile app、public portal 等 C 端入口中，需要识别当前访问者是谁、属于哪个租户、是否具备当前租户下的访问资格。当前 skeleton 已经存在一套 customer auth 实现，包括 CustomerContext、customer token middleware、local/delegate authenticator 与测试用例。这些能力本质上属于 framework 横切能力，不应长期留在单个插件或 skeleton 内重复维护。

framework 应沉淀的是 Customer Identity/Auth 的公共契约与工具，以及 `display_name`、`nickname`、`given_name`、`family_name`、`avatar_url`、`locale`、`timezone` 这类 PowerX Core customer 基础展示属性，而不是“客户标签、跟进、会员权益、学员、患者、粉丝”等行业实体。

生产数据权威源在 PowerX Core。PowerXPlugin framework 不定义生产 customer 表结构，只定义运行时 contract、middleware、adapter 和 local dev mirror 约束。插件 local 模式的 customer 表只是开发调试镜像，必须对齐 PowerX Core customer schema；如果 Core 缺表，应先补齐 Core，再同步 framework skeleton/scaffold 和插件 local mirror。

## 2. 目标

1. 在 framework 中提供统一 customer 身份上下文，使业务插件只依赖稳定的 CustomerContext。
2. 提供通用 customer 鉴权中间件，统一解析 token、校验身份、注入上下文并阻断未登录请求。
3. 提供 token 校验、membership 解析、入口解析与 Core 委托认证的接口契约。
4. 支持 local、core、wechat、delegate、mock 等不同身份来源，但对插件业务暴露一致上下文。
5. 为插件测试提供标准 helper，避免每个插件重复实现 token、context、middleware 测试工具。
6. 与 member IAM 保持语义隔离：member IAM 管后台操作者，customer auth 管 C 端外部用户。
7. 明确 PowerX Core 是 production customer identity/auth 的权威源，framework 只消费和适配。

## 3. 非目标

1. 不提供行业 customer 模型。
2. 不定义家长、球员、学员、会员、患者、粉丝等业务身份。
3. 不管理业务档案、成长等级、会员权益、训练目标、报告等插件领域数据。
4. 不复用 member IAM 的后台成员权限模型来表达 C 端用户。
5. 不要求 framework 自己持久化生产 customer。framework 只定义接口与默认实现边界，真实生产数据来自 PowerX Core 或平台级 delegated identity source；local dev store 和测试 mock 只能用于开发调试。

## 3.1 数据权威源

PowerX Core 负责生产 customer 数据模型：

1. `customer_accounts`：C 端 customer 主账号。
2. `customer_auth_identities`：密码、手机号、邮箱、微信、Apple、第三方登录身份。
3. `customer_tenant_memberships`：customer 与 tenant 的访问关系。
4. `mini_app_entries`：shared app 场景下的合法租户入口。
5. `customer_sessions`：refresh/session 生命周期。
6. `customer_login_events`：注册、登录、校验、刷新、登出审计。

Framework 与 Core 的映射：

| Framework runtime | Core 权威来源 |
| --- | --- |
| `CustomerContext.customer_uuid` | `customer_accounts.uuid` |
| `CustomerContext.profile.display_name` | `customer_accounts.display_name` |
| `CustomerContext.profile.nickname` | `customer_accounts.nickname` |
| `CustomerContext.profile.given_name` | `customer_accounts.given_name` |
| `CustomerContext.profile.family_name` | `customer_accounts.family_name` |
| `CustomerContext.profile.avatar_url` | `customer_accounts.avatar_url` |
| `CustomerContext.profile.locale/timezone` | `customer_accounts.locale/timezone` |
| `CustomerContext.tenant_uuid` | `customer_tenant_memberships.tenant_uuid` 或 `mini_app_entries.tenant_uuid` |
| `CustomerContext.membership_uuid` | `customer_tenant_memberships.uuid` |
| `CustomerContext.roles/scopes` | `customer_tenant_memberships.roles/scopes` |
| `CustomerMembership` | `customer_tenant_memberships` |
| `BootstrapContext` | `mini_app_entries` |
| `CustomerAuthResult` | `customer_accounts` + `customer_auth_identities` + `customer_sessions` |

插件 local 模式可以落本地表，但必须遵循：

1. 仅用于 `POWERX_PROVIDER_MODE=local` 的开发调试。
2. 表结构、字段命名、状态枚举、membership 语义与 PowerX Core 保持兼容。
3. Core schema 变更后，PowerXPlugin skeleton/scaffold 和业务插件 local mirror 同步变更。
4. 可以镜像 PowerX Core customer 基础展示字段，但不得把标签、跟进、归属销售、家长、球员、学员、患者、粉丝等行业概念写入 framework customer 表。

## 4. 与 Member IAM 的区别

Member IAM 面向 B 端/后台操作者：

1. 典型入口为 `/admin/*`、`/tenant/*`、后台管理 API。
2. 关注 member、role、permission、department、admin session。
3. 授权重点是后台 RBAC 与管理权限。

Customer Auth 面向 C 端外部用户：

1. 典型入口为 `/mini-app/*`、`/mobile/*`、`/customer/*`、public portal API。
2. 关注 customer identity、customer token、tenant membership、scope、入口解析。
3. 授权重点是是否登录、是否属于当前 tenant、token 是否可用于当前入口与租户。

两者可以共享类似的 context/middleware/test helper 形态，但不能混用语义。

## 5. Framework 包边界

建议新增 framework Go 包：

```text
github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw
```

该包负责：

1. 通用类型定义。
2. 鉴权中间件。
3. context getter/setter。
4. token validator 接口。
5. membership resolver 接口。
6. mini-app/bootstrap entry client 接口。
7. customer auth client 接口。
8. 测试 helper。

插件侧只依赖 `customerfw`，不直接依赖 skeleton 内部 customer service 或某个具体 PowerX Core SDK 实现。

## 6. 核心类型

### 6.1 CustomerContext

`CustomerContext` 是业务插件唯一应直接依赖的 C 端运行时身份结构：

```go
type CustomerContext struct {
    TenantUUID     string
    CustomerUUID   string
    MembershipUUID string
    Profile        CustomerAttributes
    Roles          []string
    Scopes         []string
    Source         string // local, core, wechat, delegate, mock
}

type CustomerAttributes struct {
    DisplayName string
    Nickname    string
    GivenName   string
    FamilyName  string
    AvatarURL   string
    Locale      string
    Timezone    string
}
```

字段说明：

1. `TenantUUID`：当前请求解析出的租户。
2. `CustomerUUID`：C 端外部用户身份。
3. `MembershipUUID`：customer 在当前 tenant 下的 membership。
4. `Profile`：PowerX Core customer 主账号基础展示属性，只用于通用展示与识别。
5. `Roles`：C 端身份角色，只用于 customer 侧语义。
6. `Scopes`：C 端 token 或授权范围。
7. `Source`：身份来源，用于审计、观测与差异化策略。

可选扩展：

```go
type CustomerContext struct {
    TenantUUID     string
    CustomerUUID   string
    MembershipUUID string
    Profile        CustomerAttributes
    Roles          []string
    Scopes         []string
    Source         string
    Attributes     map[string]any
    RawClaims      map[string]any
    Authenticated  bool
}
```

`Profile` 只能承载 `display_name`、`nickname`、`given_name`、`family_name`、`avatar_url`、`locale`、`timezone`。`Attributes` 与 `RawClaims` 只能承载通用 claim，不应放入行业业务字段。

### 6.2 CustomerMembership

```go
type CustomerMembership struct {
    TenantUUID     string
    CustomerUUID   string
    MembershipUUID string
    Status         string
    Roles          []string
    Scopes         []string
}
```

`Status` 建议保留通用值：

```text
active | suspended | disabled | pending
```

## 7. 中间件

framework 提供 customer auth middleware：

```go
func Authenticate(validator CustomerTokenValidator, opts ...AuthOption) gin.HandlerFunc
```

职责：

1. 从 `Authorization: Bearer <token>` 或 `X-Customer-Token` 解析 token。
2. 从请求上下文、header、query 或 bootstrap 结果中解析 `tenant_uuid`。
3. 调用 `CustomerTokenValidator` 校验 token。
4. 校验 token tenant 与请求 tenant 是否一致。
5. 注入 `CustomerContext` 到 `gin.Context` 与 `request.Context`。
6. 必要时把 `tenant_uuid` 写入请求上下文，供 tenant middleware 继续使用。
7. 对未登录、无效 token、tenant mismatch、上游不可用返回统一错误。

推荐路由形态：

```go
protected := base.Group(
    "",
    customerfw.Authenticate(customerValidator),
    customerfw.RequireMembership(membershipResolver),
    tenantfw.EnsureTenant(),
)
```

middleware 顺序原则：

1. `Authenticate` 先解析 customer token。
2. `RequireMembership` 确认 customer 与 tenant 的关系。
3. `tenantfw.EnsureTenant` 复用已解析 tenant 做租户隔离。

## 8. Context Helper

framework 提供标准 getter/setter：

```go
func WithContext(ctx context.Context, cc *CustomerContext) context.Context
func ContextFrom(ctx context.Context) (*CustomerContext, bool)
func MustContextFrom(ctx context.Context) *CustomerContext

func SetGinContext(c *gin.Context, cc *CustomerContext)
func ContextFromGin(c *gin.Context) (*CustomerContext, bool)
func MustContextFromGin(c *gin.Context) *CustomerContext
```

handler 示例：

```go
cc := customerfw.MustContextFromGin(c)
tenantUUID := cc.TenantUUID
customerUUID := cc.CustomerUUID
```

service 层示例：

```go
cc, ok := customerfw.ContextFrom(ctx)
if !ok {
    return ErrCustomerContextMissing
}
```

## 9. Token Validator

framework 定义通用 token 校验接口：

```go
type CustomerTokenValidator interface {
    Validate(ctx context.Context, token string, tenantUUID string) (*CustomerContext, error)
}
```

实现来源：

1. `CoreCustomerTokenValidator`：委托 PowerX Core 校验。
2. `LocalCustomerTokenValidator`：local dev 或 standalone 使用。
3. `WechatCustomerTokenValidator`：接入微信等 C 端登录渠道。
4. `MockCustomerTokenValidator`：测试使用。

validator 不应返回行业业务模型，只返回 `CustomerContext`。

## 10. Membership Resolver

shared app 与多租户 customer 场景需要 framework 提供 membership 解析接口：

```go
type CustomerMembershipResolver interface {
    Resolve(ctx context.Context, customerUUID string, tenantUUID string) (*CustomerMembership, error)
    List(ctx context.Context, customerUUID string) ([]CustomerMembership, error)
}
```

用途：

1. 确认 customer 是否属于当前 tenant。
2. 支持 customer 登录后选择机构/租户。
3. 防止 token 跨租户使用。
4. 为 `CustomerContext.MembershipUUID`、`Roles`、`Scopes` 补全当前 tenant 下的授权信息。

framework 可提供中间件：

```go
func RequireMembership(resolver CustomerMembershipResolver, opts ...MembershipOption) gin.HandlerFunc
```

`RequireMembership` 应只判断通用 membership，不判断插件业务身份。

## 11. MiniApp Bootstrap Client

C 端入口常通过 scene、invite_code、org_code 等参数进入应用。framework 至少需要定义统一 client/interface，让插件不自行解析所有入口规则。

```go
type MiniAppBootstrapClient interface {
    ResolveEntry(ctx context.Context, input BootstrapInput) (*BootstrapContext, error)
}

type BootstrapInput struct {
    Scene      string
    InviteCode string
    OrgCode    string
    TenantHint string
    Channel    string
}

type BootstrapContext struct {
    TenantUUID string
    OrgUUID    string
    EntryType  string
    Campaign   string
    Metadata   map[string]any
}
```

职责：

1. 将入口参数解析为 tenant context。
2. 支持 shared app 的多租户入口。
3. 允许 PowerX Core 统一维护入口规则。
4. 避免插件重复实现 invite、scene、org_code 解析。

## 12. Customer Auth Client

framework 定义插件委托 PowerX Core 的 customer auth client：

```go
type CustomerAuthClient interface {
    Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
    Login(ctx context.Context, input LoginInput) (*AuthResult, error)
    Validate(ctx context.Context, token string) (*CustomerContext, error)
}
```

通用输入输出：

```go
type RegisterInput struct {
    TenantUUID string
    Identifier string
    Password   string
    Channel    string
    Profile    CustomerAttributes
    Attributes map[string]any
}

type LoginInput struct {
    TenantUUID string
    Identifier string
    Password   string
    Channel    string
    Code       string
    Nickname   string
    AvatarURL  string
    Profile    CustomerAttributes
    Attributes map[string]any
}

type AuthResult struct {
    AccessToken  string
    RefreshToken string
    ExpiresIn    int64
    Customer     *CustomerContext
}
```

`RegisterInput.Profile` 与 `LoginInput.Profile` 只用于 PowerX Core customer 基础展示属性。`RegisterInput.Attributes` 与 `LoginInput.Attributes` 只用于通用登录通道扩展参数，不承载插件业务档案。

## 13. 错误模型

framework 应定义通用错误：

```text
CUSTOMER_TOKEN_MISSING
CUSTOMER_TOKEN_INVALID
CUSTOMER_UNAUTHENTICATED
CUSTOMER_TENANT_MISMATCH
CUSTOMER_MEMBERSHIP_REQUIRED
CUSTOMER_MEMBERSHIP_DISABLED
CUSTOMER_AUTH_DELEGATE_UNAVAILABLE
CUSTOMER_BOOTSTRAP_FAILED
CUSTOMER_CONTEXT_MISSING
```

错误映射建议：

1. token 缺失或无效：`401 Unauthorized`。
2. customer 未登录：`401 Unauthorized`。
3. tenant mismatch：`403 Forbidden`。
4. membership 不存在或禁用：`403 Forbidden`。
5. delegate/core 不可用：`503 Service Unavailable`。
6. bootstrap 参数无效：`400 Bad Request` 或 `404 Not Found`，按入口语义决定。

## 14. 配置建议

framework 层保留通用配置结构：

```yaml
customer_auth:
  mode: delegate # local | core | wechat | delegate | mock
  delegate_endpoint: ""
  bootstrap_endpoint: ""
  jwt_issuer: ""
  jwt_audience: ""
  jwt_secret: ""
  cache_ttl_seconds: 60
  request_timeout: 3s
```

配置原则：

1. production 默认不应启用弱 local secret。
2. delegate/core validator 必须设置超时。
3. token 校验缓存 TTL 不得超过 token 自身有效期。
4. 启动日志应输出 customer auth mode，但不得输出 secret/token。

## 15. 测试工具

framework 提供测试 helper：

```go
func TestToken(input TestTokenInput) string
func WithCustomerContext(ctx context.Context, cc *CustomerContext) context.Context
func NewMockCustomerValidator(fn func(context.Context, string, string) (*CustomerContext, error)) CustomerTokenValidator
func NewMockMembershipResolver(fn func(context.Context, string, string) (*CustomerMembership, error)) CustomerMembershipResolver
```

典型用途：

1. middleware 单元测试。
2. handler 测试中注入 customer context。
3. service 测试中绕过真实 token。
4. 模拟 tenant mismatch、membership disabled、delegate unavailable。

## 16. 迁移路径

### 16.1 当前状态

skeleton 已有插件内 customer auth 实现：

1. `CustomerContext`。
2. customer context getter/setter。
3. customer auth middleware。
4. local/delegate authenticator。
5. mini-app 路由接入。
6. local/delegate 相关测试。

### 16.2 目标状态

将通用能力从 skeleton/internal 抽到 framework：

```text
skeleton/internal/domain/customer
skeleton/internal/middleware/customer
skeleton/internal/middleware/customer_auth.go
skeleton/internal/services/customer/*authenticator*
        ↓
framework/backend/go/runtime/customerfw
```

skeleton 与业务插件只保留：

1. 具体配置装配。
2. 具体 Core client 注入。
3. 插件业务模型与业务 handler。
4. 必要的 local dev adapter。

### 16.3 建议步骤

1. 在 framework 新增 `runtime/customerfw` 包，先迁移类型、context helper、错误定义与测试 helper。
2. 抽象 `CustomerTokenValidator`，将 skeleton authenticator 适配成 validator。
3. 迁移 customer auth middleware，保持现有 header、tenant mismatch、context 注入行为兼容。
4. 新增 `CustomerMembershipResolver` 与 `RequireMembership`，先提供 no-op/mock/local adapter。
5. 新增 `MiniAppBootstrapClient` 与 `CustomerAuthClient` 接口，等待 PowerX Core 真实 API 接线。
6. 更新 skeleton 模板改为依赖 framework customerfw。
7. 保留旧路径短期兼容或通过 adapter 过渡。

## 17. 最小落地版本

1. `CustomerContext` 与 context helper。
2. `CustomerTokenValidator` 接口。
3. `Authenticate` middleware。
4. `CustomerMembershipResolver` 接口。
5. `RequireMembership` middleware。
6. `CustomerAuthClient` 与 `MiniAppBootstrapClient` 接口定义。
7. mock/test helper。
8. skeleton customer auth 改为通过 framework 接口装配。

## 18. 待确认项

1. PowerX Core customer auth 的正式 API、SDK、错误码与鉴权方式。
2. Core 是否统一维护 customer membership，还是允许插件 local adapter 提供。
3. bootstrap entry 的输入字段与返回字段是否已有底座规范。
4. customer token 是否全局有效，还是必须绑定 tenant。
5. `Roles` 与 `Scopes` 的命名是否需要与 capability/permission 系统对齐。
6. local dev 模式是否继续支持注册/登录，或只提供 mock validator。
7. framework 是否需要提供非 Gin adapter，例如 FastAPI/Node runtime 的同名契约。

## 19. 实现状态

截至 2026-06-24，framework customer identity/auth 主体迁移已完成：

1. `framework/backend/go/runtime/customerfw` 已提供 context、token validator、auth middleware、tenant resolver、membership middleware/cache、bootstrap contract、delegated auth client、source policy 与测试 helper。
2. skeleton mini-app 已通过 adapter 接入 `customerfw.Authenticate`、`customerfw.RequireMembership`、`/mini-app/bootstrap/resolve`、`/mini-app/auth/validate`。
3. scaffold 与 CLI Go-Gin 模板已同步 customer auth adapter、handler、routes 与 config source policy 字段。
4. 对外文档已整理到 `docs/guides/develop/auth/customer.md` 和 `docs/contracts/customer-auth.openapi.yaml`。
5. 静态边界检查已加入 `scripts/contracts/validate-customer-auth-boundary.sh`，并挂入 `npm test`。
