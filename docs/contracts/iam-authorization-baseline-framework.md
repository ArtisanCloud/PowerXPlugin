# IAM 连续授权基线 Framework 接入

本文件记录 Framework F01–F04 的消费入口与验证边界。业务语义、初始化、回滚和 B01–B24 验收以 [Core 实施指导 1.2](../../../../PowerX/docs/guides/features/iam-authorization-baseline/guide.md) 为唯一权威正文；独立检出时查阅 PowerX 仓库 `docs/guides/features/iam-authorization-baseline/guide.md`。

## SDK 合同与调用

Go：`iam/contracts.AuthorizationBaselineService.ReadMemberBaselines`；从 `iam/adapters.Registry.AuthorizationBaselines()` 获取。普通 `AuthzService` 的允许／拒绝合同保持独立。

查询包含 `TenantUUID`、`MemberUUIDs`、`PermissionCodes`。`TenantUUID` 是消费者从服务端可信上下文取得的校验断言，标记为 `json:"-"`，不传给 Core。Core 继续从已验证服务凭据推导租户及插件；客户端按本地配置的插件身份和该租户断言校验响应。成员 UUID 规范化后检测重复；一次最多 100 成员、10 权限，不自动拆批、不无限枚举。

`AuthorizationBaselineResult.Items` 为完整的成员／权限组合。`Status=error` 的 `Allowed` 必须为 nil；消费者不得将它视为 false。`Status=decided` 必须有明确布尔和修订。`BaselineTime` 严格接受 UTC 六位微秒字符串，拒绝数字时间、毫秒、时区偏移和精度损失；初始化允许 `GrantedAt=nil`，消费独立 `VisibleSince`。

Framework 拒绝未知／重复 JSON 字段、遗漏／重复／额外组合、错误分支夹带授权字段、schema／身份错配、无效范围、缺失时间／修订及不一致时间边界。完整响应内的合法逐项错误正常返回，畸形响应整批失败。

`iam/delegated.CoreClient` 复用现有 STS `TokenProvider` 或后端 API Key，只调用固定 `core_internal` capability／operation。无浏览器 token、自由 URL 或本地 IAM 回退。`AuthorizationBaselineRequestError` 保留 Core `Status`、稳定码、Trace／Request ID；上游 HTTP 200 但合同无效时保留 `Status=200`，对下游映射 502。取消、超时及响应体大小有界；禁止跨请求缓存。

.NET 对应 `IAuthorizationBaselineService`、`AuthorizationBaselineQuery`、`AuthorizationBaselineResult` 和 `PowerXIamClient.ReadMemberBaselines`。`IAMAdapterBundle` 可绑定可选 provider 与可信 `PluginId`，经 `GetAuthorizationBaselines()` 获取。两种语言共享同一 JSON fixture，检查 nullable 分支、微秒精度及严格集合语义。

## 启动装配与启用

1. delegated 沿用启动期配置的 Framework CoreClient，由 AuthProxy 暴露类型化 provider，进入 Bundle、Registry 和 `Deps.IAMAuthorizationBaselines`。没有启用本功能的既有 IAM 使用不强制绑定该 provider。
2. local 由插件在绑定前设置 `Deps.LocalAuthorizationBaselines`。Framework 使用验证包装器检查其结果，不实现历史存储、不提供默认放行或空结果 adapter。
3. 消费者从自己的可信、持久化功能状态设置 `Deps.IAMAuthorizationBaselineEnabled`；Skeleton 同时将 manifest 中明确 required 基线能力视为启用声明。在路由／消费者创建前调用 `ValidateAuthorizationBaselineFeature`，显式启用但缺少 required 声明必须失败。
4. local 启用要求自身 provider 完整绑定。delegated 启用要求 manifest 同时 required：`com.corex.iam.authorization_baseline.service_read`、`com.corex.iam.directory.read`、`com.corex.iam.identity_context.service_read`、`com.corex.capabilities.grant_status.read`，并通过现有 `RequireGrants` 实时检查。启动检查不代替每次调用的 Core grant 校验。
5. Skeleton 默认没有生产 local 基线，不默认启用本功能；接入业务插件时必须联动其功能状态、local v2 provider 和 required 声明。.NET 消费者通过 `RequireAuthorizationBaselineFeatureAsync(enabled, required, grants)` 执行同等门禁。

错误使用稳定机器码，例如 `IAM_ADAPTER_NOT_BOUND`、`IAM_BASELINE_REQUIREMENTS_INVALID`、`IAM_BASELINE_RESPONSE_INVALID`；用户可见文案由消费页面通过 locale 映射。服务授权失败不触发用户反复登录。

## WS 最终发送与模板

Skeleton 业务下行在取得 `sendMu` 后调用当前 lease 的 `AuthorizeTopic`，复核成功后再执行 `CheckActive`，最后才 `WriteJSON`。订阅时校验继续保留；事件回调不使用早先的判定替代末端检查。撤权、依赖错误、到期、取消或迟到结果禁止写帧，关闭现有 lease／连接并释放订阅。该保证针对最后判定点可见的事实，不声明分布式原子写入。

现有 `scripts/template-sync-config.yaml` 的 backend 映射覆盖新增 bootstrap、Deps、AuthProxy 及 handler 文件，生产代码同步至 scaffold 与 CLI 内嵌模板。测试不作为业务源码模板发布；生成验证将同一批装配与最终发送测试注入测试生成物，验证生成代码而非修改生成实现。

## 验证与待办边界

本轮记录见 [2026-10-15 验证收据](evidence/iam-authorization-baseline-framework-20261015.json)。收据中的 fixture／本地快照结果与真实 Core、已发布 SDK、安装环境分别记录。

backend 映射核验 998 个模板目标均一致。全仓 `sync:templates --check` 最终复核通过；中途出现的四个 Nuxt locale 模板差异已在最终复核时消失，收据保留该检查过程。

- Framework Go：合同、客户端、可选 Registry、local 结果验证、错误保真、取消／超时及 lifecycle 的定向 race 测试。
- Skeleton：真实 AuthProxy → Bundle → Registry → Framework HTTP 客户端路径；关闭、缺 provider、缺 required、缺 grant 和成功启用；真实 WebSocket `send → WriteJSON` 排队、撤权、503、取消、到期与迟到结果。
- .NET：共享 JSON fixture、typed 服务凭据、Registry／Factory、启用门禁及错误保真。
- 生成物：`PX_TEST_GENERATED_BUILD=1 go test ./tools/cli/internal/templates -run '^TestIAMBaselineGeneratedPlugin$' -count=1 -timeout=10m`。测试构建实际 CLI、生成新插件，使用不可变的本地打包 Framework module 快照运行编译及 race 测试；不使用 local replace，不代表已发布 SDK 或安装态验收。现有 backend CI 已包含该测试。
- 同步与 CLI：`npm run sync:templates -- --check`、`make build-px-plugin`。

插件 v2、阅读确认、重连不推进订单游标、delegated worker 与 B01–B24 的真实安装联调属于下一阶段。特别是 B20、B23 的业务结果不能由通用 WS 或 SDK fixture 代替。本次不执行服务 grant、迁移／初始化、部署或用户日常服务重启；不把 Core 文档中的历史授权状态当作本轮实时验证结果。
