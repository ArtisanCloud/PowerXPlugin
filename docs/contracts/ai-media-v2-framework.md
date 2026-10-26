# Framework AI 媒体 v2 对齐记录

2026-10-20，适用 Go Framework、Nuxt Framework Client 和 Gin/Nuxt Skeleton。Core 权威职责和协议见 [v2 契约快照](ai-media-v2.md)。[来源记录](ai-media-v2-source.json)固定 Core 工作区文件的 SHA-256；其中 source_commit 仅标识当时 HEAD，相关文件尚未提交，文件摘要才是本次精确基线。

## 调用边界

- `runtime/powerx/ai.Client` 的 `ListMediaModels/ListMediaProviders` 读取媒体目录，`VLMInvoke` 返回 `AnalysisResult`，`ImageInvoke/VideoInvoke/TTSInvoke` 返回 `MediaTask`；`GetMediaTask/CancelMediaTask/SubscribeMediaTask` 处理任务生命周期。
- JSON 操作统一调用 `/api/v1/tenant/invocations`，固定 `preferred_protocol=core_internal`，payload 只有 typed body；生成 body 自动携带 `contract_version=2`。媒体 SSE 使用正式 `/api/v1/ai/media/tasks/{task_uuid}/events`。
- `MediaInvokeInput.IdempotencyKey` 必须由插件业务尝试记录持久保存。同一尝试重试保持请求参数和键不变；SDK 不生成键、不重提生成、不访问供应商 URL。`AI_SUBMISSION_OUTCOME_UNKNOWN` 由业务明确展示和人工核查。
- 任务使用 Core `task_uuid`，成果使用 `media_asset_uuid`。下载凭证通过现有媒体模块获取；不能持久使用供应商 URL。`cancel_status=requested/unsupported` 不表示任务已取消。
- `runtime/ai.GenerativeService` 在启动时绑定一种模式。Skeleton local Ollama 可返回类型化 VLM 分析；尚未实现的媒体目录、生成任务和订阅明确 unsupported。设置页的显式 PowerX 目录来源使用独立 Host 客户端，不能改变业务生成通道。
- `runtime/aisettings.CatalogModels/CatalogProviders` 读取媒体 v2。文本设置目录改用独立的 `SettingsCatalogProviders/SettingsCatalogModels`，不再用通用 `ListLLMModels` 反推 Provider；所需 Core 新操作、补丁与尚未完成的运行实例验收见 [Settings 目录 v1 对接](ai-settings-llm-catalog.v1.md)。Embedding/ASR/3D/Rerank 不假装具备媒体 v2 目录。不保留旧 Admin REST 代理 payload 或原始 `ModalInvokeOutput.Data`。

## 插件与前端消费

Skeleton 的 `/admin/ai-settings/media` 暴露 models/providers、四种 invoke，以及 `tasks/{task_uuid}`、`/cancel`、`/events`。这些接口复用已有插件认证中间件，只调用启动绑定的 Service；拒绝请求选择环境。

Nuxt 使用 `useAIMediaApi()`，Framework 客户端入口为 `createAIMediaClient()`。设置页按 operation Schema 展示枚举、布尔、数值和文本参数，并在分区切换时保留草稿；窄屏使用横向模态导航。`listModels()` 保留 operation Schema、defaults 和 available；`mediaOperationDefaults(model, capability)` 只读取显式 operation，保留 0/false。

业务流程是：先持久化生成尝试与幂等键 → 调用 `generate()` → 持久保存 Core 任务 UUID → 使用 `watchTask()` 更新状态 → 成功后通过媒体服务读取成果。刷新后只读取原任务并重新订阅。分析使用 `analyze()`，不创建媒体任务。

订阅要求调用方传入 AbortSignal 和 onTask。离页、主动关闭和身份/租户切换会释放读取；身份切换后旧任务不能继续写入当前状态。SSE 仅在 header 发送用户鉴权和 `Last-Event-ID`，保留 event id，按 revision 去重；初始状态直接来自流内完整快照。不写入 token、不增加详情轮询。Core STS/API Key 只保留在插件后端。

Go 订阅负责重新取得服务凭证；Nuxt 订阅每次使用当前插件会话。正常长连接续期与断线均同步快照；连续失败有上限退避。401/403 和撤权事件终止；瞬时 429/502/503/504 的重连失败保留最终 HTTP 状态、详情及 trace。心跳不查询任务。

HTTP 错误保留 `error_code/reason_code/details/trace_id/request_id`；任务快照保留 `error_code/provider_error_code/upstream_status/provider_trace_id`。这些机器字段由业务 locale 映射，原始供应商正文仅供诊断。

## 验证与交付边界

共享样例同时保存在仓库 contracts 与 Go SDK testdata，`npm run check:ai-media-v2-contract` 校验摘要和副本一致性。Go 受控 HTTP 测试覆盖 API Key 和 STS 请求、目录、四类调用、同键请求、查询/取消、结构化错误和 SSE 恢复；Gin 测试覆盖插件到 Core 的 HTTP 链路及 local 不切换通道。Nuxt 测试覆盖 header 鉴权、游标、去重、撤权、离页取消和重连上限。

这不代表真实 Core grant-status、安装态撤权/跨租户、付费供应商调用或成果播放已验收。.NET 媒体 v2 后置。SDK 源码及模板同步不等于包已发布；生成物使用当前 Framework 模块快照验证，正式发布前需发布匹配的 Go/Nuxt SDK 并更新生成器默认版本。

最终命令及结果见 [本轮受控验证记录](evidence/ai-media-v2-framework-validation-20261020.json)。
