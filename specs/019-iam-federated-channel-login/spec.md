# Feature Specification: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

**Feature Branch**: `019-iam-federated-channel-login`  
**Created**: 2026-04-11  
**Status**: Draft  
**Input**: User description: "支持 SCRM 渠道账号同步员工并扫码登录（企业微信/钉钉/飞书），输出 provider 抽象、扫码授权回调、账号绑定/映射、JIT 入库、角色映射策略、审计与风控。"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - 员工扫码登录插件系统 (Priority: P1)

作为企业员工，我希望通过企微/钉钉/飞书扫码登录插件系统，无需单独记密码。

**Why this priority**: 这是业务侧最直接需求，也是联邦登录改造的核心价值。

**Independent Test**: 在单租户环境使用任一渠道完成“扫码 -> 回调 -> 登录态建立”全链路验证。

**Acceptance Scenarios**:

1. **Given** 企业已配置可用渠道提供方，**When** 员工扫码并完成授权，**Then** 系统创建有效登录会话并返回统一身份上下文。
2. **Given** 员工首次扫码登录，**When** 系统未找到本地绑定，**Then** 按策略执行 JIT 绑定或创建并返回可审计结果。

---

### User Story 2 - 渠道账号与 IAM 身份统一映射 (Priority: P2)

作为租户管理员，我希望外部渠道员工与本地 IAM 账号可管理地绑定/解绑，并支持角色、部门映射。

**Why this priority**: 若无可控映射，扫码登录难以进入组织权限体系，无法用于生产。

**Independent Test**: 管理员执行绑定、解绑、角色映射更新后，验证员工下次扫码权限即时生效。

**Acceptance Scenarios**:

1. **Given** 已存在本地成员，**When** 管理员绑定渠道账号，**Then** 该成员可通过扫码直接登录并继承映射角色。
2. **Given** 角色映射规则变更，**When** 成员再次扫码登录，**Then** 权限按新规则生效并保留变更审计。

---

### User Story 3 - 风控与审计可追溯 (Priority: P3)

作为安全与运维人员，我希望扫码登录过程具备完整审计与风控策略，防止重放、越权与误绑定。

**Why this priority**: 联邦登录引入外部身份输入，必须满足审计和风险控制要求。

**Independent Test**: 构造过期 state、回调重放、跨租户回调等异常流量，验证系统拒绝并输出审计记录。

**Acceptance Scenarios**:

1. **Given** 回调 state 失效或不匹配，**When** 渠道回调到达，**Then** 系统拒绝登录并记录风控事件。
2. **Given** 同一授权码被重复使用，**When** 二次回调发生，**Then** 系统拒绝重放并输出可检索审计日志。

---

### Edge Cases

- 渠道返回用户信息不完整（无手机号/邮箱）时的绑定策略。
- 员工在多个租户存在同标识时，如何避免跨租户误绑定。
- 渠道账号已解绑但会话未失效时，如何进行会话撤销。
- 外部渠道不可用时，如何与密码登录共存并保障可用性。

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: 系统 MUST 提供统一联邦登录 provider 抽象，至少支持企微、钉钉、飞书三类渠道扩展。
- **FR-002**: 系统 MUST 支持扫码授权发起、回调校验、登录态建立完整链路。
- **FR-003**: 系统 MUST 支持外部身份与本地 IAM 成员绑定/解绑与查询。
- **FR-004**: 系统 MUST 支持首次登录 JIT 入库或绑定策略，并可按租户配置开关。
- **FR-005**: 系统 MUST 支持角色与部门映射策略，并在登录时应用到权限上下文。
- **FR-006**: 系统 MUST 提供密码登录与扫码登录并存策略，不因单一渠道故障导致整体不可登录。
- **FR-007**: 系统 MUST 对扫码登录链路执行风控校验（state、nonce、ttl、replay、tenant boundary）。
- **FR-008**: 系统 MUST 提供完整审计日志，至少记录 provider、tenant、external identity、binding result、risk decision、trace。
- **FR-009**: 系统 MUST 支持 standalone 与 delegated 模式下统一使用联邦登录结果并生成一致身份上下文。

### Key Entities *(include if feature involves data)*

- **Federated Provider**: 渠道提供方定义与配置实体。
- **External Identity**: 渠道侧用户标识（provider + external_user_id + tenant）。
- **Identity Binding**: 外部身份与本地成员的绑定关系与状态。
- **Login Challenge**: 扫码登录挑战对象（state/nonce/ttl/redirect context）。
- **Risk Event**: 登录风控判定记录（拒绝原因、证据、处置结果）。

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 三个渠道中任意一个接入后，扫码登录主流程成功率在灰度期达到 98% 及以上。
- **SC-002**: 首次扫码登录 JIT 绑定成功率达到 95% 及以上，且绑定错误可定位率 100%。
- **SC-003**: 重放/过期/跨租户回调攻击场景拦截率达到 100%。
- **SC-004**: 接入扫码登录后，目标租户员工密码登录占比在 30 天内下降至少 50%。

