# Feature Specification: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Feature Branch**: `019-iam-federated-channel-login`  
**Created**: 2026-04-11  
**Updated**: 2026-04-13  
**Status**: Draft  
**Input**: User description: "支持 SCRM 渠道账号同步员工并扫码登录（企业微信/钉钉/飞书），输出 provider 抽象、扫码授权回调、账号绑定/映射、JIT 入库、角色映射策略、审计与风控。"

## Clarifications

### Session 2026-04-12

- Q: 可复用给其他插件的 provider factory 与默认渠道实现应放在哪里？ → A: 放在 framework，可版本化复用；skeleton 仅做装配、路由接线与运行时配置。
- Q: 首次扫码未绑定本地账号时默认 JIT 策略？ → A: 仅自动绑定“可唯一匹配”的现有成员；无法唯一匹配时转管理员处理。
- Q: 角色/部门映射默认生效时机？ → A: 登录时按映射版本检查，仅在版本变化时重算并覆盖权限上下文。
- Q: 风控拦截后的对外反馈默认策略？ → A: 返回可区分错误码，前端统一展示通用失败文案，详细原因只进审计。
- Q: delegated 模式联邦登录结果由谁作为权威？ → A: 宿主会话/令牌为权威，插件仅做上下文适配与最小缓存。

### Session 2026-04-20

- Q: 在现有企微链路已可用后，钉钉/飞书应先做什么层级？ → A: 先做“配置与登录主链路对齐（P1）”，组织同步（P2）随后补齐。
- Q: 钉钉与飞书谁优先？ → A: 钉钉优先（组织模型更接近企微），飞书紧随其后。
- Q: 本阶段是否强制引入第三方 SDK？ → A: 不强制；以 provider 抽象为主，SDK 作为可替换实现细节。

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 员工扫码登录插件系统 (Priority: P1)

作为企业员工，我希望通过企微/钉钉/飞书扫码登录插件系统，无需维护独立密码。

**Why this priority**: 这是联邦登录的最小可交付价值，直接影响员工登录效率和推广可行性。

**Independent Test**: 在单租户环境配置一个可用渠道，完成“扫码 -> 授权回调 -> 会话建立 -> 进入管理页”全链路验证。

**Acceptance Scenarios**:

1. **Given** 渠道配置有效且员工已在渠道侧存在，**When** 员工完成扫码授权，**Then** 系统建立登录态并返回统一身份上下文。
2. **Given** 员工首次扫码且本地尚无绑定关系，**When** 登录流程继续，**Then** 系统仅在“唯一匹配本地成员”时自动绑定，否则要求管理员处理并输出审计结果。

---

### User Story 2 - 渠道账号与 IAM 身份统一映射 (Priority: P2)

作为租户管理员，我希望对渠道账号与本地成员的绑定、解绑和角色映射进行可控管理。

**Why this priority**: 没有可治理的映射关系，扫码登录无法稳定纳入组织权限模型。

**Independent Test**: 管理员调整绑定和角色映射后，成员下一次扫码登录即按新映射生效，且变更可追溯。

**Acceptance Scenarios**:

1. **Given** 已存在本地成员，**When** 管理员执行绑定，**Then** 成员可通过扫码直接登录并继承映射角色/部门权限。
2. **Given** 管理员修改角色映射，**When** 成员再次扫码，**Then** 系统检测映射版本变化后重算权限并记录新版本。

---

### User Story 3 - 风控与审计可追溯 (Priority: P3)

作为安全与运维人员，我希望扫码登录具备完整风控与审计能力，阻断重放和跨租户误绑定风险。

**Why this priority**: 外部身份输入引入安全边界变化，必须确保 Zero Trust 与租户隔离规则不退化。

**Independent Test**: 构造过期 state、重复 code、跨租户回调、签名异常等流量，验证系统拒绝并产生可检索事件。

**Acceptance Scenarios**:

1. **Given** state 过期或 nonce 不匹配，**When** 回调到达，**Then** 系统拒绝登录、返回可区分错误码并写入风险事件。
2. **Given** 相同授权码被重复使用，**When** 二次回调到达，**Then** 系统拦截重放、返回可区分错误码并输出审计日志。

---

### Edge Cases

- 渠道返回身份字段不完整（无手机号/邮箱/union_id）时，JIT 策略如何降级。
- 同一 external identity 命中多个租户候选时，如何避免跨租户自动绑定。
- 绑定已解除但历史会话未过期时，如何触发会话失效与强制重登。
- 渠道 API 短时故障时，如何保证密码登录和现有会话不受影响。
- delegated 模式上游身份不可用时，如何回退并保持错误语义一致。
- 回调失败时前端需要统一展示通用文案，避免暴露风控细节。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 在 framework 提供可复用的联邦 provider 抽象与 registry/factory 能力，支持多插件直接复用。
- **FR-002**: 系统 MUST 在 framework 提供默认渠道实现扩展点，至少覆盖企业微信、钉钉、飞书三类 provider。
- **FR-003**: 系统 MUST 支持扫码授权发起、挑战校验、回调处理、登录态建立的统一流程。
- **FR-004**: 系统 MUST 支持 external identity 与本地 IAM 成员的绑定、解绑、查询，并保证租户隔离。
- **FR-005**: 系统 MUST 支持首次扫码 JIT 绑定/创建策略，并支持租户级开关与策略选择。
- **FR-005a**: 默认 JIT 策略 MUST 为“仅唯一匹配自动绑定”；未唯一匹配时 MUST 进入管理员处理路径。
- **FR-006**: 系统 MUST 支持角色与部门映射策略，并在登录后注入统一权限上下文。
- **FR-006a**: 默认映射生效策略 MUST 为“版本变化时重算”，避免每次登录无差别重算。
- **FR-007**: 系统 MUST 提供密码登录与扫码登录并存机制，单一渠道故障不影响整体可登录性。
- **FR-008**: 系统 MUST 实施风控校验（state、nonce、ttl、replay、tenant boundary、signature validity）。
- **FR-008a**: 风控拒绝 MUST 返回可区分错误码给调用方，前端 MUST 使用统一通用文案呈现失败提示。
- **FR-009**: 系统 MUST 输出完整审计事件，至少包含 provider、tenant、external identity、binding outcome、risk decision、trace。
- **FR-010**: 系统 MUST 在 standalone 与 delegated 两种模式下输出一致的身份上下文结构与错误语义。
- **FR-010a**: delegated 模式下登录权威 MUST 为宿主会话/令牌，插件仅负责上下文适配与最小缓存。
- **FR-011**: skeleton MUST 仅负责 framework factory 装配、配置注入、路由接线与页面对接，不重复实现渠道底层能力。
- **FR-012**: 当渠道返回身份字段不完整（手机号/邮箱/union_id 缺失）时，系统 MUST 禁止自动创建新成员，并转入管理员处理路径且记录审计原因码。
- **FR-013**: 当绑定关系被管理员解除时，系统 MUST 使该绑定对应的历史会话失效，并要求下一次访问重新认证。
- **FR-014**: delegated 模式上游身份不可用时，系统 MUST 保持与 standalone 一致的错误码语义，并允许密码登录链路继续可用。
- **FR-015**: 系统 MUST 为 `dingtalk` 与 `lark` 提供与 `wecom` 对齐的租户级配置读取/保存能力（Admin API），并支持 challenge 阶段按租户配置动态解析 provider 参数。
- **FR-016**: 系统 MUST 在浏览器扫码登录回调链路中，对 `wecom|dingtalk|lark` 统一支持 callback host 重写策略（通过租户配置注入 host）。
- **FR-017**: 系统 MUST 为 `dingtalk` 与 `lark` 提供与 `wecom` 一致的渠道同步任务能力（触发、列表、清空、进度事件），并保证任务状态在刷新后不回退。
- **FR-018**: 系统 MUST 在渠道同步落库时复用统一幂等键：`tenant_uuid + provider + tenant_scope + external_user_id`，并用于后续再次同步 upsert。
- **FR-019**: 系统 MUST 在 Web Admin 为 `dingtalk` 与 `lark` 提供可操作配置页（非占位页），字段、校验、主题适配与错误提示遵循现有 `wecom` UX 基线。

### Key Entities *(include if feature involves data)*

- **Federated Provider Definition**: 渠道提供方定义（标识、状态、租户可见范围、能力声明）。
- **Provider Factory**: framework 对外暴露的 provider 构造与注册实体。
- **External Identity**: 渠道侧主体标识（provider + external_user_id + tenant_scope）。
- **Identity Binding**: 外部身份与本地成员绑定关系（状态、来源、版本、生效时间）。
- **Login Challenge**: 扫码挑战对象（state、nonce、ttl、redirect_context、trace）。
- **Mapping Policy**: 角色/部门映射策略对象（规则、优先级、版本）。
- **Risk Event**: 风控事件对象（风险类型、证据、决策、处置结果）。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 三个渠道中任意一个接入后，扫码登录主流程成功率在灰度期达到 98% 及以上。
- **SC-002**: 首次扫码 JIT 绑定成功率达到 95% 及以上，且失败原因可定位率达到 100%。
- **SC-003**: 重放、过期、跨租户、签名异常等高风险回调拦截率达到 100%。
- **SC-004**: 接入扫码登录后，目标租户员工密码登录占比在 30 天内下降至少 50%。
- **SC-005**: 新插件接入联邦登录时，复用 framework factory 的接入步骤较自研实现减少至少 40%。

## Error Semantics（Polish）

- 风控拒绝统一使用可区分错误码（如 `FEDERATED_RISK_SIGNATURE_INVALID`、`FEDERATED_RISK_REPLAY`、`FEDERATED_RISK_TENANT_BOUNDARY`）。
- 回调 challenge 无效统一使用 `FEDERATED_INVALID_CHALLENGE`。
- provider 兑换/解析失败统一对外返回通用失败文案“登录失败，请稍后重试”，内部审计记录原因码。
- delegated 上游不可用与 standalone 本地不可用统一归一为 `FEDERATED_AUTH_UNAVAILABLE` 语义。
