# 通知业务导航合同

Go `runtime/powerx/notifications.CreateInput` 新增可选 `Navigation *Navigation`；.NET `CreateNotificationRequest` 在已有参数之后增加可选 `Navigation`。对象字段为 `label`、`path`。保持原 `Publisher.Create`、`INotificationPublisher.CreateAsync` 和鉴权凭据，创建路径为 `<APIPrefix>/notifications`，默认前缀为 `/api/v1`。

Core 接受同源绝对业务路径，校验后返回持久化动作 `actions`：`id=view_business`、`type=primary`、`action=navigate`、`params.path` 等于请求路径。Go／.NET 通知响应均保留动作。请求有导航却未返回匹配动作时明确失败，不能声称已完成导航接入或自动重发不确定的创建请求。省略导航不改变普通通知创建行为。

`label` 由调用方的 i18n 文件提供。Core 通知内容保持通用提示；业务目标宜由插件使用不透明入口映射，并在打开时重新核验当前租户、成员、权限、对象状态。Core 已读不等于插件业务内容已阅。

AI Craft 安装验收需要运行含该合同的 Core 后端及前端，并重新安装使用本地 Framework replace 编译的插件。旧通知不自动补动作。先确认旧聚合变化，再产生新业务变化测试跳转。SDK 未发布，本轮不打版本、不升级 CLI、不重启用户服务。

验证：Go 通知 transport / runtime race tests；.NET `NotificationRuntimeTests`；Core REST、typed invoke、持久化和成员 WS 用例；AI Craft 成员／租户定位隔离、授权失效、跨分页 UUID 定位及桌面／窄屏浏览器 fixture。真实安装联测独立验收。

## 业务已阅后的通知已读回执

新增独立 `ReadAcknowledger`／`INotificationReadAcknowledger`，不改变既有 Publisher 创建接口。Go `Client.Acknowledge` 与 .NET `AcknowledgeAsync` 通过原服务凭据调用 `POST <APIPrefix>/notifications/read-receipts`；输入为成员 UUID 和 1–100 个唯一规范通知 UUID。Go `Runtime.ReadAcknowledger()` 只从当前已选择的 publisher adapter 取得能力；适配器不支持时明确不可用，不切换 local／delegated。

宿主声明 `com.corex.notifications.read_receipts`。Core typed operation 为固定 `acknowledge_read`；本轮不扩展 Skeleton Host Contract Lab 的演示操作。Core 按实际凭据核验服务归属、租户、目标成员和通知范围；创建授权不能替代回执授权。

响应合同为 `notification_read_receipt.v1`，必须包含同一成员和完全相同的通知 ID 集合。Go／.NET 均拒绝缺失、额外、重复 ID、不同成员或错误合同。业务调用方在自己的持久化事务中限定已确认范围并保存待同步回执，不能让 Core 通知已读反向确认业务内容。

正常 Core migration 增加可信 `service_owner`，插件保存新通知 UUID 与业务范围；不追溯猜测旧通知。SDK 源码、Core 后端／前端及调用插件需一起部署验收；本地 replace 可直接用于自有测试包，SDK 尚未发布。已通过 Go race 及 .NET 通知专项测试，真实安装环境能力与铃铛仍单独验收。

## Review 对齐：地址和错误边界

遵循 [Core API 前缀共享规范](../../../../PowerX/docs/standards/_shared/api-prefix-and-versioning.md)，Gateway 地址和 Core API 前缀分别配置。通知创建和已读回执只声明相对资源路径，在同一地址构建边界拼接一次：

- Go：`notifications.Config.APIPrefix`，地址由 `runtime/common/hosthttp.APIBaseURL` 构建。Skeleton 将已加载的 `cfg.Gateway.APIPrefix` 传入；scaffold 和 CLI 内嵌模板保持相同装配。
- .NET：`PowerXNotificationClient` 新增末尾可选参数 `apiPrefix`，地址由 `Runtime.Common.CoreApiEndpoint.BaseUrl` 构建。调用方从自己的启动配置传入，不从请求参数选择前缀。
- 默认值集中为各语言共享构建器中的 `/api/v1`；显式 `/api/v2`、自定义前缀与 `/` 根前缀均支持。这只切换路由挂载地址，不代表通知业务合同升级为 v2。
- Skeleton 配置加载也保留显式 `/`，不把根前缀悄悄改回 `/api/v1`；`PX_GATEWAY_API_PREFIX → Gateway.APIPrefix → 通知客户端` 有独立回归，scaffold／CLI 内嵌模板同步。
- `BaseURL` 是 Gateway 地址，可以包含明确的反代挂载路径，例如 `https://core.example/deployment`；Core 前缀单独填写，例如 `/core/custom`，最终为 `/deployment/core/custom/notifications`。不要在 `BaseURL` 和 `APIPrefix` 重复配置同一个 Core 前缀。SDK 不猜测 URL 中的版本或自动剥离路径。
- 配置包含 query、fragment、用户凭据，或前缀含点路径、空中间段及编码分隔符时，在构造客户端时明确失败。

Go 回执错误分为三类：本地成员／通知 UUID、重复或批量限制错误使用 HTTP 400 与 `NOTIFICATION_READ_RECEIPT_INVALID`；Core 返回的错误成员、版本或通知集合使用 HTTP 502 与 `NOTIFICATION_READ_RECEIPT_RESPONSE_INVALID`；nil／零值客户端和缺失依赖使用 HTTP 503 与 `NOTIFICATION_CLIENT_UNAVAILABLE`，不触发 panic。上游 HTTP 错误继续保留状态和原因码，取消继续返回 context 取消错误。.NET 保持对应的 `InvalidArgument`、`InvalidResponse` 等稳定错误分类。

服务归属按 Core 当前合同隔离：STS 使用插件 ID，API Key 使用 Key UUID。Key UUID 轮换或 API Key／STS 切换不自动继承旧归属；旧通知使用普通 Core 已读操作处理，不从 metadata 推断或回填。客户端不改变这一规则。

定向验证覆盖默认／v2／自定义／根前缀、反代挂载路径、创建与回执共享前缀、非法配置提前拒绝、本地参数零 HTTP 请求、无配置客户端、响应合同错误、凭据故障和取消。源码验证与真实服务 grant、Core 前端 session／分页／重连修复及安装验收分开记录；CLI 构建安装和 SDK 发布仍暂缓。

本次命令与结果见 [Framework review 对齐收据](evidence/notification-sdk-review-alignment-20261018.json)。路径矩阵验证的是通知 SDK 地址构建，不代替其他客户端的全局前缀迁移或安装环境验收。

Core 对应的 session、分页、重连与配置注入修正，以及后续 Skeleton 根前缀补充验证，见 [Core review 回归收据](../../../../PowerX/docs/contracts/evidence/notification-core-review-alignment-20261018.json)。本轮没有代替插件修改自有启动装配、配置归一化或安装包。
