# Framework 用户会话失效接入

权威接口是 [Core 合同快照](plugin-session-termination-v2.md)，来源与摘要见 [来源记录](plugin-session-termination-source.json)。本轮只修改 PowerXPlugin 的共享前端客户端、Admin 能力调用、Skeleton 与生成链路。Framework 不调用宿主私有 session/refresh/logout 接口，不拥有宿主登录跳转或存储。

## 调用与状态边界

- HTTP 401 的结构化 `IAM_CONTEXT_USER_SESSION_INVALID`、WS `type=error` 帧及 SSE `event=error` 帧中的同一结构化码，只有属于当前 snapshot 才报告。报告经 Core 注入的 `reportSessionInvalid({sessionId, revision, code})` 发送；sequence 由 Core 消费器填写。报告不携带 Token、成员、租户或返回地址。同一 snapshot 并发只报告一次。
- 报告后停止当前受保护请求和连接，清理插件身份。保留业务 store，让插件自行按原用户、租户保存草稿；是否真正退出由宿主核验。宿主确认 `auth-cleared`、新登录或验证到身份变化时，清理完整插件会话状态。恢复时不自动重放旧业务写入。
- `IAM_CONTEXT_USER_SESSION_EXPIRED` 保留 API 的一次续期重试。fetch-SSE 的初始 GET 在尚未消费事件时最多续期重试一次；流已经开始后不自动重放。原生 WebSocket/EventSource 的网络错误与关闭原因没有可靠结构化状态，不猜测失效，不自动续期或跳转；应用在新有效 snapshot 下显式重连。
- 403、服务凭据、proof、未知 401、503 不报告登录失效。过期续期只接受同一登录与身份的新快照，身份变化则取消旧操作。
- 初始化 context 请求失败不另外发送 `auth:required`：`PowerXHostContextError` 保留结构化 code/statusCode，原始错误放在 `cause`；Core 消费器通过 `auth:rejected` 处理。携带 INVALID 但 HTTP 状态不是 401 的异常不会伪装为登录失效。旧 context 的晚到错误不能拒绝新登录。
- 旧 Core 缺少 `reportSessionInvalid` 时，在首次需要该功能时明确返回 `PLUGIN_BRIDGE_SESSION_TERMINATION_REQUIRED`（409），暂停身份消费并显示 locale 提示。不会改为直接发消息、刷新 Token 或跳转父窗口。

## 插件接入

使用共享 `createPowerXApiFetchClient`、`createPluginWsClient`/`createPluginWsBusClient`、`createPluginSSEClient` 和 `PowerXHostSession`，不要复制“全部 401 退出”的回调。Admin 的能力调用也通过共享 API 客户端；原始结构化错误继续抛出。

使用 `createPowerXRouterNavigator(router, baseURL, origin, session)` 将 Router 操作绑定到当前会话，身份变化会在路由提交前取消旧导航。

自定义 `PowerXHostSessionAdapter.clearContext(scope)` 时，`identity` 只清身份与受保护连接，`session` 清完整会话及原身份业务状态。独立插件必须明确实现这两个清理范围。Skeleton 已落实并同步到 scaffold/CLI 模板；共享包不能自动替换既有插件自己的 adapter。

Bridge 的 `onAuthToken` 必须返回 `session.apply(...)` 的 Promise，使 Core 在 context 验证完成后才发出 applied/rejected。WS/EventSource 可用 `onSessionError` 接收桥接能力错误；同时可订阅 `session.getState().error`。用户反馈由 locale 翻译，不能把机器码当正文。

## 可复跑验证

```sh
node scripts/contracts/validate-session-termination.mjs
node scripts/sync-templates.mjs --check
cd skeleton/web-admin/nuxt
npm run test:unit -- tests/unit/session-termination-v2.spec.ts tests/unit/session-termination-admin.spec.ts tests/unit/session-termination-adapter.spec.ts tests/unit/session-termination-router.spec.ts tests/unit/bridge-v2-interoperability.spec.ts
```

`python3 scripts/contracts/package-session-termination.py` 生成内容寻址的本地 client/Admin 包和 CLI 示例，以实际安装字节复跑协议测试与严格类型检查；不发布 npm 或 Git tag。`node scripts/contracts/test-session-termination-browser.mjs <bundle>` 使用同包和真实 Core 注入消费器进行隔离浏览器验证，服务在 finally 中关闭。

验收记录：[2026-11-07 Framework 验证](evidence/plugin-session-termination-framework-validation-20261107.json)。源码、受控测试与本地包验证不代表 npm 已发布、既有插件已安装或日常 Core/AI Craft 已联调。AI Craft 仍需消费相同版本、移除自行退出逻辑、绑定草稿身份，随后由各仓库所有者完成部署与真实登录验收。
