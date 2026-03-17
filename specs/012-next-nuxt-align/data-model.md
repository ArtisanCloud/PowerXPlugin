# Data Model

> 本特性不新增后端持久化实体；数据模型用于描述“迁移与一致性验证”的领域对象。

## PageParityMap（页面对照映射）
- **Purpose**: 描述 Nuxt 页面与 Next 页面之间的一一映射关系与验证状态。
- **Key Fields**:
  - `domain`（Auth/Templates/IAM/Capabilities/Integration/Operations/Security）
  - `nuxt_route`
  - `next_route`
  - `mode_scope`（standalone/host/both）
  - `status`（pending/migrated/verified/blocked）
  - `owner`
- **Validation Rules**:
  - 每个 `nuxt_route` 必须且仅能映射一个 `next_route`。
  - `status=verified` 必须存在对应验证用例标识。

## SessionState（会话状态）
- **Purpose**: 统一描述会话生命周期与异常处理行为。
- **States**: `unauthenticated` -> `authenticated` -> `refreshing` -> `expired`
- **Transitions**:
  - 登录成功：`unauthenticated` -> `authenticated`
  - Token 即将过期：`authenticated` -> `refreshing`
  - Refresh 成功：`refreshing` -> `authenticated`
  - Refresh 失败或过期：`refreshing/authenticated` -> `expired`
- **Validation Rules**:
  - `expired` 状态必须触发一致的重定向与提示。

## VisibilityRuleSet（权限可见规则集）
- **Purpose**: 定义页面/操作在不同角色权限下的可见性与可执行性。
- **Key Fields**:
  - `resource`
  - `action`
  - `role_scope`
  - `ui_visibility`（show/hide/readonly）
  - `error_semantics`
- **Validation Rules**:
  - 规则结果需在宿主模式与独立模式一致。

## CapabilityInvocationFlow（能力调用流程）
- **Purpose**: 追踪能力调用在前端侧的请求、状态变化与结果呈现。
- **Key Fields**:
  - `capability_id`
  - `request_payload_digest`
  - `invoke_status`（submitted/running/succeeded/failed）
  - `trace_ref`
  - `error_display_code`
- **Validation Rules**:
  - `failed` 状态必须输出可区分的错误语义。

## ParityGapRecord（差异归因记录）
- **Purpose**: 记录 Nuxt 与 Next 差异问题的归因结论与处置路径。
- **Key Fields**:
  - `gap_id`
  - `domain`
  - `symptom`
  - `baseline_reference`（Nuxt 证据）
  - `root_cause`（next_deviation/gin_defect/unknown）
  - `decision`
  - `opened_at`
  - `resolved_at`
- **Validation Rules**:
  - `resolved_at - opened_at` 不得超过 2 个工作日。
  - 若 `root_cause=gin_defect`，`decision` 必须标记“最小化修复 + 双端回归”。
