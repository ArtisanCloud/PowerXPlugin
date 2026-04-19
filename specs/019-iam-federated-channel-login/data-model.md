# Data Model: IAM 联邦渠道扫码登录（企微/钉钉/飞书）

## 1. Federated Provider Definition

- **Purpose**: 描述渠道提供方及启用状态。  
- **Key Fields**:
  - `provider_code` (`wecom|dingtalk|lark`)
  - `tenant_uuid`
  - `enabled`
  - `config_version`
  - `updated_at`
- **Rules**:
  - 同租户同 provider 仅允许一条有效配置。
  - `enabled=false` 时不得发起扫码 challenge。

## 2. Login Challenge

- **Purpose**: 扫码登录挑战上下文（一次性授权入口）。  
- **Key Fields**:
  - `challenge_id`
  - `tenant_uuid`
  - `provider_code`
  - `state`
  - `nonce`
  - `expires_at`
  - `redirect_uri`
  - `trace_id`
  - `status` (`pending|used|expired|revoked`)
- **Rules**:
  - `state` 与 `nonce` 必须随机且不可预测。
  - challenge 只能被消费一次；二次消费视为 replay。

## 3. External Identity

- **Purpose**: 渠道侧用户主体标识。  
- **Key Fields**:
  - `provider_code`
  - `tenant_uuid`
  - `external_user_id`
  - `union_id` (optional)
  - `mobile` (optional)
  - `email` (optional)
  - `raw_profile_hash`
- **Rules**:
  - 主键语义：`tenant_uuid + provider_code + external_user_id`。
  - 外部身份不得跨租户复用绑定。

## 4. Identity Binding

- **Purpose**: external identity 与本地 member 关系。  
- **Key Fields**:
  - `binding_id`
  - `tenant_uuid`
  - `provider_code`
  - `external_user_id`
  - `member_id`
  - `status` (`active|unbound|disabled`)
  - `source` (`jit|admin`)
  - `mapping_version`
  - `created_at`
  - `updated_at`
- **Rules**:
  - 同一 external identity 仅允许一个 `active` 绑定。
  - 解绑后应触发会话撤销策略（或强制短期失效）。

## 5. Mapping Policy

- **Purpose**: 角色/部门映射规则。  
- **Key Fields**:
  - `policy_id`
  - `tenant_uuid`
  - `provider_code`
  - `version`
  - `role_rules`
  - `department_rules`
  - `enabled`
  - `updated_at`
- **Rules**:
  - 登录时比较 `binding.mapping_version` 与最新 policy version。
  - 仅版本变化时触发重算并更新绑定版本。

## 6. Federated Session Projection

- **Purpose**: 联邦登录后统一上下文投影。  
- **Key Fields**:
  - `tenant_uuid`
  - `user_id`
  - `member_id`
  - `roles[]`
  - `permissions[]`
  - `provider_code`
  - `trace_id`
  - `auth_source` (`password|federated`)
- **Rules**:
  - standalone/delegated 输出结构必须一致。
  - delegated 模式下令牌权威来自宿主，插件仅适配输出。

## 7. Risk Event

- **Purpose**: 记录挑战与回调风控判定。  
- **Key Fields**:
  - `event_id`
  - `tenant_uuid`
  - `provider_code`
  - `risk_type` (`expired|replay|cross_tenant|signature_invalid|state_mismatch`)
  - `decision` (`allow|deny`)
  - `error_code`
  - `evidence`
  - `trace_id`
  - `occurred_at`
- **Rules**:
  - `decision=deny` 必须有 `error_code`。
  - 风控拒绝对外返回可区分错误码；前端展示通用失败文案。

## Relationships

- Provider Definition 1:N Login Challenge  
- External Identity 1:1 Active Identity Binding  
- Identity Binding N:1 Member  
- Mapping Policy 1:N Identity Binding（按 version 关联）  
- Login Challenge 1:N Risk Event
