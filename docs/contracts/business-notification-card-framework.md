# Framework 业务通知卡片对齐记录

> 本文保留此前对齐记录。2026-10-22 的派生 v2、服务态 SSE、卡片 v2 及当前升级引用见 [本轮交付](media-thumbnail64-framework.md)。此前本地快照不作为本轮功能版本。

本轮实现 `business_notification_card.v1` 的 Go 通用合同、运行时、Host Contract 与公共清单链路，供 AI Craft 消费。Core 权威合同与样例已原样快照到本目录，来源及 SHA-256 见 [来源记录](business-notification-card-source.json)。Core 来源基线为 `0be9945cf7544bf18d0ed58be1043a313bba9de8`，合同来自其未提交工作区，联调必须使用包含该实现的 Core 构建。

| 范围 | 本轮结果 | 验收边界 |
|---|---|---|
| Go SDK | 卡片引用、展示 DTO、创建版本、InvalidateCard、旧通知绑定、版本回执、上游错误诊断 | 共享样例和受控 HTTP transport 通过 |
| Go runtime / Skeleton | 固定启动适配器、显式 API Key Core Host、真实 STS TokenProvider、Host operation 分发及 Navigation / Metadata 透传 | 本地卡片能力明确不可用；未执行真实安装态投递 |
| 公共清单 / CLI | 模型、五份实际 Schema、路径和重复校验、打包后回读、注册与管理端导出、生成器参数、两套模板同步 | tar.gz 解包检查两种绑定；生成插件校验和构建通过 |
| Nuxt Client | 引用、展示、通知、事件及回执类型；按 Core revision 合并同 ID 通知 | 重复和旧事件不增条目、不覆盖新版本；未新增卡片 UI 或连接方案 |
| .NET | **未完成，后置** | 缺少卡片接口、版本字段与回执映射、完整冲突诊断、展示绑定模型及对应测试；不计入全语言交付 |

## 调用与运行通道

`Publisher.Create` 保持原接口；可用 `CreateInput.WithBusinessCard` 复制并扩展 metadata，不覆盖调用方字段。卡片创建要求成员 UUID，校验返回的引用、其他 metadata、Navigation、通知 UUID、成员、版本及正式响应必填字段。`metadata.business_card.source_revision` 为初始绑定值；当前水位读取通知顶层 `sourceRevision`。

`runtime.CardInvalidator()` 与 `runtime.ReadAcknowledger()` 解析同一个启动适配器。`InvalidateCard` 仅调用配置 API 前缀下的 `/tenant/invocations`，固定 capability `com.corex.notifications.invalidate_card`、`preferred_protocol: core_internal`、`payload.body.operation: invalidate_card`，不接受调用方覆盖 endpoint、method、actor、租户或协议。Core 注册负责 `core://notifications`、`INVOKE`、`service_actor`。响应必须明确 `protocol_used: core_internal` 和 `fallback_used: false`。

更新和绑定使用原 UUID，重试保留相同 `source_revision` 与 reason。`changed: false` 是正常成功，返回的当前 `source_revision` 可以大于请求版本。`content_changed`、省略 reason、`access_changed` 均按合同原样交给 Core。绑定冲突、所有者不匹配和版本冲突直接返回，不创建新通知、不交换服务凭据、不刷新后自动确认已读。

阅读回执验证成员、通知 ID 集合和完全相同的 revisions 映射；版本键只能来自请求 ID 集合。批量请求完整提交一次，409 原样返回。该操作只确认 Core 通知已读，订单内容已阅由 AI Craft 维护。

默认 local 使用插件数据库通知；仅配置 API Key 不会替换它。显式联调 Core 通知时配置：

```yaml
runtime:
  notifications:
    channel: core_host
gateway:
  base_url: http://127.0.0.1:8077
  api_prefix: /api/v1
  auth_scheme: apikey
```

密钥通过后端环境变量 `PX_GATEWAY_API_KEY` 提供；通道也可通过 `PX_NOTIFICATION_CHANNEL=core_host` 明确选择。缺少 Core 配置、凭据或选择了未知通道会拒绝启动。安装态保持 delegated，通过已有真实 STS exchange 的 TokenProvider 获取服务 token；不能使用仅配置 STSClientID 等字段的占位客户端，后者明确返回 `NOTIFICATION_STS_TOKEN_PROVIDER_REQUIRED`。

纯本地数据库适配器拒绝 business_card 创建，未实现的更新／回执明确返回能力不可用。普通本地通知仍可创建并保留 Navigation。Host 的 `notification.create`、`notification.invalidate_card`、`notification.acknowledge_read` 都有正式目录、字段校验与分发入口。通知错误保留 HTTP status、机器码、reason、details、trace_id、request_id；Host 不再把 409 压成通用 upstream。

## 清单、权限与 AI Craft 边界

[展示绑定样例](fixtures/notification-presentations.v1.yaml)包含 sample_track 与 bulk_order。通用 Skeleton 默认为空列表，AI Craft 必须按自己的真实路由登记。`px-plugin init --notification-presentations <yaml>` 接收包含顶层 notification_presentations 的 YAML；非法绑定在生成前失败。打包复制原始 plugin.yaml 后再次校验，测试同时解包最终 tar.gz 检查绑定。

路径相对插件 `endpoints.http_base_path`，必须以 `/admin/` 开头。共享校验拒绝重复业务类型、重复／未知占位符、外部 URL、query、fragment、穿越、编码、反斜杠与异常分隔。展示绑定不授予读取权限，插件仍需显式维护 API 路由与 RBAC。

AI Craft 只声明实际使用的服务能力：`com.corex.notifications.create`、`com.corex.notifications.invalidate_card`、`com.corex.notifications.read_receipts`。API Key 更新授权要求 `_scope.notifications.invalidate_card`、action `invalidate_card`、resource type `capability`、pattern `notifications`，并满足能力发布、租户登记、实时 grant。Admin 权限不能代替服务授权。

AI Craft 负责展示接口、当前用户图片权限、成本裁剪、成员可见版本、业务阅读游标与可靠投递。浏览器展示／预览使用当前用户身份；服务凭据只在后端。Core 负责铃铛、列表、详情及预览；本轮未添加新的通知 UI、SSE 或轮询。

## 本地可消费快照与准确升级引用

本轮改动尚未提交或公开发布，**没有包含本轮修复的提交 SHA**。Framework 工作区基线为 `f123a2310568bd0e3625e825b4fa419b9cfb4f6c`，不能把它当作修复提交。可消费产物是当前工作区的本地开发快照，包含既有 AI Media 工作区改动；精确文件哈希与实际生成消费结果见 `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/references.json`。

| 产物 | 本地快照版本 |
|---|---|
| Go SDK | `v0.0.26-dev.card.f5edb08989906979` |
| Nuxt Client | `0.0.14-dev.card.b47c4d7571973c64` |
| CLI / 生成器 | `v0.0.0-business-card.f5edb08989906979` |

AI Craft 后端可使用该本地模块代理升级；在 AI Craft 后端目录执行以下命令会修改其 go.mod / go.sum，本轮仅验证了隔离生成插件，未修改 AI Craft：

```bash
GOWORK=off \
GOPROXY="file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/go-proxy,https://proxy.golang.org,direct" \
GONOSUMDB=github.com/ArtisanCloud/PowerXPlugin/framework/backend/go \
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.26-dev.card.f5edb08989906979
```

Nuxt 在其项目目录引用已打包的客户端：

```bash
npm install --save-exact "file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/artisan-cloud-plugin-framework-client-0.0.14-dev.card.b47c4d7571973c64.tgz"
```

使用本轮生成器必须明确指定对应 SDK 引用：

```bash
"/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/px-plugin" init com.powerx.plugins.card-example \
  --directory /tmp/powerx-card-example --git-init=false \
  --notification-presentations "/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/contracts/fixtures/notification-presentations.v1.yaml" \
  --framework-version v0.0.26-dev.card.f5edb08989906979 \
  --framework-client-ref "file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-business-card-20261021-f5edb08989906979/artisan-cloud-plugin-framework-client-0.0.14-dev.card.b47c4d7571973c64.tgz"
```

CLI 默认的 Go `v0.0.26` / Client `0.0.13` 未作为包含本轮实现的发布版本验收；正式交付需发布 SDK／CLI 并同步默认引用。本地代理与 tarball 不能作为其他机器可下载的公开版本。

## 验证与剩余安装态验收

[验证记录](evidence/business-notification-card-framework-validation-20261021.json)区分单元／受控 transport、打包解包、生成消费与安装态。验证覆盖两种凭据头、固定 binding、相同版本重试／乱序响应、绑定透传和冲突、access_changed、版本回执与批量 409、响应不完整、错误诊断及不支持适配器。Go race、vet、Nuxt 单元与 TypeScript 检查通过；模板同步、共享合同快照和仓库合同检查通过。

真实安装态仍需双方联调：使用包含当前卡片实现的 Core 构建；分别验证 API Key 与安装态 STS 的发布／登记／grant；验证原创建者迁移与跨所有者拒绝、绑定不变性和重放、Core 更新／读态效果、原子批量冲突、真实用户展示和图片权限。源码测试不能替代这些业务验收。
