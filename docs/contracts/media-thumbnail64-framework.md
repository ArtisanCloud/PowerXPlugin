# Framework thumbnail_64 与通知卡片 v2 交付

本轮完成 Framework Go／Nuxt、运行时、Host Contract、公共清单和生成链路的源码对齐。权威基线为 Core `standard-image-v2`、`thumbnail_64`、`business_notification_card.v2`，来源路径与逐文件 SHA-256 见 [合同来源](media-thumbnail64-source.json)。Core 工作区基于 `0be9945cf7544bf18d0ed58be1043a313bba9de8`，本次实现尚未提交／发版，不能用基线提交代替实现提交。

| 范围 | 状态 | 实际交付 |
|---|---|---|
| Go Media | 源码与受控验证完成 | 规格级错误／retryable、按名称读取、实际变体元数据和票据、状态／ensure typed binding、服务态 SSE、API 前缀与错误诊断 |
| Go Notifications | 源码与受控验证完成 | 显式 v1／v2 引用、独立 v2 展示 DTO、presentation_changed、原 UUID 升级和 409 透传；版本回执沿用既有合同 |
| 运行时／Skeleton | 源码与受控验证完成 | 同一启动适配器的派生扩展；显式 API Key Core Host；安装态 STS provider；Host 查询与 ensure 分发；本地显式 64px 变体字节校验 |
| 清单／CLI／模板 | 源码与消费验证完成 | business_type + contract 去重；五份 Schema；打包解包及生成校验；生成器携带 v1／v2 正式 Schema 与共享样例；两套模板同步 |
| Nuxt Client | 源码与受控验证完成 | 派生 DTO／校验／命名查找、插件桥接客户端及 SSE 快照恢复、v1／v2 展示联合类型和校验、通知 revision 合并 |
| .NET | **v2 扩展未完成，后置** | 既有基础媒体 DTO 不等于本轮支持；规格级诊断、状态／订阅、卡片 v2 及对应测试未计入交付 |
| 公开发布／真实安装态 | **待完成** | 本地可消费快照已验证；公开版本、真实 Core 凭据、业务权限和部署验收仍需分别完成 |

## 媒体调用与状态语义

CreateAsset → presign/upload → CompleteUpload 及原图复用保持原接口，不在普通上传、列表、GetAsset、启动或迁移时新增 ensure 调用。自动生成不增加 ensure grant；显式查询／订阅使用 `com.corex.media.image_derivatives.read`，人工 ensure／retry 使用 `com.corex.media.image_derivatives.ensure`。Skeleton 不把这两个可选能力加入所有插件的默认必需授权。

`ImageDerivatives.Items` 保留三项以上条目（旧 v1 配方仍保留其原规格）；每项透传 variant、status、variant_uuid、error_code 和可选 retryable。`snapshot.Item(name)` 按规格名查找，原图状态、整体派生状态和单项状态分别保留。ready 项必须有有效 UUID，未 ready 项不能携带可读 UUID。整体 queued／processing／failed 不阻止消费已 ready 的 preview／thumbnail；缺少 thumbnail_64 不选择其他规格。

消费顺序为：取得当前快照 → 按 thumbnail_64 查项 → 确认 ready 与 UUID → 现有 GetVariant 取得实际 MIME／size → 现有 PresignVariantDownload 使用 parent asset_uuid + variant_uuid。`HostClient.ReadImageDerivative` 提供同样的便捷流程；运行时消费者也可通过 `Runtime.Media()` 的现有变体与票据方法完成这几步。读取不会触发补图，不拼接对象存储地址。

显式 CreateVariant 支持 thumbnail_64。Host 客户端预检 checksum、JPEG／PNG、1..32768 bytes，实际内容及边长由 Core 校验；返回 Core 已生成的 ready 变体时保留其实际 MIME／size。Skeleton 的有限内存适配器对显式 thumbnail_64 上传确认也检查实际 PNG／JPEG、完整解码及两边不超过 64；它没有实现 Core 自动派生任务。AI Craft Local 原图编码、方向、透明度、GIF 首帧等生产规则仍由 AI Craft 实现。

## 固定 binding 与运行通道

`Runtime.ImageDerivatives()` 是独立扩展，基础 `media.Service` 未增加强制方法。本地适配器未实现该扩展时明确返回 `FRAMEWORK_MEDIA_DERIVATIVES_UNAVAILABLE`，不会调用 delegated 兜底。

| SDK 方法 | 固定调用 |
|---|---|
| GetImageDerivatives | `/tenant/invocations`，read capability，core_internal，body.operation=status |
| EnsureImageDerivatives | 同一 Gateway，ensure capability，body.operation=ensure，显式 retry_failed |
| StreamImageDerivatives | 单次租约：GET `/tenant/media/assets/{asset_uuid}/derivatives/events` |
| SubscribeImageDerivatives | 同一凭据提供者重连上述服务 SSE，重新鉴权并接收最新快照 |

路径统一通过 `HostClientConfig.BaseURL` 与 `APIPrefix` 组合；默认前缀 `/api/v1`，不在资源路径重复拼接。Core 注册负责 `core://media/image-derivatives`、INVOKE、service_actor。客户端只能发送固定 capability／preferred_protocol 和 payload.body；校验响应明确使用 core_internal 且 fallback_used=false。租户不由请求参数覆盖。

普通 local 保持本地数据库／内存通道，配置 API Key 本身不切换媒体适配器。需要 API Key Host 模拟时显式配置：

```yaml
runtime:
  media:
    channel: core_host
gateway:
  base_url: http://127.0.0.1:8077
  api_prefix: /api/v1
  auth_scheme: apikey
```

后端环境变量 `PX_GATEWAY_API_KEY` 提供密钥，`PX_MEDIA_CHANNEL=core_host` 可显式选择通道。安装态继续使用启动时共享的真实 STS TokenProvider。通道缺配置或不支持时明确失败。Host 的 derivatives.status／derivatives.ensure 有真实分发入口；ensure 要求明确确认。媒体错误保留 HTTP status、error_code、reason_code、details、trace_id、request_id，并经 Host 返回。

## SSE 与 Nuxt 接入

派生 SSE 使用 status／reconnect／invalidate，按快照恢复，不使用任务 revision 游标或 Last-Event-ID 回放。心跳不触发状态 GET；重连由同一服务 SSE 的初始快照恢复。撤权 invalidate 后停止，不读取新状态，不切换凭据或通道。请求取消后释放流，迟到事件不触发回调；重连失败有界并保留上游错误。

Nuxt 的 `createImageDerivativesClient` 要求消费者显式提供已鉴权的插件桥接 paths、request 与现有 fetch-SSE stream。Framework 不猜测业务 BFF 路由。服务凭据只在后端；浏览器使用当前用户会话。首次／手动刷新可以 get，人工操作可以 ensure，常驻更新使用 watch，不添加轮询。watch 的旧租约和取消后的回调会被忽略。

Core 已负责卡片可见性加载、点击预览、身份隔离和 Blob 释放，本轮没有重建卡片组件或新增页面。

## 卡片 v2、清单与原位升级

v1 常量和单 image_key DTO 保留，v2 使用独立 `BusinessCardPresentationV2`／`BusinessCardImageV2`。thumbnail.variant 固定 thumbnail_64，preview.variant 固定 preview；两个状态独立。ready 必须有不透明 image_key，其他状态禁止携带该字段。Nuxt 使用区分 contract 的联合类型，不在 v1 image 中混入 v2 字段。

新增 reason=presentation_changed，原样通过既有 InvalidateCard 固定 binding。媒体就绪／失败或展示协议升级只改变展示版本，Core 保留 isRead/readAt；AI Craft 的业务内容版本、业务更新时间和未读次数不因派生完成增加。

原位升级由原创建服务发送 v2 card_reference、同一 source_plugin／business_type／delivery_key、更高 source_revision 和 presentation_changed，保留原通知 UUID。绑定冲突／版本冲突直接返回；不新建通知、不自动换凭据、不自动刷新后重试或确认已读。相同／较旧版本返回 changed=false 继续作为正常成功。

清单唯一键为 business_type + contract。同一业务可以分别登记 v1／v2，重复同一组合仍拒绝。路径和占位符安全约束保持原合同。通用 Skeleton 仍默认空绑定，由插件声明真实路由与权限。混合样例见 [四种业务／合同组合](fixtures/notification-presentations.v2.yaml)；测试会解包最终 tar.gz 并检查完整保留，实际生成插件也运行清单校验与后端构建。

生成插件的 docs/contracts 同时携带卡片 Schema、媒体 Schema，以及 business-notification-card.v1.json、business-notification-card.v2.json、media-image-derivatives.v2.json。

## 可消费版本与 AI Craft 升级引用

本轮仍是当前分支的未提交源码，包含前轮 AI Media／卡片 v1 工作区改动；没有包含这些修复的提交 SHA，也未公开发布。可消费的本地快照、逐组件源码哈希、产物哈希和真实生成消费结果见 `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/references.json`。

| 产物 | 已验证的本地版本 |
|---|---|
| Go SDK | `v0.0.27-dev.thumbnail64.2d0a80a518254549` |
| Nuxt Client | `0.0.14-dev.thumbnail64.1f47c36788746a87` |
| CLI／生成器 | `v0.0.0-thumbnail64.b5544696989a9112` |

AI Craft 在自己的后端目录升级时使用：

```bash
GOWORK=off \
GOPROXY="file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/go-proxy,https://proxy.golang.org,direct" \
GONOSUMDB=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go \
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.27-dev.thumbnail64.2d0a80a518254549
```

Nuxt 引用为：

```bash
npm install --save-exact "file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/artisan-cloud-plugin-framework-client-0.0.14-dev.thumbnail64.1f47c36788746a87.tgz"
```

生成器命令需要指定本轮 SDK 引用：

```bash
"/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/px-plugin" init com.powerx.plugins.thumbnail64-example \
  --directory /tmp/powerx-thumbnail64-example --git-init=false \
  --notification-presentations "/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/contracts/fixtures/notification-presentations.v2.yaml" \
  --framework-version v0.0.27-dev.thumbnail64.2d0a80a518254549 \
  --framework-client-ref "file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-thumbnail64-20261022-43252ceb268a47d2/artisan-cloud-plugin-framework-client-0.0.14-dev.thumbnail64.1f47c36788746a87.tgz"
```

这些命令供 AI Craft 后续升级使用，本轮未修改其仓库。CLI 默认的旧版 SDK 引用未宣称包含本轮功能；正式发布还需发布 SDK／CLI、同步默认引用并在无本地代理的环境验证。可用 `python3 scripts/contracts/package-media-thumbnail64.py` 按源码内容重新构建独立本地快照。

## 验收边界

Framework 受控记录见 [验收证据](evidence/media-thumbnail64-framework-validation-20261022.json)。覆盖 API Key／STS token-provider 两种传输、固定 binding、部分失败、乱序／缺规格、实际 MIME／size、64px 字节限制、SSE 重连／撤权／取消／迟到回调、v1／v2 共存、原位升级和重复／冲突、打包与生成消费。源码／受控验证完成不等于真实安装态业务验收。

Core 与 AI Craft 后续联调仍需锁定包含实现的 Core 构建，验证真实 API Key 和安装 STS exchange、能力发布／租户登记／live grant、当前成员与成本裁剪、跨租户及撤权、业务投递映射和 20／50 行真实图片下载量。Redis namespace 与历史补图由部署负责人处理；补图先 dry-run，submitted 不当作生成完成。本轮未执行 Core 补图、改 Redis、重启用户服务或修改 Core／AI Craft 仓库。
