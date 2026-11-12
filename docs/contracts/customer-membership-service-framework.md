# Framework 客户成员关系服务核验

权威合同为 [Core v1 快照](customer-membership-service-verification.md) 和 [OpenAPI](customer-membership-service-verification.openapi.yaml)，来源及 SHA-256 见 [来源记录](customer-membership-service-source.json)。SDK 的 testdata 包含同源 Schema 和合成响应夹具，发布模块可独立复跑合同测试。

## 调用

```go
gatewayClient, err := gateway.NewClient(gateway.Config{
    BaseURL: coreURL,
    TenantUUID: configuredTenantUUID,
    AuthScheme: "bearer",
    TokenProvider: serviceTokenProvider,
})
if err != nil { return err }
defer gatewayClient.Close()

verifier, err := customerfw.NewServiceMembershipVerifier(gatewayClient)
if err != nil { return err }
verified, err := verifier.Verify(ctx, customerUUID)
if err != nil { return err }
// 在本次业务操作内使用 verified，不跨交易缓存。
```

API Key 模式使用 Gateway 的 `AuthScheme: "apikey"` 和服务 `APIKey`。STS 模式由 `TokenProvider` 每次取得当前凭据。客户 JWT、Admin Token、调用方 tenant/plugin/actor 覆盖均不作为本接口输入。

`TenantUUID` 是受信部署配置中的本地响应断言，不授予身份或权限，也不改写 Core 的租户。`gateway.Client.TenantUUID()` 只读返回该配置；新 verifier 在构造时要求规范非零 UUID，每次调用检查范围未变化，并核对返回租户。未配置租户的其他 Gateway 客户端行为保持原合同。

请求只有固定 capability `com.corex.customer.memberships.service_read`、`preferred_protocol=core_internal` 和 `payload.body={operation:verify_customer,schema_version:1,customer_uuid}`。固定 binding 由 Core 选择；客户端不填写 endpoint、method、query 或自由 headers。

## 响应与失败

- `VerifiedCustomerMembership` 为一次状态快照，包含版本、四个 UUID、active 状态、可空到期时间、UTC 核验时间、revision 和调用 TraceID。它不证明客户本人登录、交易同意或订单所有权。
- 使用 Gateway 的原始 `RawData` 校验 `data.payload.item`；自定义 `ServiceMembershipGateway` 必须保留这些原始 JSON 字节。缺少原始数据明确失败，不改从 map 重建。重复字段、未知字段、非规范大小写、错版本、空对象及损坏值均拒绝。
- `expires_at` 必须存在；null 是无到期设置，缺失是坏响应。非空值必须为合法 RFC3339 时间，且严格晚于 Core 的 `checked_at`；UTC 核验时间与 revision 均须合法。
- 成功 TraceID 严格来自 `data.trace_id`，不使用 HTTP request ID 或外层请求 trace 代替。响应校验失败时尽可能保留该调用 trace。
- Core/Gateway 返回的 `*gateway.InvocationError` 原样返回，保留 HTTP 状态、公共错误码、`details.reason_code`、trace、请求 ID、details 及原始 body/cause。授权、平台故障和业务不存在不合并。
- 客户端自身错误为 `*customerfw.ServiceMembershipVerificationError`，公开 `Code`、`StatusCode`、`TraceID` 和 `Unwrap()`。Code 为机器码，由消费方在 locale 边界翻译；此服务客户端不使用客户登录专用的 `CodeOf`/`HTTPStatus` 映射。
- `context.Canceled` / `context.DeadlineExceeded` 保持 `errors.Is` 可识别。每次 Verify 仅一次调用；不缓存核验、不隐式重试、不查询客户列表或插件本地表，不构造客户 JWT。

现有 `AdminClient`、`AccountSelectorClient`、`DelegatedMembershipResolver` 和 `MembershipCache` 不承担此服务核验；客户本人双凭据合同保持独立。

## 验证与交付

```sh
cd framework/backend/go
GOWORK=off go test ./runtime/customerfw ./gateway ./internal/integration/gateway -count=1
GOWORK=off go test -race ./runtime/customerfw ./gateway ./internal/integration/gateway -count=1
GOWORK=off go vet ./runtime/customerfw ./gateway ./internal/integration/gateway
```

受控 HTTP transport 覆盖 API Key/STS 固定请求、服务凭据更新、成功后撤权的下一次请求、分类错误及 trace 保留、输入/响应 UUID、空与缺失字段、版本/时间/摘要损坏、重复与大小写变体、取消及超时。测试不监听端口、不访问数据库或真实 Core。

SDK 按 CI → 合并 → Go 子模块 tag → 公共 proxy/sumdb → 仓库外无 replace/workspace 的真实 SDK 消费验证发布。版本及公开验证收据以对应 GitHub Release 为准；受控测试不等于正式部署验收。

Core 所有者随后限定发布该能力、构建重启并给电商精确只读 grant，使用实际身份核对 grant-status。此能力无需业务迁移。电商升级已发布 SDK，验收有效/跨租户/停用/到期/主联系人异常及撤权场景后，再绑定 `Deps.CheckoutCustomers`；本轮 Framework 不修改其源码或日常实例。
