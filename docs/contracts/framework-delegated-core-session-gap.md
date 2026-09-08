# Agent Session：历史合同差异与正式接入

## 当前结论（2026-09-08 更新）

Core 已发布独立 `/api/v1/tenant/agent/sessions`，Framework 已实现 12 项 SessionService 操作、`Runtime.Sessions()` 和 Skeleton STS 装配。以下“不一致/等待冻结”段落是本次交付前的历史记录，不再作为阻塞项。人工历史模型仍不是插件 Session 资源，也没有迁移为插件归属。

正式依据：Core `specs/007-integration-gateway-and-mcp/contracts/agent-session.http-openapi.yaml`、`internal/transport/http/openapi/agent_session/handler.go` 和 `docs/contracts/agent-session-grant-delivery-status.md`。本轮代码及测试见 Framework `runtime/powerx/agent/service_sessions*.go`、`service_session_events.go`；插件接入见 local adapter 手册第 7 节。

Session 仅 STS、追加仅 user、Invoke 幂等、SSE 订阅不执行且断连不取消；旧 delegated Invoke/StreamSSE 返回 `AGENT_SESSION_REQUIRED`，不自动创建资源或兼容 numeric 路径。Core P2 静态队列/总线、动态能力与 Framework 安装态验收仍独立跟踪。

## 以下为历史审计记录

复核日期：2026-09-08。仅记录本地 Core 源码证据；没有修改、提交、部署或重启 Core。

## 已确认的不一致

| 来源 | 当前事实 |
|---|---|
| Core `specs/007-integration-gateway-and-mcp/contracts/agent.http-openapi.yaml` | CreateSessionRequest 使用 UUID `agent_id`；响应使用 UUID `session_id` |
| Core `backend/config/platform_capabilities/agent.yaml` | `com.corex.agent.session.manage` 声明 POST `/api/v1/agents/sessions` |
| Core `backend/internal/transport/http/admin/agent/agent_session_handler.go` | 实际 createSessionReq 接受 numeric `agentId/userId` 以及 `agentUuid`，并直接返回会话服务对象 |
| Core `backend/internal/transport/http/admin/agent/api.go` | 会话详情、更新、归档、删除、消息列表路由使用 `:id`；不能依据参数名推断其已支持 UUID |
| Framework `runtime/agent/runtime.go` | 已有 invoke、SSE、六项 lifecycle；session 字段仅用于调用关联，不是 session CRUD |

## 请求 Core 冻结并实现的边界

1. 明确正式服务态 session 最小操作集：创建、读取、追加消息、消息列表及会话调用/SSE；生命周期中的更新、归档、删除若发布也需列入 OpenAPI。不要仅添加声明而继续返回数据库模型。
2. DTO 只接受、返回稳定的 `agent_uuid/session_uuid/message_uuid` 对象引用。字段名若决定沿用 OpenAPI 的 `agent_id/session_id`，必须明确其类型只能是 UUID，并与 handler 一致；双方冻结后 Framework 只实现该一种格式。禁止 numeric 兼容入口。
3. 租户和插件服务主体从 STS 推导；服务态流程不能依赖人工 numeric userId。若业务确实需要委托成员，单独定义验证合同，不能由插件任意传 userId。
4. 对每个正式路由登记 capability binding，并校验能力发布、当前租户注册和当前凭证 grant。同一租户内是否允许不同插件访问同一 session 必须明示；默认不得靠猜测 UUID 获得访问权。
5. 统一返回 data envelope、稳定 reason_code；覆盖 400 非法 UUID/身份覆盖、401、403 未获授权、404 缺失或越界 session、409 状态冲突、502/503 依赖错误。
6. 成功消息/SSE 保留 trace 与 session UUID，取消后终止读取；不得在错误后自动切换另一 agent、local adapter 或传输。
7. 提供实际 handler/业务 service/合同测试路径与测试结果。涉及对象 UUID/关联迁移时先 migrate；capability 变更发布 seed 后重启。

## Framework 已完成及后续接入

- 已将 Agent SSE 改为逐事件回调；消费者返回错误即停止，不缓存到 EOF。
- 已补 HTTP status、Core reason_code、trace/request ID 的错误传递。
- 已修复 Skeleton Agent 测试替身缺失接口导致的编译失败。
- Core session 合同未冻结前，不增加猜测请求格式的客户端，也不把 Agent 整体标为已完成。
- 合同冻结后补 session DTO、AgentService 方法、delegated transport、local 注入合同与对应替身/测试，再冻结面向插件的统一 local 接入文档。插件仍不需要实现 Core HTTP 或在业务层判断模式。

## 本轮其他修复

- Knowledge 补 Framework 自身的 `NewRuntime(...).Provider()`；Skeleton 通过该入口装配。
- Notifications 对齐真实 `POST /api/v1/notifications` 和 `com.corex.notifications.create`；不增加旧路径兼容。
- Media 实际 tenant Host 是 10 项操作；补逐操作及错误矩阵测试，拒绝空/缺失 data 与必需资源引用。
- Plugin Release 的 stop 只确认受理的 session UUID，不自行制造 `stopping` 状态；状态由 GetInstallSession 获取。
- Customer Auth validate 不允许空身份/非 active membership 被转换为已认证上下文。
- Metadata 解析覆盖分页，拒绝缺失 payload，管理端保留稳定错误与状态；Factory 拒绝 typed nil adapter。

上述代码测试不代表已完成安装态、跨租户或第三方服务真实联调；后者按用户要求后置。

## 后续补齐记录（2026-09-08）

- Metadata：所有 UUID 定位请求在出站前校验规范 UUID（拒绝 numeric、nil UUID 和非法路径）；对象响应必须带有效 UUID，列表必须带有效分页和资源 UUID。字典项创建补齐 Core 已支持的 metadata 字段。空 STS 不发送请求。
- Customer：正式 Auth/Membership 客户端改用严格 data 信封解码，不再复用旧裸对象解码；HTTP 成功但 success=false 仍明确失败。增加同一个客户端并发 20 个客户凭证的隔离测试。
- Agent：SSE 校验准确的媒体类型，拒绝非 2xx 与空客户端；真实 httptest HTTP 流验证取消后关闭宿主请求，不依赖缓存或轮询。
- IAM：Registry 拒绝 typed nil 的 Directory/Authz/Context；失败绑定不写入 bound 状态，nil Registry 读取返回未绑定错误而非 panic。

本轮不改变 Core Session 缺口结论，也未冻结包含独立 Agent Session 的 local 全模块交付文档。
