# 套件插件 Framework 消费就绪台账

授权查询与 Framework 交付状态更新：2026-09-08。下表套件自身的审计结果仍为 2026-09-04 快照，本轮未重新审计套件工作区，不能据此认定其当前实现仍有相同缺口。

本台账约束 Framework 对 PowerX 套件插件能力的消费边界。它不是能力目录，也不授权插件调用任何接口。只有同时具备稳定的 Host Contract、租户隔离、服务身份授权与合同测试的能力，才可以新增对应的强类型 Framework client。

## 统一准入条件

套件插件提供给其他插件的业务能力必须满足以下条件：

1. 以 capability 为唯一入口，并声明版本化的 input/output schema；不得把 Admin 页面路由或插件内部路由当作 Host API。
2. 所有业务对象及关联均使用 UUID。正式 Contract 不接受 numeric ID、`id` 或可猜测的内部主键作为跨插件参数。
3. tenant 仅从 Gateway API Key 或 STS service token 推导；请求体、查询参数和 header 中的 `tenant_uuid` 必须拒绝。
4. service actor 必须经过 capability 发布、tenant registration 与凭证 `allowed_capabilities` grant 校验；`sts_direct` 不能替代授权。
5. 返回稳定 DTO、稳定 `reason_code`，并覆盖 200、400、401、403、跨租户 404 与上游 502/503 合同测试。
6. Framework delegated adapter 只能调用上述 Host Contract；local adapter 由消费插件按相同 contract 注入。业务 handler 不得自己选择模式、直查套件数据库或拼套件内部 URL。

## 已审计套件插件

| 套件插件 | 已发现能力 | 当前状态 | 阻断 Framework 强类型 client 的缺口 |
| --- | --- | --- | --- |
| SCRM（`com.powerx.plugins.scrm`） | `com.powerx.plugins.scrm.leads.read` 及相关 REST exposure | `contract_gap` | 路径和输入仍使用 `lead_id`；能力描述没有版本化 input/output schema；现有 JWT exposure 不是声明了 STS service-actor grant 的 Host Contract。 |
| e-commerce（`com.powerx.plugins.ecommerce`） | `com.powerx.plugins.ecommerce.product.sku.read` 及产品相关能力 | `contract_gap` | REST 路由使用 `:id`；当前 SKU 输出 schema 为开放对象，未声明 UUID DTO；尚无可审计的 tenant/grant/稳定错误合同。 |
| CRM | 本工作区未发现可审计的已安装 checkout | `not_audited` | 需先提供 capability 清单、OpenAPI/schema、STS/API-Key grant 策略和合同测试证据。 |

以上状态不表示能力不可供其自身 Admin UI 使用；它只表示 Framework 尚不能把它们承诺为跨插件稳定 SDK。

## SCRM 需要的 Contract 交付

SCRM / PowerX 侧需发布新的、面向 service actor 的 Lead Host Contract，而不是给既有 Admin 路由增加兼容入口：

1. 为列表、详情及实际需要的活动读取定义独立 capability、OpenAPI 和版本化 JSON schema。
2. 对外 DTO 和路径统一使用 `lead_uuid`、`activity_uuid` 等 UUID；删除正式 Contract 中的 `lead_id` / `activity_id` 输入。
3. 由 STS 或 Gateway API Key 派生 tenant，拒绝任何 caller supplied `tenant_uuid`。
4. 将 capability 绑定到发布、tenant registration、插件实例 credential grant；最小权限不复用 Admin 用户 RBAC。
5. 规定稳定错误 envelope，并补齐 success、无认证、未授权、跨租户不存在、非法 UUID、上游失败的合同测试。

交付完成后，Framework 才可新增 `runtime/powerx/scrm`，并仅实现已有真实消费者需要的 DTO 和 operation。

## e-commerce 需要的 Contract 交付

e-commerce / PowerX 侧需把 SKU 读取能力提升为正式 Host Contract：

1. 将 `:id` 替换为 `sku_uuid`（及需要时的 `product_uuid`），并在列表、详情、关联对象中保持 UUID-only。
2. 将开放的 output schema 改为明确版本化 DTO，至少声明 UUID、业务可读名称、状态、分页/cursor 及可选字段的语义。
3. 明确 capability 的 service-actor 认证、tenant 推导和 credential grant；不允许 consumer 直接访问 `/admin/product/*`。
4. 统一稳定错误码和 `reason_code`，并提供全套合同测试。

完成后，首个候选包是 `runtime/powerx/ecommerce`；是否实现取决于是否有实际消费插件，避免先建空 SDK。

## Framework 后续动作

| 条件 | Framework 动作 |
| --- | --- |
| 任一准入条件缺失 | 仅在 Capability Lab 显示为 `contract_gap`；不增加 typed client，不用 generic invoke 伪装业务 SDK。 |
| Contract 已发布但未有消费者 | 记录为 `ready_for_consumer`，等待消费场景确定 DTO 边界。 |
| Contract 已发布且有消费者 | 新增独立 `runtime/powerx/<module>` 包，完成 DTO、delegated transport、稳定错误映射、Skeleton 装配及 local/delegated contract tests。 |
| 插件安装态验证 | 核验 manifest capability、tenant registration 和 STS `allowed_capabilities`；API Key 成功不能替代 STS 验收。 |

Capability Lab 的职责是展示目录和通用安全诊断；Host Contract Lab 的职责是验证已经准入的强类型模块。两者都不得绕过上述准入条件。

## 授权状态查询（已实现）

目录返回 capability 的存在性和提供方，不等于当前插件凭证拥有调用权。Core 已提供 `POST /api/v1/tenant/capabilities:grant-status`；Framework 的 `runtime/powerx/capability.Client.GrantStatus` 已接入。输入为 capability ID 列表，tenant 与调用主体由 Gateway API Key 或 STS 推导，逐项返回 `granted`、`not_granted` 或 `unknown`。

Capability Lab 已通过 typed `GrantStatus` client 查询当前凭证状态；查询失败不得伪装成已授权或单项未授权。不得根据 catalog、manifest 或 API Key 存在性推断已授权。suite capability 仍须满足本台账的准入条件，才可获得 typed business client。

Framework `runtime/capability.RequireGrants` 已用于 Skeleton delegated 启动检查：按每批最多 100 项检查显式 required 的完整性、顺序、reason_code 和 granted 状态。它不自动授予权限、不缓存永久授权，也不替代 Core 对每次业务调用的校验。安装态 STS 和授权撤销验证仍分别由 T062–T064、T073 跟踪。

### T066 已接入的 Core Host Contract

`POST /api/v1/tenant/capabilities:grant-status` 是“调用者查询自己的授权状态”的只读诊断接口。调用者还须获授该接口自身的 `com.corex.capabilities.grant_status.read`，不能复用 Admin capability 目录或以一次真实业务调用探测授权。

请求只允许：

```json
{
  "capability_ids": [
    "com.powerx.plugins.scrm.leads.read"
  ]
}
```

规则如下：

1. 认证接受 Gateway API Key 或 STS service token；tenant、STS plugin/service actor 和 API-key profile 均从已验证凭证推导。body 仅接受 `capability_ids`；query 中的 `tenant_uuid`、`plugin_id` 和非空 `X-Tenant-UUID`、`X-Plugin-ID` 会被拒绝，返回 400，reason_code 为 `CAPABILITY_GRANT_STATUS_INVALID_ARGUMENT`。
2. `capability_ids` 必须包含 1–100 个合法能力标识，拒绝空白及重复输入，并保持响应与请求一一对应的顺序；不得静默去空白或去重。
3. 每项仅返回 `capability_id`、`status` 和稳定的机器可读 `reason_code`；不得回传另一插件的 credential、注册详情、用户角色或任意 PII。
4. `granted` 表示该 capability 已发布、当前 tenant 已注册，且当前 STS 凭证的 `allowed_capabilities`（或当前 API-key 的精确 scope 映射）实际包含它。
5. `not_granted` 表示 capability 对当前 tenant 可用，但当前凭证未被授予；`unknown` 表示 capability 不存在、未发布、当前 tenant 未注册或无法作为该 credential 类型调用。两者均为 HTTP 200 的逐项状态，不泄露其他 tenant 或插件的安装状态。
6. 无效/缺失凭证为 401，reason_code 为 `CAPABILITY_GRANT_STATUS_UNAUTHORIZED`；调用者无权使用本诊断接口为 403、`CAPABILITY_GRANT_STATUS_FORBIDDEN`；Core 服务依赖故障映射为 503、`CAPABILITY_GRANT_STATUS_UPSTREAM_DEPENDENCY`。Framework transport 自身失败与 Core 响应应分开判定；完整请求失败不得伪装成单项 `not_granted`。
7. Core 必须提供 OpenAPI、capability/route 声明、STS/API-key 双路径合同测试，以及 tenant、credential、grant 三层校验；`sts_direct` 不构成 grant。

成功响应的 `data` 结构：

```json
{
  "items": [
    {
      "capability_id": "com.powerx.plugins.scrm.leads.read",
      "status": "not_granted",
      "reason_code": "CAPABILITY_NOT_GRANTED"
    }
  ]
}
```

Framework 将逐项状态展示为诊断信息，但仍只有通过本台账“统一准入条件”的能力能进入 typed client 或 Host Contract Lab 调用。当前稳定模块的 local adapter contract、Factory 接入和验收要求见 [插件接入手册](../guides/features/009-consume-powerx-capability/guide.md)；插件可以开始填充 local 实现，不需要等待套件客户端或 Agent Session 扩展。
