# 本地启动误选 Host 的历史诊断与更正

更正：2026-11-13。本页保留 2026-11-12 的历史拒绝与 Host 接入准备说明。此前把 `POWERX_PROXY=1` 当作 Host 事件选择条件是 Framework 装配错误，不能据此要求本地开发修改 Core 授权。现行规则为 local 使用 Redis、delegated 使用 Host；代理开关不改变事件运行时。[模式判定](../guides/develop/standalone/mode-resolution.md)与[本地启动修复验证](evidence/local-provider-event-startup-20261113.json)为当前依据。

历史日期：2026-11-12。Framework 当前分支源码修正；没有发布 SDK/CLI，没有更改 Core 源码、Key、主题、ACL、数据库或日常服务。

`.env` 的 `POWERX_PROVIDER_MODE=local`、`POWERX_PROXY=1` 保持原值。当时业务使用本地 provider，但事件被错误选到 Core；事件配置不再设置另一组 mode/provider/fallback。此前缺声明、缺凭据租户及声明未进入安装包的问题已在 Framework 修正。

## 当时确认的真实拒绝

用当前开发服务 Key 的正式只读查询确认：

| 检查 | 结果 |
|---|---|
| 发布、消费及 grant-status 服务能力 | `granted`；这不等于插件 owner 或主题 ACL |
| `GET /api/v1/tenant/iam/tenant` | 成功取得该凭据真实租户；不使用浏览器用户身份 |
| 当时配置与装配 | `local + proxy=1`；从包内声明读取 5 个发布 key，无伪造物理映射 |
| `POST /api/v1/tenant/invocations`，`service_publish/resolve_topics` | HTTP 502，内层 `EVENT_FABRIC_OWNER_FORBIDDEN` |
| 调试路由别名 `com.powerx.plugins.base.local` | 同样被 owner 校验拒绝；不能作为绕过身份检查的替代 |

当时最后一次只读装配请求 UUID：`60e2a64d-cf2b-44c0-abbf-90e55b2eb0f3`。完整脱敏结果与测试见 [验证记录](evidence/event-fabric-startup-unblock-20261112.json)。没有发送业务事件、安装插件或执行资源写入；该轮没有验证本地启动成功，现行验证见文首更正记录。

Core 的 `CredentialAccess.OwnsPlugin` 来自已认证 STS 或 Key 的显式 owner 绑定；能力 grant 不自动赋予 owner。Framework 保留真实拒绝，不切换为 Admin、WSBus、本地队列或其他租户。

## 准备好的资源范围

源码声明：[event_fabric_service.yaml](../../skeleton/config/event_fabric_service.yaml)。仅发布，principal 均为 `com.powerx.plugins.base`；Core 使用凭据真实租户生成 FullTopic。

| 业务 key | 自有 namespace | name | ACL |
|---|---|---|---|
| `_topic.template.update` | `com.powerx.plugins.base.template` | `update` | 本插件 publish |
| `_topic.audit.template.updated` | `com.powerx.plugins.base.audit.template` | `updated` | 本插件 publish |
| `_topic.template.validate.completed` | `com.powerx.plugins.base.template.validate` | `completed` | 本插件 publish |
| `_topic.template.batch_clone.completed` | `com.powerx.plugins.base.template.batch_clone` | `completed` | 本插件 publish |
| `_topic.template.update.completed` | `com.powerx.plugins.base.template.update` | `completed` | 本插件 publish |

`plugin.d/capabilities.yaml` 已声明 `com.corex.eventfabric.service_publish` 和既有 `com.corex.iam.directory.read`。这份示例不消费可靠事件，不额外要求 `service_consume`。旧 WS 配置及其能力服务独立功能，不导入其共享主题或成员 ACL。

## 仅显式 Host 接入时的管理边界

本节不作为 local 或 local + proxy 的启动要求，本轮不执行，也不表示已获授权。只有另行确认的 Host 接入任务才适用。以下步骤会改变本机 Core 授权与资源，需先获得用户对跨仓库／实际环境操作的明确确认，再使用 Core 正式管理员入口执行：

1. 为本插件创建专用开发服务 Key，owner 精确限定 `com.powerx.plugins.base`；本次事件链路最小授权为 `com.corex.iam.directory.read`、`com.corex.eventfabric.service_publish`。不放宽鉴权，不修改当前共享 Key 的 owner 或其他插件权限；如需 grant-status 诊断，单独声明并审核该只读能力。若此 Key 将替换整个 Gateway 凭据，必须先核对其他已启用模块的既有批准能力，不能只用这两项覆盖原权限，也不能顺带扩权。
2. 核对该插件在凭据真实租户的正式实例和声明，按[安装与租户准备合同](plugin-event-fabric-install.md)事务准备上述 5 个主题、发布 ACL 和绑定。不得直接写库拼造 `auth.credentials` 或 FullTopic。
3. 当前 Core debug-host 注册仅挂载 `.local` 路由并启用虚拟实例，不携带包内 topic manifest；普通 `resolve_topics`、重启也不准备资源。不能承诺只换 Key 或对未安装插件调用 resync 就能启动。如果没有正式实例，需先由 Core 明确本地开发资源准备路径；如涉及 Core 新接口／源码或重启，先确认具体改动，日常服务由用户操作。
4. 专用 Key 仅写入用户批准的本机私密配置／凭据存储，不写文档、提交或聊天。用同一身份重跑 IAM → resolve_topics，核对全部 5 个 key，最后由用户启动日常插件验证。若需临时服务，记录 PID、端口并在结束时清理。

如果后续在 registration、实例或 ACL 处被拒绝，保留新错误与请求 UUID，使用对应正式管理操作修复，不把已读到映射当作后续业务发布已获授权。撤销本轮新增专用 Key／资源也走 Core 管理流程；不删除旧申请、业务数据或其他插件资源。

## Framework 验证与边界

覆盖声明路径／符号链接／大小／多文档、凭据租户错误与取消、显式租户、禁止混合静态映射与业务 key、API Key 不切换 STS、发布中心快照和冻结包保留声明而排除私有配置，并校验模板和独立生成插件。受控构建检查不代表原生 Linux/Nuxt 包或 Core 安装验收。旧的模式统一记录保留为历史证据。
