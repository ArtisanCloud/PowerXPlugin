# Framework Event Fabric 接入

应用合同为 `event_fabric.service.v1`。权威 Core 正文、OpenAPI、Proto 与摘要分别保存在同目录的 `event-fabric-service-v1.*` 和 `event-fabric-service-source.json`。本地 fixture 的维护者为 Framework，它来自上述合同，不代表 Core 已提供或验收的权威 fixture。

## 运行装配

SDK 包：`github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric`。

Host 使用 `com.corex.eventfabric.service_publish` / `com.corex.eventfabric.service_consume`，固定 `core_internal`，普通请求 `/tenant/invocations`，订阅 `/tenant/invocations/stream`。API prefix 只拼接一次。API Key 与 STS 在启动时选定；401/403 停止，不切换通道或凭据。SSE 正常 reconnect、EOF 和临时依赖失败使用有上限退避。连续恢复失败默认最多 6 次；正常 reconnect 和收到投递后重新计数。Close 取消响应体、重连及运行时持有的订阅。

## 真实租户映射与生命周期（2026-11-05）

Core 安装、恢复与租户映射规则见 [安装合同](plugin-event-fabric-install.md)。`resolve_topics` 只读，不准备资源，也不增加 grant 或 ACL。Framework 不调用 Admin 安装／启用入口。

`Client.ResolveTopics(ctx, ResolveTopicsRequest)` 明确选择 `PublishCapability` 或 `ConsumeCapability`；消费方向必须提供等于插件 ID 的 `SubscriberID`。请求只带业务 `TopicKeys`，不会接收自由 URL、method、headers 或物理主题。发布、订阅和 ACK/NACK DTO 不提供 `TopicKeys`，其实际请求不携带这个字段。

响应须同时匹配 contract、目标租户、插件和全部请求 key；缺失、多余或错误 key、外租户 FullTopic、空物理主题及不同 key 指向同一 FullTopic 都失败。两方向出现同 key 时，物理映射必须一致。FullTopic 直接取自 Core，不在本地拼接或修复。

为业务租户创建 Runtime 时优先使用 `NewTenantHostRuntime`，两方向分别核验，全部成功才交付 Runtime：

```go
// credentials 必须来自可信的目标租户服务凭据提供器。
// 不读取页面用户 JWT，不复用 System 的 API Key、STS provider 或静态映射。
options := eventfabric.TenantRuntimeConfig{
    Connection: eventfabric.Config{
        BaseURL: coreURL, APIPrefix: apiPrefix,
        AuthScheme: "apikey", APIKey: credentials.APIKey,
    },
    PluginID: pluginID,
    TenantUUID: credentials.TenantUUID,
    PublishTopicKeys: publishKeys,
    ConsumeTopicKeys: consumeKeys,
}
runtime, err := eventfabric.NewTenantHostRuntime(ctx, options)
if err != nil { return err }
defer runtime.Close()
bindings := runtime.TopicBindings() // 返回副本，可用于构造订阅 Topics。
```

使用 STS 时显式设置 `AuthScheme: "sts"` 和该租户独立的服务 `TokenProvider`；由 Core 验证真实凭据租户、精确 owner、实时 grant、registration、实例状态和逐主题 ACL。Framework 不从不透明 Key 猜测租户，不会通过改 tenant 字段让 System 凭据变成业务租户凭据。API Key 选定后不会调用 STS 兜底。

多租户并发任务分别持有独立 Runtime 与凭据提供器。单个界面／工作会话发生租户切换时，可由协调者持有零值 `eventfabric.RuntimeSlot`，调用 `slot.Switch(ctx, options)`：先关闭旧订阅和运行中请求，再解析新租户；失败后保持空槽，旧租户不再可用。`slot.Close()` 会取消进行中的查询并释放 Runtime。切换／关闭由生命周期协调者调用，不能在当前 Runtime 自己的消费处理回调内等待自身退出。业务处理必须响应取消；已提交业务仍按既有幂等记录处理，不因切换撤销提交或重跑副作用。

`NewHostRuntime` 的静态映射入口现在要求明确的凭据租户 UUID，所有 FullTopic 都必须属于该租户；没有无租户／跨租户通配 Runtime。System host-values 的平面 `topic_bindings` 只服务 System 自己，不表示业务租户已经完成准备。

`runtime/taskbus.HostProvider` 同时提供带 context 的 `NewResolvedRuntime(ctx)`；配置该租户的服务凭据、`PublishTopicKeys` 和 `ConsumeTopicKeys`。不得同时提供静态 `TopicBindings`。`NewEmitter()` 在显式配置业务 key 时选择这个解析入口，查询失败明确返回错误。Skeleton 与两份模板已转发这些字段：

```yaml
runtime:
  event_bridge:
    enabled: true
    mode: taskbus
    taskbus_provider: host
    fallback_to_local: false
    source_plugin: com.powerx.plugins.example
    topic_bindings: {}
    publish_topic_keys: [example.changed.v1]
    consume_topic_keys: []
```

这份配置只针对实际运行凭据的单个租户。处理其他业务租户时，应用必须按其真实服务凭据分别装配，不能仅修改全局配置里的租户 UUID。原有显式静态映射入口仍用于自己的凭据租户；不在解析错误时切回它。发布方向不能使用仅解析了消费权限的主题，消费方向也不能使用仅解析了发布权限的主题。

未准备、未注册、撤权、缺 key 和查询失败均保留明确错误；查询没有自动重试或缓存授权。后续发布、重连与 ACK/NACK 仍使用该 Runtime 的凭据，Core 每次重新检查权限；成功解析不保证将来的操作仍获准。同 owner 的不同凭据不能确认对方领取的旧回执。

## SCRM outbox 发布

保留原 `Event` 的 topic、meta、payload。新增字段 `Reliability` 仅作为传输输入，JSON/YAML 序列化忽略它，不会改变原业务信封。

```go
// 使用 SCRM 既有 decodeEvent 的结果；它已核对 env.EventUUID == row.UUID。
// env.DedupeKey 来自已持久化业务信封，不生成新的 UUID。
ev.Reliability = &event.ReliableIdentity{
    EventID: env.EventUUID,
    IdempotencyKey: env.DedupeKey,
}
err := deps.EventEmitter.Emit(ctx, ev)
// 仅 err == nil 才可登记平台接收成功；这不等于下游业务已处理。
```

也可调用 `deps.ReliableEvents.PublishEvent(ctx, event.ReliablePublish{Identity: ..., Event: ev})` 获取发布回执。请求不自动重试；超时可能已经提交，业务 outbox 使用同一事件 ID、幂等键及原信封重试。`Emit` 缺少 Reliability 明确返回 `EVENT_FABRIC_STABLE_ID_REQUIRED`。不解析 payload 中的客服字段来猜测稳定 ID。

SCRM 当前 outbox 构造 Event 时尚未设置 Reliability，升级依赖后需补上这两项字段。Framework 不修改 SCRM 业务源码或原 outbox 数据。

## 消费与事务

```go
request := eventfabric.SubscribeRequest{
    TenantUUID: tenantUUID,
    PluginID: consumerPluginID,
    SubscriberID: consumerPluginID,
    Topics: []string{fullTopic},
    BatchSize: 1,
    CompatibilityMode: "strict",
    SupportedVersions: []string{"1"},
    ExpectedPublishers: map[string][]string{fullTopic: {"com.powerx.plugins.scrm"}},
}
err := deps.ReliableEvents.RunConsumer(ctx, request, func(work context.Context, ev event.Event) error {
    // 在执行前复核当前业务权限、endpoint_uuid 和会话范围。
    // ApplyEventOnce 必须把业务变更和去重记录放在同一事务。
    // 重复事件已提交则返回 nil，不重复业务副作用。
    return business.ApplyEventOnce(work, ev)
})
```

`ExpectedPublishers` 是可信启动策略，不能来自请求参数。SDK 校验合同、租户、subscriber、物理 topic、发布插件、回执 UUID、版本、attempt 和 deadline；解包后再核对原 Meta 中的租户、发布插件、版本与业务 topic 映射。Core trace 单独保留在 Delivery，不能覆盖原 Meta.TraceID。SCRM 的 PayloadVersion `1` 保持字符串 `1`。

`RunConsumer` 串行处理，把 ACK deadline 放入业务 context。业务失败时 NACK；业务返回成功后才 ACK。它不会使用进程内去重代替业务事务。调用方还应为处理循环设置上限及退出策略，不在请求取消后继续处理。

ACK/NACK 错误可用 `errors.As(err, &settlement)` 读取 `*eventfabric.SettlementError`：保存 Receipt，Uncertain=true 表示可能已登记。不得回滚已提交业务、直接重复业务或生成新事件。`EVENT_FABRIC_RECEIPT_STALE` 停止旧回执确认，后续新投递通过业务去重后确认。NACK 成功仅代表安排重试或死信。

同租户同插件为一个竞争消费身份，多实例不是独立广播。重连使用相同 tenant/plugin/subscriber/topics；不加随机实例 ID。不支持租约续期，不能用新凭据确认旧凭据领取的回执。若需要独立订阅组或续期，先升级 Core 正式合同。

## Proto 与共享样例

Proto 原样固定在 `event-fabric-service-v1.proto`。生成的 Go DTO 和 RPC 声明位于 `runtime/eventfabric/pb`，用于合同对照；Host runtime 不选择 gRPC，也不会在 SSE 失败后切换到 gRPC。

```bash
protoc -I docs/contracts --go_out=framework/backend/go --go_opt=module=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go --go_opt=Mevent-fabric-service-v1.proto=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric/pb --go-grpc_out=framework/backend/go --go-grpc_opt=module=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go --go-grpc_opt=Mevent-fabric-service-v1.proto=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/eventfabric/pb docs/contracts/event-fabric-service-v1.proto
```

`fixtures/event-fabric-service.v1.json` 同步到 SDK testdata、CLI embed 和新生成项目。其中发布响应、ACK/NACK、完整信封、不同 trace 和错误码被真实 HTTP/SSE 客户端测试消费。fixture 不等于真实服务验收。

## 版本与验收

执行 `python3 scripts/contracts/package-event-fabric.py` 生成内容摘要固定的 SDK Go proxy、CLI 和插件样例。`artifacts/event-fabric-latest.json` 给出精确 SDK 版本、Go checksums、生成器版本、artifact hash 和升级命令。此为本机可消费开发包，不是已提交或公开发布的版本；无新提交 SHA，以 base commit + 内容摘要标识工作区快照。

消费方若已有 Framework 的 `replace`，必须先仅移除该条，再按 references 中的命令升级；`GOWORK=off` 不会取消 go.mod 内的 replace。SCRM 当前 backend/go.mod 使用 `../../PowerXPlugin/framework/backend/go`，本轮未修改它。升级后用 `GOWORK=off go list -m -json github.com/ArtisanCloud/PowerXPlugin/framework/backend/go` 检查精确版本且没有 Replace 字段。

仅 Go 及 Skeleton/CLI 本轮交付；浏览器不持有服务凭据，Nuxt 无需新增事件消费；.NET 可靠事件 SDK 未实现。

本地 HTTP/SSE、race、vet、生成项目和开发包检查与真实安装态分开记录。真实安装前仍需 Core 正式迁移、能力及 registration 发布、精确 owner/grant/topic ACL、用户重启，然后由 SCRM 验证现有 outbox → 查询消息 → 业务事务 → ACK、跨 endpoint 授权及撤权、真实渠道收发。全链路语义为至少一次，不声明任意业务副作用恰好一次。
