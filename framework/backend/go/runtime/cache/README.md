# Framework Cache 服务适配

`Runtime` 在启动时固定选择一份 Local 或 delegated `Service`。缓存错误必须返回 error，不能转换成 miss，也不选择其他适配器。

`NewCapabilityProvider(invoker)` 消费固定类型 `cache.service.v1`，适用于显式 API Key Host 模拟和安装 STS。调用只包含 `body`，由适配器生成 contract_version、tenant_key、operation、namespace、key，以及 set 专用的 value_base64／ttl_ms；调用方不能指定任意 endpoint、HTTP method、headers 或服务主体。

固定能力为 `com.corex.runtime.cache.service_read`（get）和 `com.corex.runtime.cache.service_manage`（set／delete），首选协议为 core_internal、端点由正式合同固定为 `core://runtime/cache`。Invoker 必须使用启动时选择的凭据及受信任租户；tenant_key 仅用于和 Core 已鉴权租户做一致性校验，不是授权来源。

响应必须为 completed、core_internal、fallback_used=false，并返回相同版本、operation、namespace 和 key。Get 验证显式 found、canonical Base64 和 expiry；miss 与空值命中不同。错误沿既有 Gateway 类型传播，拒绝／失效／超时不会触发其他凭据或本地缓存。

现有 `NewHostProvider(hostapi.Config, ...)` 继续执行其 STS 专用 REST Cache 合同；它不是 API Key 后备通道。应用显式选择其中一种正式适配，不能在调用失败后改构造另一种。

共同限制：可信 canonical UUID 租户、namespace 128 字节、key 512 字节、值 1 MiB、TTL 1 毫秒～24 小时且为整毫秒。Local 存储及资源关闭由应用提供。TTL 不能证明离线许可或延长授权事实。

`runtime_test.go / capability_test.go` 覆盖模式隔离、缺适配、取消、三操作固定 DTO、依赖错误和错误响应。实际 API Key／STS、grant、环境／服务隔离仍需在目标 Core 上核对。
