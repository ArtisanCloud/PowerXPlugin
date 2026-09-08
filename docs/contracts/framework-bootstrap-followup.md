# Framework 装配收尾记录

日期：2026-09-08。仅修改 PowerXPlugin；未修改 Core、授权数据或已安装实例。

## 本轮实现

| 边界 | 实现 | 测试 |
|---|---|---|
| required grant 启动检查 | `runtime/capability.RequireGrants` 按 Core 每批最多 100 项核对顺序、完整性、状态和 reason_code；Skeleton 从 manifest/catalog 读取 required，在 delegated 服务启动前检查 | `runtime/capability/preflight_test.go`、Skeleton `internal/capabilities/required_test.go` |
| local Capability | Core API-key 调试客户端保留在独立依赖中；不再注入 local Registry。插件未提供 local adapter 时明确不可用 | `TestLocalRuntimeDoesNotUseCoreDebugRegistry` |
| Realtime Redis | 显式 Redis 必须获得订阅确认；构造/订阅失败终止启动，清理连接，不切换 MemoryHub。默认或显式 memory 仍有效；未知 provider 拒绝 | `runtime/wsbus/local_factory_test.go` |
| Customer 旧客户端 | 删除 `NewDelegatedCustomerAuthClient` 和 Skeleton 重复选择模式的旧构造器；正式 delegated adapter 由 Runtime 装配 | `runtime/customerfw/runtime_auth_test.go` 及现有正式 Core Auth 合同测试 |

`RequireGrants` 只检查显式 required，不从“客户端存在”推导启用模块，不自动授予权限。
可选模块调用仍由 Core 实时授权；启动通过不是后续授权永久有效的证明。
required 能力不可用会阻止 delegated 启动；缺失非必需 local adapter 不阻止其他模块运行，但访问该 adapter 必须失败。

## 仍待完成的工作

- F1 更新：manifest 已改为插件显式 required 策略，取消对所有消费者强制要求 IAM/Knowledge；补重复/空白/格式校验。已修正 Host Lab 的 IAM directory、Agent lifecycle、Notifications 映射。Core P2 尚未交付的完整授权位置映射不作已验收声明。
- Customer mini-app 的旧密码/tenant 请求形状仍属于插件本地入口；不能因为换了客户端就宣称该入口已经支持 Core Shopify 登录。
- 当前稳定接口的 [local adapter 接入手册](../guides/features/009-consume-powerx-capability/guide.md) 已交付，列出 13 类 contract、装配与验收要求；插件负责实际 local 持久化。T078 已加入正式 Agent Session 扩展。
- Core P1/P3 已交付：T078 完成 Session 客户端接入；P2 剩余能力审核与 Framework 安装态撤权验证仍单独跟踪，不使用旧人工 Session DTO。
- 本轮测试是本地 Go/HTTP mock 合同验证，不是安装态 STS、真实 Redis 集群或 UI 联调。

## 迁移风险

旧 Customer 构造器删除是源码接口变更：消费者应改用正式 Core adapter 和 Runtime，不新增旧路径兼容。
配置了 Redis 却无法连接的环境现在会启动失败；修复 Redis 地址/连通性，或由维护者明确选择 memory，不能自动切换。
delegated 启动失败时检查 required capability 发布、租户 registration 和当前凭证 grant；不要改为 local 绕过。
源码删除可通过 Git 历史恢复；本轮没有删除数据库或用户数据，也没有发布新版本。

## 本地验证结果

- `go test ./framework/backend/go/... ./skeleton/backend/go-gin/... ./tools/cli/internal/templates -count=1`：最终全量通过。首轮发现 Customer 缺失 Runtime 被误映射为 401，已修正为 503 并保留原始错误；未放宽测试断言。
- `npm run sync:templates -- --check`：通过；修改来自 Skeleton 源文件，模板由同步命令生成。
- 本轮代码路径的 `git diff --check`：通过。无安装态成功链路或生产授权验证声明。

### local 接入交付追加验证

- `go test -p 4 ./framework/backend/go/... ./skeleton/backend/go-gin/... ./tools/cli/internal/templates -count=1`：通过。
- `node .codex/skills/ci/manifest-align/scripts/manifest-align-check.mjs --fix`：清单/catalog、exposure RBAC 和资源动作覆盖通过；未使用 `--stage`。
- catalog 重建后 `npm run sync:templates -- --check`：通过；本轮路径 `git diff --check` 通过。
- 已交付指南中的 Notifications 示例由 `ExampleNewRuntime` 实际编译执行，未添加 Framework 生产 local 存储。

## 未初始化依赖边界追加

- 共享 `module.Factory.Mode()` 支持 nil receiver，返回未配置模式，不再 panic。
- `capability.RequireGrants` 在调用前拒绝 typed-nil 指针/函数等检查器，返回 `FRAMEWORK_MODULE_ADAPTER_UNAVAILABLE`，不尝试调用或替换 adapter。
- `runtime/module/runtime_boundaries_test.go` 对 12 类 Runtime 的 nil/零值业务 accessor 统一验证稳定不可用；存在 Mode 方法的模块同时验证模式读取。IAM 使用现有 Registry 的独立 nil/typed-nil 测试。
- 这属于 Framework 自身防御性收尾，不涉及 Core P1–P3 的合同推测或权限数据变更。
- 验证通过：Framework/Skeleton/CLI 全量 Go 回归；`runtime/module` 与 `runtime/capability` 的 `-race` 和 `CGO_ENABLED=0` 定向测试；模板同步检查及本轮变更路径的 diff 检查。

## Customer 错误传播追加

- `customerfw.Error` 保留 Core `StatusCode`（不序列化）与 `ReasonCode`；`HTTPStatus(err)` 和 `ReasonOf(err)` 供插件 transport 使用，`CodeOf` 保留 Framework 语义码。
- Core Auth/membership 失败解析共享机器码提取逻辑；404 membership 缺失不再经 HTTP 映射变成 403/401，502 和 503 不再合并。`error_code` 信封也正确解析。
- Framework 默认中间件、Skeleton mini-app 注册/登录/验证出口及鉴权中间件保留原始状态与原因；本地错误继续使用本地语义映射。
- 对外 message 仅输出机器码，由消费者 locale 显示；不返回内部错误、上游 raw body 或凭证明文。没有新增登录通道、路由或放宽 tenant 输入。
- 回归覆盖成功装配后的错误出口、包装错误和内部文本不泄露；不代表已安装插件真实登录链路验收。
- Customer 批次 Framework/Skeleton/CLI 普通 Go 全量回归通过（113 个测试包）；Customer 相关定向 race 通过。该批次曾发现 middleware 整包 race 失败，后续已由 T076 修复并复测通过，见下方记录。

### 本轮发现的独立剩余项

此前 Skeleton `Timeout` 在 goroutine 中执行 `c.Next()`，超时分支同时操作同一 Gin Context，race 检查报并发读写；无缓冲 finish 通道还可能令超时后的发送方阻塞。此问题已在 T076 修复：HTTP timeout 移到完整 Gin engine 外层，由 Framework `runtime/common/middleware.TimeoutHandler` 管理。Gin Context 只由处理请求的 goroutine 使用，不再被超时分支或外层 Gin middleware 并发访问。

## T076 超时收尾

- `internal/server/adapter.go` 在 Gin engine 外层装配默认五分钟超时；`internal/router/router.go` 移除旧 `engine.Use(Timeout(...))`。
- 普通请求响应先缓冲，成功完成后提交；超时取消 context 并及时返回 408 `REQUEST_TIMEOUT`。晚到写入返回 `http.ErrHandlerTimeout`，不能覆盖超时响应或泄露部分响应/header。
- 完成通知使用有缓冲通道，不会因调用方已经超时返回而卡住发送方。Go 无法强杀不合作的业务 goroutine；数据库/HTTP/任务操作必须使用请求 context，取消不等于事务自动回滚。
- SSE 与 WebSocket 在包裹前显式绕过，原始 writer、长连接和取消行为保持不变。不引入轮询、内存后端或其他传输降级。
- 覆盖普通请求成功、及时超时、请求取消、晚到响应、panic、header 保留、SSE 与 WebSocket。Framework timeout、Skeleton middleware/server/router 的定向 race 已通过，原先失败的 timeout 用例也通过。
- API 调整：Skeleton `Timeout(duration, next http.Handler)` 现在返回 `http.Handler`，不能再放进 `engine.Use`。生成模板同步新装配，不保留旧有竞争的 Gin API。
- 自行启动 Gin 的插件须将外层 handler 交给 HTTP server；直接 `engine.Run()` 不会经过 Skeleton 的 server adapter。普通响应会缓冲至完成，真正的流式路由须在外层明确 bypass，不能依赖普通请求路径上的 Flush。
- T076 最终验证：Framework/Skeleton/CLI 全量 Go 回归 114 个测试包通过；相关 middleware/server/router 整包 race 通过，Timeout 定向 race 重复十次通过；模板检查及本轮 diff 检查通过。未运行服务重启或真实流式安装态联调。

## T077 delegated 响应读取与取消

- Capability、Integration、Skills、Notifications、Plugin Runtime 五个客户端在 token 获取失败或响应体读取失败时，优先传播调用方 context 的取消/截止错误，不将取消改写成 503。
- 未取消但响应体读取失败时返回模块自身 `HTTPError`：502 与对应 `*_UPSTREAM_DEPENDENCY`；不再返回缺乏合同类型的裸 IO 错误。
- 回归验证 token 错误/空 token 后不发出业务 HTTP 请求，响应体失败后关闭资源，读取失败/读取时取消/获取 token 时取消分别保留正确语义。
- Skills 补齐 400/401/403/404/409/429/502/503 错误信封矩阵，及缺失 data、null data、错误 JSON、success:false 的失败测试。
- T077 不改变现有 Host 路由、DTO、授权、不新增自动重试或 local fallback，也不引入未经合同约定的响应大小限制。之后的 Core Session 交付与 T078 接入见下节。

## T078 Agent Session 正式接入

后续 review 发现 T078 仅完成 SDK 和启动装配，旧 Skeleton handler 仍消费 Gateway。T079 已迁移实际消费链路，不能用 T078 的依赖装配测试代替 handler/页面验证。

- 新增 `runtime/powerx/agent/service_sessions.go`、`service_session_events.go`，覆盖正式 `/api/v1/tenant/agent/sessions` 的 12 项操作，使用 UUID DTO 与现有后端 STS provider；API Key/standalone 不可用。
- `runtime/agent.WithSessions(local, delegated)` 在启动期绑定 `SessionService`；`Runtime.Sessions()` 缺失/typed-nil 明确失败，Skeleton 复用同一 Core client 装配，无本地持久化空实现。
- 追加仅 user，幂等键由消费者保存、原样传递，不自动追加消息、不重试 Invoke。订阅明确区分 state/final/error/end；EOF、错误事件和 callback 失败不伪装成功；断开不调用 cancel。
- 旧 delegated `Invoke/StreamSSE` 明确返回 `AGENT_SESSION_REQUIRED`，消费者迁移到 Sessions；不自动创建会话或复用人工路由。生命周期六项保持独立。
- HTTP 状态/Core reason_code 保留；请求不跟随重定向；成功响应校验资源 UUID、状态、分页和请求关联。普通 JSON 响应本地保护上限 8 MiB，SSE 单事件 1 MiB；超限明确依赖错误，不截断返回。订阅继承 HTTP timeout，不自动重连。
- 本地合同覆盖全部路径、错误信封、重复调用/撤权响应、非法角色/UUID/空键、token 取消、缺失 payload、SSE 终止与断连行为。测试不等于 Framework 安装态旧 token 撤权证明；后者保留 T073。
- 验证：`go test -p 4 ./framework/backend/go/... ./skeleton/backend/go-gin/... ./tools/cli/internal/templates -count=1` 通过（114 个有测试包）；最后补充的传输/资源关闭用例随 Agent、Runtime、module 定向 `-race` 通过。`npm run sync:templates -- --check` 与本轮路径 diff 检查通过。未提交、发布、迁移、重启或修改 Core/已安装实例授权。

## T079 Agent 实际消费链路迁移

- `/plugin/agent/sessions` 的 12 项 HTTP 操作全部通过 `Deps.AgentLifecycle.Sessions()`，不再调用 CapabilityGateway。旧人工 Session Gateway 方法、numeric/tenant 输入 DTO 和旧 SSE 路由删除；相应过时测试由新合同测试替换。
- 页面 `agent-skill-bridge` 通过 `useAgentSessionsApi` 消费标准插件 success/data/error 信封；创建仅 agent_uuid/title，消息和执行使用独立幂等键，查询分页使用 page/page_size。不再把 user/tenant、agent_id 或 regen_from_message_id 发给 Core。
- Invoke/订阅分离；同一待确认请求的显式重试保留键和已经取得的消息/执行 UUID。不把 EOF 当成功，不在断连后调用 cancel；停止按钮显式请求 cancel。Core 未发布的历史消息改写不提供兼容功能。
- Skeleton 清单声明 `com.corex.agent.session.manage`、`com.corex.agent.invoke`，RBAC 同步正式 12 路由。声明不等于获授，已安装实例需后续按正式升级/启用流程同步；本轮未修改实际 grant。
- local 仍需插件注入 SessionService；未注入时返回稳定不可用，不借用 API Key/Core Gateway 冒充 local。
- 验证结果：Framework/Skeleton/CLI 全量 Go 回归通过（114 个有测试包）；Agent handler、Gateway、skills 定向 race 通过；Session 前端单元测试 5 项通过，Nuxt build 通过，Manifest/模板同步检查通过。Playwright 新合同用例 4 项仅做发现检查，未运行浏览器联调。
- 前端整套 unit 执行结果为 15 通过、8 失败；失败全部来自既有 `useAuth.fallback.spec.ts` 无法解析 `tenant-context.ts` 的 `#app` 导入。单独执行该文件可重现；本轮未修改认证逻辑或该测试配置，不能宣称前端全量 unit 通过。
- T077 验证：Framework/Skeleton/CLI 全量 Go 回归 114 个测试包通过，五个客户端整包 race 通过，新增 Skills 矩阵再次 race 通过；模板检查及本轮 diff 检查通过。
