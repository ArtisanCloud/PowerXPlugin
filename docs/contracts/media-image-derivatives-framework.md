# Media 标准图片预览 Framework 对齐

Framework 消费 [Core Media 标准图片预览合同](../../../../PowerX/docs/contracts/media-image-derivatives.md) 与 [OpenAPI](../../../../PowerX/specs/001-media-storage/contracts/image-derivatives.openapi.yaml)。Core 负责原图复用、标准图生成和统一 TaskBus 生命周期；Framework 负责响应类型与映射；插件负责业务关联、预览读取与页面状态展示。

## 已对齐的响应

Go `runtime/powerx/media.HostAsset.Derivatives` 是可选的 `*ImageDerivatives`，`CreateAsset`、`CompleteUpload`、`GetAsset` 沿用原接口及服务凭据读取。`runtime/media.Service` 没有新增方法，local 实现不需要为此增加接口方法。

.NET `MediaAsset.Derivatives` 使用相同字段语义，`PowerXMediaClient` 的创建、上传确认和 `GetAssetAsync` 读取均保留该字段，并映射原图 `Status`；`IMediaService` 保持不变。

`ImageDerivatives` 映射 `asset_uuid`、`upload_state`、`status`、`recipe`、`job_uuid`、`attempts`、`error_code`、`retryable`、`items`。每个 item 保留 `variant`、`status` 和可选 `variant_uuid`，不由原图状态推算处理状态。

- 原图的 `status=ready` 只表示上传完成；处理状态可能仍是 `queued`、`processing`、`failed`、`unsupported` 或 `unavailable`。
- 旧 Core 未返回 `derivatives` 或返回 null 时，SDK 保持 nil/null，并在再次序列化时省略；不补造 ready 或失败状态。此兼容仅针对用户明确要求的可选响应字段。
- `derivatives` 为错误 JSON 类型时，沿用现有解码失败行为；不会被当成缺省值吞掉。
- SDK 只接收查询当时的快照，不在读取后启动轮询、生成任务或额外 ensure 请求。

## 插件接入顺序

1. 首次上传继续使用 `CreateAsset`、票据、字节上传和 `CompleteUpload`。Core 在确认真实上传后自动检查标准预览。
2. 重复上传也应进入 `CreateAsset` 的复用分支，使 Core 检查已有预览；仅调用 `GetAsset` 不会触发补图。原图已经 ready 时应复用原图，不将其重新解释为待上传。
3. 保存原图 UUID 与处理快照，按各 item 的状态使用变体 UUID。Go 继续通过 `GetVariant` 取得实际 MIME、大小，通过 `PresignVariantDownload` 获取下载票据；不能用本地编码参数覆盖 Core 返回的元数据。
4. 插件 local 的编码和 Variant 流程仍由插件维护。当前 Skeleton 的 local adapter 不承诺 Core 标准图任务语义，未提供处理快照时保持 nil。

## 异步通知边界

截至本次对齐，Core 已有带用户 JWT、tenant 和 RBAC 校验的 Admin SSE；尚无对应的插件服务态订阅合同。Admin SSE 不能直接用 STS/API Key 调用，Redis 内部频道和 TaskBus 工作主题也不作为插件公开订阅协议。

后续由 Core 明确服务态通知合同，Framework 接入已有事件机制。插件后端在权限检查后向自己的浏览器连接发出变更提示，再按现有读取接口获取状态。断线重连应重新鉴权并读回当前状态，不依赖每个中间事件都送达。完成／失败后的页面自动更新仍需三方联调，不能由本次 DTO 测试代替；不增加常驻轮询。

显式状态／重试能力 `com.corex.media.image_derivatives.read`、`com.corex.media.image_derivatives.ensure` 不属于本次响应映射范围，未扩展为新的 Service 方法，也不成为上传的强制前置调用。

## 验证和交付状态

共享样例位于 `framework/backend/go/runtime/powerx/media/testdata/host_asset_derivatives.json`，Go 与 .NET 测试同时读取，覆盖原图 ready、部分变体 ready、整体仍 processing。

- Go：创建／确认／读取三条路径的 STS 和 API Key 凭证、可选字段缺省与 null、排队／失败／不支持／依赖不可用、错误类型拒绝，以及复用变体时保留实际 MIME 与大小。
- .NET：相同三条路径和两类凭证、公开 DTO 的字段序列化、可选缺省与 null、排队／完成／失败／不支持／依赖不可用和错误类型拒绝。
- 模板：SDK 类型由依赖提供，Skeleton 的 Service 调用无需修改；检查现有 scaffold 和 CLI 内嵌模板同步状态。

命令与结果见 [2026-10-18 验证收据](evidence/media-image-derivatives-framework-20261018.json)。测试使用 HTTP fixture，不代表真实 Core 生成／补图、服务 grant、通知推送、插件接管或安装态验收。本轮未发布 SDK、修改 CLI 默认版本或重建安装 CLI。
