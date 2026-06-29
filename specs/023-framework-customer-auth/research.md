# Research: Framework Customer Identity/Auth

## Decision 1: 新建 023，而不是并入 018 或 010

**Decision**: 新建 `023-framework-customer-auth`。

**Rationale**: `018-framework-iam-unification` 面向后台 member IAM；`010-auth-customer` 是 skeleton/mini-app customer auth 的早期实现计划。当前需求是 framework 级 C 端 customer identity/auth 契约，必须与后台 member IAM 语义隔离，并从 skeleton 实现中抽象上提。

**Alternatives considered**:

- 放入 018：会把后台 member 与 C 端 customer 权限模型混淆。
- 延续 010：010 偏 skeleton 本地/委托双模落地，不足以定义 framework 公共契约。
- 放入插件业务 feature：会把行业模型误带入 framework。

## Decision 2: 生产身份权威源为 PowerX Core/平台

**Decision**: 生产环境 customer login、token validation、membership、bootstrap 以 PowerX Core 或配置的平台身份源为权威。

**Rationale**: Customer identity 是平台级能力。插件自建生产权威源会导致重复账号体系、token 语义漂移、membership 跨租户风险和审计割裂。

**Alternatives considered**:

- 插件生产自主管理 customer：安全与治理成本高，且违背 framework 不持久化行业 customer 的边界。
- 混合权威：Core 管 token、插件管 membership 会让越权判断分散，不利于 shared app。
- 完全配置化无约束：容易误把 local/mock 带入生产。

## Decision 3: 支持全局 customer token，但租户资源必须有当前 tenant

**Decision**: Customer token 可表达全局 customer 身份；访问 tenant-scoped 资源时必须从 token、请求或 bootstrap 解析当前 tenant，并强制校验 membership。

**Rationale**: C 端用户可能属于多个租户或机构。强制 token 单租户绑定会降低 shared app 体验；完全不绑定 tenant 则会导致跨租户风险。全局身份 + 当前 tenant + membership 校验是更稳妥的模型。

**Alternatives considered**:

- token 必须单租户：简单但不适合多租户 customer 登录与机构选择。
- token 携带多 tenant 自动选择第一个：不可预测，容易误入错误租户。
- 只依赖 tenant middleware：缺少 customer-tenant membership 证明。

## Decision 4: Membership 可短 TTL 缓存，但平台状态为权威

**Decision**: Framework 可缓存 membership 判定，但 TTL 不得超过 token 有效期，并且必须支持显式失效或在状态变化后采取保守拒绝。

**Rationale**: 每次请求实时查询平台 membership 成本高；仅信任 token claims 则无法及时反映禁用/挂起。短 TTL + 失效机制可以平衡性能和安全。

**Alternatives considered**:

- 每次请求实时查询：安全性好，但对高频 C 端请求成本较高。
- token claims 为唯一权威：性能好，但 membership 状态变更滞后。
- 插件本地 membership 为权威：与生产平台权威源冲突。

## Decision 5: 多 token 同时存在必须一致

**Decision**: 当请求同时携带多个支持的 customer token 凭证时，必须解析为同一 customer 与 tenant 上下文；不一致时拒绝。

**Rationale**: Host/proxy 场景可能保留多个 header。简单优先级策略会隐藏冲突并留下绕过空间；直接禁止双 token 又会降低代理兼容性。

**Alternatives considered**:

- 永远优先 Authorization：可能忽略代理注入 token 的冲突。
- 永远优先 X-Customer-Token：同样存在绕过风险。
- 双 token 直接拒绝：安全但对代理/迁移链路不够友好。

## Decision 6: 生产 local/mock 默认禁止，break-glass 例外需审计

**Decision**: 生产环境默认拒绝 local/mock identity source；只有显式 break-glass 配置并输出审计/诊断标记时允许启动。

**Rationale**: local/mock 是开发测试工具，误入生产会绕过平台身份源。保留 break-glass 是为了紧急迁移、受控演示或平台依赖故障排查，但必须可审计。

**Alternatives considered**:

- 完全禁止：最安全，但缺少受控应急空间。
- 允许强 secret local：仍然会形成生产第二身份源。
- framework 不判断：风险转嫁给每个插件。

## Decision 7: Skeleton 迁移采用 adapter/wrapper 兼容

**Decision**: 先将 framework customer auth 契约落地，再把 skeleton 现有 local/delegate authenticator、context middleware 和 mini-app route 接入为 adapter/wrapper。

**Rationale**: skeleton 已有完整 MVP 行为和测试。直接删除重写风险高；adapter/wrapper 可保持现有受保护接口行为，同时逐步减少 skeleton internal customer auth 的公共职责。

**Alternatives considered**:

- 一次性替换全部 skeleton customer auth：改动面大且容易破坏现有回归。
- 只新增 framework 包不迁移 skeleton：无法证明 framework 能承接真实插件路径。

