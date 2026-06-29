# Feature Specification: Framework Customer Identity/Auth

**Feature Branch**: `023-framework-customer-auth`  
**Created**: 2026-06-24  
**Status**: Draft  
**Input**: 基于 `docs/plan/023-framework-customer-auth.md`，为 PowerXPlugin Framework 定义面向 C 端外部用户的通用 Customer Identity/Auth 能力。该能力只解决插件如何识别、校验、获取 customer 身份和租户关系，不承载任何行业 customer 模型。

## Clarifications

### Session 2026-06-24

- Q: 是否并入 `018-framework-iam-unification`？ → A: 不并入。018 面向后台 member IAM；本 feature 面向 C 端 customer identity/auth，二者保持语义隔离。
- Q: framework 是否提供行业 customer 模型？ → A: 不提供。家长、球员、学员、会员、患者、粉丝等业务身份留在插件侧。
- Q: customer auth 是否必须由 framework 自己持久化 customer？ → A: 不必须。framework 定义通用契约、上下文、鉴权、membership 与测试工具，真实 customer 数据可来自 PowerX Core、local dev、第三方登录或 mock。
- Q: 现有 skeleton customer auth 如何处理？ → A: 当前实现可继续支撑 MVP，但通用部分需要迁移到 framework，skeleton 和业务插件只保留装配与业务模型。
- Q: 生产环境 Customer 身份权威源是谁？ → A: PowerX Core 或平台身份源是生产权威源；插件只能委托校验、登录、membership 与 bootstrap，不在生产环境自建权威身份源。
- Q: Customer token 是否必须绑定单一 tenant？ → A: 不必须。token 可表达全局 customer 身份；访问 tenant-scoped 资源时必须解析或指定当前 tenant，并强制校验 membership。
- Q: Membership 校验是否允许缓存？ → A: 平台 membership 为权威；framework 可短 TTL 缓存，但缓存不得超过 token 有效期，并且必须支持显式失效或状态变化后的拒绝策略。
- Q: 同时携带多个 customer token 时如何处理？ → A: 多个 token 可同时存在，但必须校验为同一 customer 与 tenant 上下文；不一致时拒绝请求。
- Q: 生产环境是否允许 local/mock identity source？ → A: 生产环境默认禁止 local/mock；仅允许显式 break-glass 配置启动，并且必须记录审计与诊断标记。

## User Scenarios & Testing

### User Story 1 - 插件统一获取 C 端身份上下文 (Priority: P1)

作为插件业务开发者，我希望在 C 端请求处理过程中通过统一 customer 身份上下文获取当前 customer、tenant 和 membership 信息，而不是在每个 handler 或 service 中重复解析 header、token 和租户参数。

**Why this priority**: 这是所有 C 端受保护接口的基础。没有统一上下文，插件会重复实现 token 解析、租户校验和身份传递，导致多租户隔离和鉴权语义漂移。

**Independent Test**: 构造一个带有效 customer token 的 C 端请求，经过通用 customer 鉴权后，后续 handler 和 service 均能读取同一份 customer context，且不需要重新读取 token。

**Acceptance Scenarios**:

1. **Given** 请求携带有效 customer token 和匹配的租户信息，**When** 请求进入受保护 C 端接口，**Then** 系统注入包含 customer、tenant、membership、roles、scopes 和身份来源的 customer context。
2. **Given** 请求未携带 customer token，**When** 请求进入受保护 C 端接口，**Then** 系统拒绝请求并返回稳定的未登录错误。
3. **Given** 业务服务只接收请求上下文，**When** 服务需要当前 C 端身份，**Then** 服务可以从上下文读取 customer context，而不是依赖传递 header 或 token。

---

### User Story 2 - 阻止 customer token 跨租户使用 (Priority: P1)

作为平台安全负责人，我希望 C 端 customer 访问租户资源时必须具备明确当前租户并通过 membership 校验，避免外部用户凭全局身份或其他租户上下文访问不属于自己的租户数据。

**Why this priority**: customer auth 的核心安全边界是 customer 与 tenant 的关系。跨租户 token 使用会直接造成数据越权。

**Independent Test**: 使用全局 customer token 访问租户资源但不提供可解析租户时必须被拒绝；使用同一 token 访问有 active membership 的租户时通过，访问无 membership 的租户时拒绝。

**Acceptance Scenarios**:

1. **Given** customer token 只表达全局 customer 身份，**When** 请求访问 tenant-scoped 资源但没有当前租户，**Then** 系统拒绝请求并要求提供或解析租户上下文。
2. **Given** 请求没有显式租户但 token 或入口中可唯一解析租户，**When** token 有效且 membership active，**Then** 系统将解析出的租户作为当前请求租户继续处理。
3. **Given** customer 有多个租户 membership，**When** 请求没有明确租户且无法唯一确定租户，**Then** 系统要求调用方选择或提供租户上下文。

---

### User Story 3 - 校验 customer 与 tenant membership (Priority: P1)

作为插件业务开发者，我希望 framework 能统一确认当前 customer 是否属于当前 tenant，并提供 membership 结果，避免每个插件重复实现 customer-tenant 关系校验。

**Why this priority**: shared app 和多机构 C 端场景下，登录成功不代表可访问当前 tenant。membership 校验必须成为通用能力。

**Independent Test**: 构造 active、disabled、不存在三种 membership，受保护接口分别允许访问、拒绝访问、拒绝访问并返回稳定错误。

**Acceptance Scenarios**:

1. **Given** customer 在当前 tenant 下存在 active membership，**When** 请求进入需要 membership 的接口，**Then** 系统允许继续并补全 membership 信息。
2. **Given** customer 在当前 tenant 下没有 membership，**When** 请求进入需要 membership 的接口，**Then** 系统拒绝请求并返回 membership required。
3. **Given** customer membership 已被禁用或挂起，**When** 请求进入受保护接口，**Then** 系统拒绝请求并返回 membership disabled。

---

### User Story 4 - 统一 C 端入口解析到租户上下文 (Priority: P2)

作为 C 端应用开发者，我希望 framework 提供统一入口解析能力，将 scene、invite code、org code 或 tenant hint 等入口信息解析为租户上下文，而不是让每个插件维护入口规则。

**Why this priority**: shared app 的入口规则通常由平台或 Core 统一维护。插件自行解析会导致规则分叉、入口失效和租户绑定错误。

**Independent Test**: 使用有效入口参数请求 bootstrap 解析，系统返回明确 tenant context；使用过期或无效入口参数时返回稳定失败原因。

**Acceptance Scenarios**:

1. **Given** C 端入口包含有效 invite code，**When** 应用解析入口，**Then** 系统返回对应 tenant、组织和入口类型。
2. **Given** C 端入口包含无效或过期 scene，**When** 应用解析入口，**Then** 系统返回 bootstrap failed，并且不创建 customer 登录状态。
3. **Given** 请求后续携带 customer token，**When** token tenant 与入口解析 tenant 不一致，**Then** 系统拒绝请求并返回 tenant mismatch。

---

### User Story 5 - 委托 Core 完成 customer 注册、登录和校验 (Priority: P2)

作为插件开发者，我希望插件可以通过统一 customer auth 契约委托 PowerX Core 或其他身份源完成注册、登录和 token 校验，避免插件维护通用 customer 鉴权实现。

**Why this priority**: customer identity 是平台级能力。插件如果各自实现注册、登录和 token 校验，会重复造轮子并增加安全风险。

**Independent Test**: 在委托模式下，注册、登录、token 校验均走统一身份源；身份源不可用时返回稳定服务不可用错误，且插件不得静默降级到本地匿名访问。

**Acceptance Scenarios**:

1. **Given** 委托身份源可用，**When** C 端用户登录成功，**Then** 插件获得标准 auth result 和 customer context。
2. **Given** 委托身份源返回无效凭证，**When** 用户登录或 token 校验，**Then** 系统返回稳定认证失败错误。
3. **Given** 委托身份源不可用，**When** 受保护接口需要校验 customer token，**Then** 系统返回服务不可用错误，并保留可排查的 trace 信息。

---

### User Story 6 - 提供标准测试工具 (Priority: P3)

作为插件测试编写者，我希望 framework 提供 customer token、customer context、mock validator 和 mock membership resolver 等测试工具，避免每个插件重复构造鉴权测试基础设施。

**Why this priority**: customer auth 是横切能力。标准测试工具能保证插件测试覆盖一致的成功、未登录、tenant mismatch、membership disabled 和上游不可用场景。

**Independent Test**: 使用 framework 测试工具，无需真实身份源即可覆盖 handler、service 和 middleware 的 customer auth 场景。

**Acceptance Scenarios**:

1. **Given** 测试需要模拟已登录 customer，**When** 使用测试 helper 注入 customer context，**Then** handler 和 service 能读取同一份身份上下文。
2. **Given** 测试需要模拟 tenant mismatch，**When** mock validator 返回不同租户，**Then** 受保护接口返回 tenant mismatch。
3. **Given** 测试需要模拟 membership disabled，**When** mock resolver 返回禁用 membership，**Then** 受保护接口返回 membership disabled。

### Edge Cases

- 请求同时携带多个 customer token，且 token 指向不同 customer、tenant 或 membership。
- token 只表达全局 customer 身份，请求也没有入口或租户提示。
- customer token 有效但 membership 已过期、禁用或被删除。
- customer 有多个 tenant membership，但请求没有明确 tenant。
- 入口解析得到的 tenant 与 token 中 tenant 不一致。
- 委托身份源超时、返回 5xx、返回不可解析响应或返回过期 token。
- token 或 membership 校验结果被缓存，但 token 已过期或 membership 状态已变化。
- 请求缺少 trace 信息，错误响应和审计仍需可排查。
- 插件试图把行业业务字段写入通用 customer context。
- 插件误用 member IAM 的后台角色或权限来判断 customer 访问权。

## Requirements

### Functional Requirements

- **FR-001**: Framework MUST provide a common customer identity context for C-end requests, containing at minimum customer identifier, tenant identifier, membership identifier, roles, scopes, and identity source.
- **FR-002**: Framework MUST keep customer identity semantics separate from member IAM semantics; customer contexts MUST NOT be treated as后台 member contexts.
- **FR-003**: Framework MUST provide a standard protected-request flow that extracts customer tokens from supported request credentials and rejects missing tokens for protected C-end endpoints.
- **FR-003a**: When multiple supported customer token credentials are present, Framework MUST verify that they resolve to the same customer and tenant context; conflicts MUST be rejected before business logic executes.
- **FR-004**: Framework MUST validate customer tokens through a pluggable identity source and normalize successful validation into the common customer context.
- **FR-005**: Framework MUST reject invalid, expired, malformed, or unauthenticated customer tokens with a stable unauthenticated error.
- **FR-006**: Framework MUST compare the token-resolved tenant, request-resolved tenant, and bootstrap-resolved tenant whenever more than one is present, and MUST reject mismatches.
- **FR-007**: Framework MUST support globally scoped customer tokens, but tenant-scoped protected resources MUST have a current tenant resolved from token, request, or bootstrap before business logic executes.
- **FR-008**: Framework MUST support resolving customer membership for the current customer and tenant before protected business logic executes.
- **FR-009**: Framework MUST reject protected requests when membership is missing, disabled, suspended, or otherwise not active.
- **FR-009a**: Framework MUST reject tenant-scoped protected requests when a current tenant cannot be resolved from token, request, or bootstrap context.
- **FR-009b**: Framework MAY cache platform membership decisions for a short duration, but cache lifetime MUST NOT exceed token validity and MUST support explicit invalidation or conservative rejection after membership status changes.
- **FR-010**: Framework MUST expose the resolved customer context to both request handlers and downstream services without requiring token re-parsing.
- **FR-011**: Framework MUST provide a common entry/bootstrap resolution capability that maps C-end entry hints such as scene, invite code, organization code, or tenant hint to tenant context.
- **FR-012**: Framework MUST reject requests when bootstrap-resolved tenant and token-resolved tenant conflict.
- **FR-013**: Framework MUST provide a common customer auth contract for registration, login, and token validation through delegated or platform-managed identity sources.
- **FR-014**: Framework MUST support local development and test identity sources without requiring production customer persistence in the plugin.
- **FR-014a**: Production deployments MUST reject local or mock identity sources by default; any break-glass exception MUST be explicit, auditable, and visible in diagnostics.
- **FR-015**: Framework MUST define stable customer auth errors covering token missing, token invalid, unauthenticated, tenant mismatch, tenant required, membership required, membership disabled, delegate unavailable, bootstrap failed, context missing, and identity source forbidden.
- **FR-016**: Framework MUST map customer auth failures to consistent user-visible outcomes and machine-readable error codes.
- **FR-017**: Framework MUST emit audit and diagnostic information for customer auth decisions, including tenant, customer, source, result, latency category, and trace identifier when available.
- **FR-018**: Framework MUST avoid logging secrets, raw tokens, passwords, or sensitive customer credentials.
- **FR-019**: Framework MUST provide test helpers for creating test tokens, injecting customer context, mocking token validation, and mocking membership resolution.
- **FR-020**: Skeleton customer auth MUST be migrated to consume the framework customer auth contracts for shared behavior, while preserving existing protected C-end request behavior during transition.
- **FR-021**: Framework MUST NOT define or persist industry-specific customer models such as learner, member, patient, fan, player, guardian, entitlement, growth profile, or business report.
- **FR-022**: Framework MUST support multiple identity sources, including platform delegated, local development, third-party login channel, and mock testing sources, while presenting the same customer context to plugins.
- **FR-023**: Framework MUST fail clearly when a required delegated identity source is unavailable, instead of silently allowing anonymous or local fallback access.
- **FR-024**: Documentation MUST explain the boundary between Customer Auth, Member IAM, tenant context, and plugin business models.
- **FR-025**: Production deployments MUST treat PowerX Core or the configured platform identity source as the authority for customer login, token validation, membership, and bootstrap resolution; plugin-local identity sources are limited to development, standalone demos, migration adapters, or tests.
- **FR-026**: When customer register or login contracts are exposed as C-end endpoints, Framework MUST require abuse protection such as rate limiting, lockout/backoff, traceable denial, and sensitive credential redaction.

### Key Entities

- **CustomerContext**: Runtime C-end identity context attached to a protected request. It represents who the external user is, which tenant is active, which membership applies, what customer-side roles/scopes are available, and where the identity came from.
- **CustomerMembership**: Relationship between a customer and a tenant. It indicates whether the customer may access the current tenant and may contribute roles or scopes for that tenant. In production, platform membership is authoritative even when framework uses a short-lived cache.
- **CustomerTokenValidationResult**: Normalized outcome of validating a customer token through a platform, delegated, local, third-party, or mock identity source.
- **BootstrapContext**: Tenant context resolved from C-end entry hints before or during authentication, such as scene, invite code, organization code, channel, or tenant hint.
- **CustomerAuthResult**: Result of customer registration or login, including issued credentials and the normalized customer context.
- **CustomerAuthError**: Stable error outcome for authentication, membership, tenant mismatch, bootstrap, delegated identity source, or missing context failures.
- **CustomerAuthSource**: Identity source category used for diagnostics and policy decisions, such as platform, delegated, third-party login, local development, or mock testing.

## Assumptions & Dependencies

- `018-framework-iam-unification` remains the member/back-office identity feature and is not extended to model C-end customer identity.
- Existing tenant context and tenant isolation rules remain authoritative for tenant-scoped data access after customer auth resolves the current tenant.
- PowerX Core or an equivalent platform identity source will provide or expose customer registration, login, token validation, membership, and bootstrap entry capabilities.
- Local development may use local or mock customer auth, but production deployments use platform-managed or delegated identity sources as the customer identity authority. Production local/mock use is limited to explicit break-glass exceptions with audit visibility.
- Platform membership status changes are expected to be observable by framework through short TTL refresh, explicit invalidation, or conservative rejection behavior.
- Existing skeleton customer auth behavior is treated as the compatibility baseline during migration.

## Success Criteria

- **SC-001**: 100% of protected C-end request paths can obtain customer, tenant, and membership context through the framework without re-parsing request tokens in business handlers.
- **SC-002**: 100% of token/request tenant mismatches are rejected before business logic executes.
- **SC-003**: 100% of missing, disabled, or suspended customer memberships are rejected before tenant-scoped business data is accessed.
- **SC-004**: At least three identity source modes, delegated/platform, local development, and mock testing, can produce the same normalized customer context in acceptance tests.
- **SC-005**: Existing skeleton protected C-end auth scenarios continue to pass after migration to framework contracts.
- **SC-006**: New plugin example tests use the standard customer auth helpers without defining custom token signing, context injection, validator mock, or membership resolver mock helpers.
- **SC-007**: Documentation clearly identifies Customer Auth versus Member IAM boundaries, with no industry-specific customer business model included in the framework specification.
- **SC-008**: Cached membership decisions never outlive token validity, and membership disable/suspend scenarios are rejected in validation tests.
- **SC-009**: Production startup validation rejects local/mock identity sources by default, and accepted break-glass exceptions are 100% visible in audit or diagnostics output.
