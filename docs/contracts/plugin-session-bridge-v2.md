# Framework 前端桥接 v2 交接

日期：2026-10-10。前端 client 保持原版本声明 `0.0.11`，本轮仅以本地源码联调，不准备或发布新版本；Go／.NET 后端版本与 STS／API Key／capability 合同不在本轮修改范围。

提供方：[Core 合同](../../../../PowerX/docs/contracts/plugin-session-bridge-v2.md)。接入与逐项验收：[Framework 指导](../guides/features/plugin-session-bridge-v2/guide.md)。

## 共享实现

- iframe 自己的 `window.PowerXBridgeClient` v2 管理认证消费确认；Framework 不建立第二个认证消息接收器。外观同步校验父窗口、精确 origin、插件和实例。
- `PowerXHostSession.apply` 保存候选 Token，读取真实 context，验证并提交后发布 ready。每次异步结果受实例、会话代际、revision/sequence 保护；旧失败不能清新会话。
- `validateContext` 必须返回 `PowerXHostIdentity`。原始 `snapshot.tenantUuid` 不修改；`snapshot.identity` 保存验证后的 context 租户与成员。Root 默认租户来自真实 Core context，普通用户须匹配宿主租户与成员。
- `ensureSnapshot` 合并等待，默认 8s 明确超时；不需要 refreshToken。普通请求在 auth-cleared 后立即失败；显式恢复是一个新的用户操作。
- `api-fetch` 与 `usePluginApi` 都使用共享会话。仅 401 且结构化机器码明确为用户访问令牌过期（`AUTH_TOKEN_EXPIRED`／`TOKEN_EXPIRED`）时，等待真正验证后最多重试一次；403 不刷新。票据、服务凭据、来源不明的 `IAM_UNAUTHORIZED` 与未分类 401 不请求宿主续期。旧成功、旧失败与旧 SSE 事件均不能回写新会话。
- `createPowerXRouterNavigator` 按 Nuxt app.baseURL 去前缀，调用实际 Router，保留 query/hash 并尊重 AbortSignal。Core 负责实际地址确认。
- Skeleton 统一从客户端初始化接入；卸载取消认证与 Router 适配器，关闭连接和在途请求，处置插件自己的缓存，不批量删除宿主存储。

## 外部插件接入

仅升级依赖不能替换 AI Craft／CRM 已复制的 useAuth、api client 或 bridge adapter。应参照 Skeleton 的 `useEmbeddedAuth.ts`、`useHostBridgeAdapter.ts`、`auth.client.ts` 和 `host-context.ts` 对齐，随后重新构建安装包。

新接口不接旧认证别名或旧协议。`applyPowerXHostAuthToken/normalizePowerXAuthTokenPayload` 已退出公开入口；通用原生客户端只使用 tenantUuid，不再使用 numeric tenantId。standalone 的本地登录与续期仍由插件自己的实现管理。

SDK npm 包新增 `navigation.ts`，导出共享会话、身份、桥接与 Router 类型；Skeleton 与 scaffold／CLI 内嵌模板同步保留本地依赖参数，不引用未发布版本。

## 验证状态

验收记录：[脱敏命令与本地联调记录](evidence/plugin-session-framework-v2-20261010.json)。当前前端 19 个文件、168 项测试通过；完整 client 严格 TypeScript 检查、模板同步及 CLI 生成物测试通过。桥接 v2 的此前 Nuxt 生产构建通过；本轮身份错误分类调整未重复生产构建。CLI 标准命令检查遇到原有 pgx go.sum 缺失；此前使用临时 modfile 后命令测试通过，没有修改 CLI 的 go.mod/go.sum。Core 注入脚本夹具复制自 `web-admin/public/powerx-bridge-client.js`，SHA256 记录在 `skeleton/web-admin/nuxt/tests/fixtures/core-bridge-v2.provenance.json`。

真实浏览器按用户要求留待后续共同联调：CUA fetch 失败，Chrome 工具 profile 被占用。未停止用户的浏览器或日常服务，本轮未启动长期服务。当前源码不能因此宣称首次刷新、多标签、资源和实际历史导航均已通过。

## 本地依赖与发布边界

- Go Skeleton 显式使用 `replace github.com/ArtisanCloud/PowerXPlugin/framework/backend/go => ../../../framework/backend/go`；即使 `GOWORK=off`，仍消费本地 Go 源码。
- Skeleton Nuxt 使用 `file:../../../framework/frontend/nuxt/framework-client`，node_modules 指向本仓库 client；`0.0.11` 只是保留的原版本声明，不代表这些新功能已发布。
- 生成模板保留 `FrameworkClientRef` 参数，CLI 正式版本默认仍为 `0.0.11`；本地联调须提供 file 引用。Skeleton 的本地 Go replace 不写入通用生成模板。
- 已撤回提前准备的 `0.0.12` 版本与临时 tarball；没有发布 npm 或打发布 Tag。
- 先用本地依赖完成真实联调，再单独确认发布范围和版本号。

## 后端身份票据协作边界

**2026-10-11 更新：Core 已交付正式合同，Framework Go/.NET、Skeleton 与模板已完成源码接入。当前接口、错误码、WS 规则和剩余验收以 [正式接入说明](plugin-request-identity-framework.md) 为准；下面保留上一轮草案评审记录。**

参考 [Core 身份票据交接草案](../../../../PowerX/docs/contracts/plugin-identity-context-framework-handoff.md)。2026-10-10 核对时，文档仍标为待评审；Core backend 未发现 `resolve_request_identity` 的实现。因此本轮没有注册草案能力、实现草案 typed client，也没有把 delegated 解析切换到不存在的接口。

前端 v2 消费的是宿主用户会话；后端票据证明的是经 Core 网关验证的当前业务请求，两者互补。前端读取用户 context 继续走宿主用户入口；业务请求走插件入口。浏览器请求客户端会去掉调用方提供的 `X-PowerX-Identity-Context`；Core 网关仍必须删除伪造头并注入权威票据。前端不持有票据或 STS／API Key。

本轮已调整错误处理：

| 响应 | 前端处理 |
|---|---|
| 401 + 明确的用户令牌过期码 | 同一登录会话等待已验证快照后最多重试一次 |
| 401 + `AUTH_TOKEN_INVALID`／`TOKEN_INVALID` | 明确失败；嵌入模式不自续期，standalone 仍清自己的失效认证 |
| 401 + `IAM_CONTEXT_PROOF_*` | 保留 HTTP、机器码、原 Trace；显示票据链路错误，不清空浏览器会话 |
| 401 + `AUTH_STS_EXPIRED` | 显示后端服务凭据错误，不触发宿主续期或退出 |
| 401 + `IAM_UNAUTHORIZED` 或未分类错误 | 不猜测来源；保留原错误，不自动续期 |
| 403 | 保留权限拒绝，不续期 |
| 503 + `IAM_CONTEXT_UNAVAILABLE`／424 + `IAM_UPSTREAM_DEPENDENCY` | 显示身份服务不可用，不清会话或跳登录 |

分类只读取结构化 `error.code`／`code`、`reason_code` 和 `details` 中的机器码，不分析错误文案。更具体的 reason 优先；原响应不被改写。原生 fetch 保留实际 Response 及 Trace／request headers，非法 JSON 也不会丢失 HTTP 状态或被当作成功。

### 需要 Core 正式交付的事项

1. 冻结票据头、schema、固定 capability／operation、invocation envelope、TTL 及部署绑定，提供 OpenAPI 和可执行真实请求样例。
2. 冻结 API Key owner、STS 插件范围、当前 grant 和撤销校验；两种服务凭据都必须验证票据的目标插件与租户。
3. 明确用户登录失效、服务凭据失效、票据失效分别使用的机器码。当前 Core JWT middleware 的部分 401 仅返回自由文本 `error`；Framework 不据此猜测过期。浏览器主动续期依赖 snapshot 到期时间，响应触发的续期需要明确用户过期码。上游服务／票据失败不得包装为上述用户过期码。
4. 明确 Root 的 tenant/member 空值与普通成员规则，确认身份响应包含当前角色、权限版本、perms hash、source request、Trace 及有效期。
5. 明确 WS grant／握手如何消费短期票据，以及连接建立后撤销权限和退出的行为；不得在 URL 或日志中记录票据。
6. 交付缺失／伪造／过期票据、跨租户／插件、无 owner／grant、签发后撤销、依赖失败的真实 API Key／STS 验收证据。

### 合同冻结后的 Framework 改动

| 位置 | 待实施内容 |
|---|---|
| Go `iam/contracts` | 强类型票据输入和身份结果；完整保留 UUID、Root、权限版本、有效期及诊断字段 |
| Go `iam/delegated/core_client.go`、`adapter.go` | 服务凭据调用固定 typed binding；迁移后旧 delegated `ResolveIdentity(ctx, pluginBearer)` 明确失败，不回退用户 `/me/context`、Local 或未验证 claims |
| Skeleton IAM adapter／authproxy 与 scaffold／CLI 模板 | 接入可信 Core 请求票据；启动时校验绑定；保留业务权限与租户一致性检查 |
| AI Craft 等消费插件 | 同步复制的 bootstrap、middleware、RBAC 和 WS 授权入口，重新构建制品；只升级依赖不足以完成迁移 |

**当前后端 `ResolveIdentity` 仍调用用户 `/me/context`，audience 混用的根因尚未修复。** 上述表格是下一阶段迁移范围，不能用本轮前端测试宣称后端票据联调已通过。.NET 跟随正式合同另行安排，本轮不扩大其修改范围。
