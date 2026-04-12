# MCP 会话与流式能力联调手册

MCP（Model Context Protocol）由插件直接托管，PowerX 只负责调度。本文介绍如何在本地调试 REGISTER → ACK → Heartbeat → Invoke → SSE/WS 全流程，并说明 PowerX 如何消费这些接口。

## 1. 前置准备

1. 启动本地后端（参考《agent-rest-grpc-guide.md》第 2、3 节获取 Token）。
2. 确保以下文件已通过 `npm --prefix scripts/capabilities run export` 更新：
   - `contracts/exposure/mcp-tools.json`
   - `contracts/exposure/agent-streams/*.yaml`

## 2. MCP 会话生命周期

MCP 运行期是一个“先建会话、再注入工具、最后执行任务”的闭环。你需要依次完成 Register → Ack → Heartbeat → Invoke → Close，这样宿主或本地调试工具才能追踪插件状态。所有管理接口位于 `/api/v1/admin/runtime/sessions/*`：

| 步骤 | 接口 | 说明 |
|------|------|------|
| Register | `POST /api/v1/admin/runtime/sessions/register` | 创建会话，绑定 runtime assignment / tenant / JWT ID |
| Ack | `POST /api/v1/admin/runtime/sessions/{sessionID}/ack` | MCP 客户端加载完工具后反馈 READY，并可附带 `capabilities_hash` |
| Heartbeat | `POST /api/v1/admin/runtime/sessions/{sessionID}/heartbeat` | 定期上报心跳，若 `missed_heartbeats` > 0 会触发告警 |
| Close | `POST /api/v1/admin/runtime/sessions/{sessionID}/close` | 主动下线会话 |
| Invoke | `POST /api/v1/admin/runtime/sessions/{sessionID}/invoke` | 将 Integration Envelope 转交调度服务 |

所有请求都需要 `Authorization: Bearer <access_token>`，并确保租户上下文中具备 `runtime.ops` 权限（本地 DevSwitch 已默认注入）。

### 2.1 Register

**目的**：向插件后端登记一个全新的 MCP 会话，让宿主或本地调度器知道“哪一个 runtime assignment、tenant 与客户端实例”即将使用哪些工具”。这里的 *session* 指“一次 MCP 客户端与插件保持长连接、可反复调度多个 Workflow 场景的上下文容器”，由插件负责生成与管理。

**影响**：成功后会分配 `session_id` 并锁定租户上下文（相当于给该客户端开辟一个独立沙箱）；后续所有 Ack/Invoke 请求都必须带上这个 ID，否则宿主无法把事件路由回正确的客户端。若此步骤省略，PowerX 会认为 MCP 客户端尚未处于可调度状态。

```bash
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/register \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "runtime_assignment_id": "b1f9a20b-1111-4bee-99af-03d11f6ef001",
        "state": "registering",
        "jwt_id": "dev-mcp-client",
        "capabilities_hash": "sha256:..."
      }'
```





字段说明：

- `runtime_assignment_id`：PowerX Dev Console 中为该插件创建的 Runtime Assignment UUID。本地独立模式没有现成记录，可执行 `uuidgen`（或任意固定 UUID）并在 Register/Ack 阶段保持一致即可；宿主模式下由 PowerX 在分配 runtime assignment 时注入。
- `state`：注册阶段固定写 `registering`，表示客户端刚开始握手；Ack 阶段再切换为 `ready`。
- `jwt_id`：客户端自身的唯一标识，用于宿主审计（可以是进程名、容器实例 ID 或 `dev-mcp-client`），PowerX 会把该值映射到 MCP Session 的 JWT 声明。
- `capabilities_hash`：可选，建议写入 `contracts/exposure/mcp-tools.json` 内容的 SHA256 摘要；PowerX 通过比对 hash 判断工具集是否需要刷新，例如：
  ```bash
  shasum -a 256 contracts/exposure/mcp-tools.json | cut -d' ' -f1
  ```

> 提示：`skeleton/backend/go-gin/etc/integration/grant_matrix.yaml` 仅声明 tool scope 与通道授权，不会包含 runtime assignment ID；本地调试时直接使用手动生成的 UUID 即可。

返回示例：

```json
{
  "success": true,
  "data": {
    "id": "6e7f...",
    "state": "REGISTERED",
    "tenant_uuid": "00000000-0000-0000-0000-000000000001"
  }
}
```

### 2.2 Ack & Heartbeat

**目的（Ack）**：通知宿主“工具加载完毕，可以接收任务”。这一步会携带 `capabilities_hash`，PowerX 会据此判断 manifest 是否过期，从而决定是否刷新工具集。

**目的（Heartbeat）**：保持保活，避免 0 活跃会话被宿主清理。`missed_heartbeats` > 0 时，宿主会把 session 标记为风险状态并暂停新任务下发。

**影响**：只要 Ack 未完成，Agent Hub 就不会把真实 Invoke 投递到该 session；心跳中断也会触发 Force Close，SSE/WS 立即断开。

```bash
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/ack \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{ "state": "ready", "capabilities_hash": "sha256:..." }'

curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/heartbeat \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{ "missed_heartbeats": 0 }'
```

### 2.3 Invoke（Integration Envelope）

**目的**：真正触发 Workflow/Agent 任务。PowerX 会将意图识别后的 `tool_scope`、`intent` 与业务 payload 封装成 Integration Envelope，通过该接口推送给插件。

**影响**：插件收到 Invoke 后需要同步/异步返回执行结果或推送 SSE。如果 payload 不符合 `contracts/exposure/mcp-tools.json` 声明的 schema，会直接 400 并在 Agent Hub 上报失败；多次失败也会降低意图评分。

`invoke` 接口复用 Integration Dispatch：

```bash
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "d1ec6e47-b3c2-4ca5-8a93-a8e8ca5c25b2",
        "trace_id": "cd4f35d2-0f8b-41cc-92d1-92a8aa61e817",
        "correlation_id": "72c0da4d-19f2-4c1f-8ebd-8b5f81a73232",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template",
        "issued_at": "2025-12-09T02:20:00Z",
        "payload_ref": "{\"action\":\"compose\",\"args\":{...}}",
        "idempotency_key": "dev-run-001",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.compose"
        },
        "signature": "stub-signature"
      }'
```

成功后会返回 `status/trace_id/correlation_id/latency_ms/replay`。若 Grant Matrix 未允许该 tool scope，将收到 403。

## 3. 调试模式与推荐步骤

前面 2.x 小节解释的是“接口行为”。真正落地时通常分两条路径：

### 3.1 本地 skeleton（独立插件）

1. `cd skeleton/backend/go-gin && POWERX_PROXY=0 go run ./cmd/plugin`，并按《agent-rest-grpc-guide.md》获取本地 token。
2. 依序调用 2.1~2.3 中的 Register/Ack/Heartbeat/Invoke，`tool_scope`、`metadata.intent`、`capability_id` 直接引用 `contracts/exposure/*` 的声明。
3. 使用第 4 节中的 SSE/WS 订阅接口（`/api/v1/mcp/*`）观察事件；必要时结合日志确认 handler 是否执行。
4. 调试完成后调用 Close 释放会话。

此模式完全在插件仓库内闭环，适合开发期验证 handler、Grant Matrix、事件流等实现细节。

### 3.2 宿主 PowerX Dev/Prod

1. `npm --prefix scripts/capabilities run export`，随后 `px-plugin capabilities submit / quota` 将能力目录提交到宿主；确保目标租户具备授权。
2. 在 PowerX Dev Console 绑定 runtime assignment，宿主会负责创建 session、注入租户上下文，并把事件中继到 `/internal/plugins/<plugin-id>/mcp/sse`。
3. 在 Agent Hub 或 Workflow 入口触发真实意图，宿主会调用插件的 `/api/v1/admin/runtime/sessions/*` 与 `/invoke`。你只需监控日志/SSE 确认流程与本地保持一致。

> 提示：宿主模式下 Register/Ack/Heartbeat 由 PowerX 调度驱动，插件保证接口可用即可；真正需要关注的是 Invoke 行为、事件回传以及与宿主配额/限流的配合。

## 4. 订阅 MCP 事件（SSE / WebSocket）

完成 Register/Ack 之后，建议在另一个终端实时订阅事件，方便确认 `session.registered`、`session.ready`、`invoke.completed` 等通知是否按期望推送。

**SSE（Server-Sent Events）**（注意：框架会把插件挂在 `/api/v1` 前缀下，本地访问请带上该前缀）

```bash
curl -N \
  -H "Authorization: Bearer <token>" \
  "http://127.0.0.1:8078/api/v1/mcp/sse?session_id=<sessionID>"
```

- `-N` 关闭 curl 缓冲，可以连续看到 `event:`/`data:` 输出。
- 每当你 Register/Ack/Invoke/Close，终端都会打印对应事件；`payload` 会携带状态、trace_id 等关键信息。

**WebSocket**（如果想复现浏览器/Agent Hub 体验）

```bash
npx wscat \
  -H "Authorization: Bearer <token>" \
  -c "ws://127.0.0.1:8078/api/v1/mcp/ws?session_id=<sessionID>"
```

- 也可以使用 Insomnia/Postman/Newman 等工具发起 WebSocket 连接，只要在 Header 中附带 `Authorization` 即可。
- 连接建立后，所有事件会以 JSON 字符串推送；如需调试特定 intent，可与 Invoke 请求同时进行。

完成调试后，记得 `POST /api/v1/admin/runtime/sessions/<sessionID>/close` 主动关闭会话，SSE/WS 也会收到 `session.closed` 事件。

## 5. 同一 Agent Session 测试多个 Workflow 场景

**目的**：验证意图识别与工具路由逻辑，确保同一会话能够按 `intent` 选择正确的 Workflow，而不是每个场景都重新 Register。

**影响**：如果 `tool_scope`/`intent` 设置错误，宿主可能把 Workflow 绑定到错误的 SSE 通道，导致 Agent Hub 收不到期望事件。在本节示例中，你可以连续触发三个场景并观察 SSE channel 是否对应 `template.compose`、`template.audit`、`template.quality_distribute`。

巡检场景（`com.powerx.plugins.base.template.audit`）与协作场景（`com.powerx.plugins.base.template.compose`）可以复用同一个 MCP Session，只要在 `invoke` 的 `metadata.intent` 与 `tool_scope` 上区分即可：

```bash
# 场景 A：template.compose（草稿→审核→清理）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "b7bfd5dd-ec30-4c1e-9fb3-3a0b9e8be001",
        "trace_id": "e65b33a8-1d78-4d2e-8152-2d6512f60001",
        "correlation_id": "ce2b0732-bddb-4239-9216-9e9091700001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.compose",
        "issued_at": "2025-12-11T06:10:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.compose",
          "capability_id": "com.powerx.plugins.base.template.compose"
        },
        "payload_ref": "{\"draft\":{\"name\":\"Demo Template\",\"description\":\"由 MCP 指南自动创建\",\"content\":\"## hello world\"},\"review\":{\"reviewer\":\"qa-bot\",\"comment\":\"looks good\"},\"publish_channel\":\"channel:demo\",\"cleanup\":{\"reason\":\"archived after publish\"}}",
        "signature": "stub-signature"
      }'
```

示例 payload_ref（原始 JSON，使用 `jq -c` 压缩成单行后再嵌入字符串）：

```json
{
  "draft": {
    "name": "Demo Template",
    "description": "由 MCP 指南自动创建",
    "content": "## hello world"
  },
  "review": {
    "reviewer": "qa-bot",
    "comment": "looks good"
  },
  "publish_channel": "channel:demo",
  "cleanup": {
    "reason": "archived after publish"
  }
}

```

> 响应体会包含 `status`、`review_status`、`cleanup_reason` 以及 `lifecycle`（依次列出 draft/review/publish/cleanup 四个阶段）；为了兼容既有脚本，`draft_id` 与 `publish_status` 仍然保留，你可以在新版字段和旧版字段之间自由切换。

```bash
# 场景 B：template.audit（巡检→修复）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "e5c4c5d2-4f63-4f81-b884-6d7ef5f10001",
        "trace_id": "cb2232ac-0aab-477e-8e68-f77a3ce40001",
        "correlation_id": "7f0b4f0c-0d9c-4c35-9d8c-43e2b7780001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.audit",
        "issued_at": "2025-12-11T06:12:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.audit",
          "capability_id": "com.powerx.plugins.base.template.audit"
        },
        "payload_ref": "{\"filters\":{\"status\":\"\",\"page\":1,\"page_size\":5},\"update_payload\":{\"description\":\"reviewed by qa-team\",\"content\":\"## patched content\",\"metadata\":{\"owner\":\"qa-team\"}}}",
        "signature": "stub-signature"
      }'
```

示例 payload_ref：

```json
{
  "filters": {
    "status": "",
    "page": 1,
    "page_size": 5
  },
  "update_payload": {
    "description": "reviewed by qa-team",
    "content": "## patched content",
    "metadata": {
      "owner": "qa-team"
    }
  }
}
```

> `status` 置空（或填 `"*"`）即可让 Workflow 捕获已发布/已下架的模板，便于在混合环境里验证“有的记录被跳过、有的记录被更新”的效果。如果只想巡检草稿，可把 `status` 改回 `"draft"`。

```bash
# 场景 C：template.quality_distribute（巡检 + 批量克隆 + 更新）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "6a35ff98-7f15-4ddb-bccb-75e81fe20001",
        "trace_id": "2b0c7054-b4b2-4572-81a7-ef2f5c880001",
        "correlation_id": "a5f63bb0-2583-4df6-8f01-5247f57c0001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.quality_distribute",
        "issued_at": "2025-12-11T06:15:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.quality_distribute",
          "capability_id": "com.powerx.plugins.base.template.quality_distribute"
        },
        "payload_ref": "{\"scan_filter\":{\"q\":\"demo\",\"page\":1,\"page_size\":5},\"validate_rules\":[\"name_not_empty\",\"content_min_length\"],\"clone\":{\"copies\":2,\"name_prefix\":\"qa-copy-\",\"description_prefix\":\"[auto]\"},\"update_payload\":{\"description\":\"distributed by QA\",\"content\":\"## fixed content\"},\"publish_channel\":\"channel:qa-lab\"}",
        "signature": "stub-signature"
      }'
```

示例 payload_ref：

```json
{
  "scan_filter": {
    "q": "demo",
    "page": 1,
    "page_size": 5
  },
  "validate_rules": [
    "name_not_empty",
    "content_min_length"
  ],
  "clone": {
    "copies": 2,
    "name_prefix": "qa-copy-",
    "description_prefix": "[auto]"
  },
  "update_payload": {
    "description": "distributed by QA",
    "content": "## fixed content"
  },
  "publish_channel": "channel:qa-lab"
}
```

> 模板模型已新增 `status`、`review_status`、`published_at`、`publish_channel`、`cleanup_reason`、`cleaned_at` 等字段（定义见 `skeleton/backend/go-gin/internal/entity/models/template/template.go`），每次 Workflow/CRUD 都会按阶段更新这些字段，便于直接在数据库里还原生命周期。

### 三个场景分别会产生哪些记录？

| 场景 | 落库 / 事件 | 说明 |
|------|-------------|------|
| template.compose | 1) `template` 表新增一条记录，字段 `status` → `archived`、`review_status` → `approved`、`publish_channel`/`published_at`/`cleaned_at` 均会被写入（详见 `skeleton/backend/go-gin/internal/entity/models/template/template.go`）；<br>2) SSE 依次推送 `draft.created` → `template.review.completed` → `publish.status` → `template.publish.completed` → `template.cleanup.completed`，可在事件 payload 中看到三阶段的状态。 | `handleTemplateCompose` 先写入草稿，再调用 `TemplateService.MarkReviewed/Publish/Cleanup` 贯穿草稿→审核→清理流程。最终你可以在同一条模板记录上看到完整生命周期痕迹（草稿时间、审核信息、发布时间以及清理原因），同时 SSE 事件便于 Agent 端同步每个阶段的结果。 |
| template.audit | 1) `template` 表中命中的第一条记录被 `TemplateService.Update` 覆盖描述/内容；<br>2) SSE 推送 `audit.template.updated`，payload 包含最新描述与模板 ID。 | `handleTemplateAudit` 会按分页配置 (`filters.page/page_size`) 查询，并在找到模板后执行修复，响应体里的 `selected_template_id` 与事件 payload 可用来比对。 |
| template.quality_distribute | 1) `TemplateService.List`、`Validate` 仅读取数据，不写库；<br>2) `TemplateService.BatchClone` 按 `clone.copies` 写入 N 条副本（`template` 表增加多条记录）；<br>3) `TemplateService.Update` 再次修改目标模板/副本内容；<br>4) SSE 依次推送 `template.validate.completed`、`template.batch_clone.completed`、`template.update.completed`（事件定义见 `contracts/exposure/agent-streams/template-quality-distribute.yaml`）。 | 该 Workflow 串联了 `list → validate → batch_clone → update`（参考 `contracts/exposure/workflow/template-quality-distribute.json`）。执行完成后可通过 `template` 表新增的副本 ID 与 SSE 事件序列验证巡检、克隆与更新各阶段。 |

> 如未看到上述记录，请确认：1）`payload_ref` 使用内联 JSON；2）`HostInvoker` 已由 `CapabilityInvoker` 接管（`BuildDispatchService` 会默认注入）。若仍落到 `NoopInvoker`，自然不会写入业务数据。

每次 `invoke` 与 MCP 工具的绑定关系：

- `capability_id` → `contracts/exposure/workflow/*.json` 中的 Workflow ID。
- `intent` → `contracts/exposure/agent-streams/*.yaml` 中声明的 `intent`，PowerX Agent Hub 会据此做意图识别与 SSE 过滤。
- 如果要串联测试，只需对同一个 session 连续发起不同 intent 的 `invoke`，并观察后续 SSE 事件是否落在正确的 channel。

### MCP 单节点 CRUD 能力（list/read/create/update/delete）

除了三条 Workflow，`com.powerx.plugins.base.template.*` 还暴露了 CRUD 原子能力，每个能力都在 `contracts/capabilities/com.powerx.plugins.base.template.<name>.yaml` 中定义了 `metadata.protocols.agent_tool.scope`，并在 `skeleton/backend/go-gin/etc/integration/grant_matrix.yaml` 注册了 `agent.template.<name>`。下面示例同样补齐了所有字段，可直接粘贴运行（把 `<sessionID>`、`<token>`、`<templateID>` 替换成实际值）：

> `/invoke` 的 HTTP 响应统一返回 `{"success":true,"data":{...}}` 包装体，现在 `data.payload` 会直接嵌入 JSON 对象，不再是字符串。如果你在命令行想立即查看返回值，可像 list 示例那样追加 `| jq '.data.payload'`；在 Insomnia/Postman 里则直接展开 `data.payload` 即可。

```bash
# list（分页查询模板）
curl -s -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "2c38b195-dcb2-4205-9b44-8220c72fe001",
        "trace_id": "021bd88c-5f1a-4f98-8f68-4dc0bb240001",
        "correlation_id": "3e4412e2-4bfb-4bff-a282-1b8dcb130001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.list",
        "issued_at": "2025-12-11T06:20:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.list",
          "capability_id": "com.powerx.plugins.base.template.list"
        },
        "payload_ref": "{\"q\":\"\",\"page\":1,\"page_size\":10}",
        "signature": "stub-signature"
      }' | jq '.data.payload'
```

```bash
# read（读取模板详情）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "a61233cd-6ce8-4fd2-92bb-8625e1d10001",
        "trace_id": "7ac19ebc-a0e3-4b42-8306-548c71750001",
        "correlation_id": "ee96f3a7-9b73-4f0b-98db-64ad3f4f0001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.read",
        "issued_at": "2025-12-11T06:21:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.read",
          "capability_id": "com.powerx.plugins.base.template.read"
        },
        "payload_ref": "{\"template_id\":<templateID>}",
        "signature": "stub-signature"
      }'
```

```bash
# create（创建单个模板）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "4b9a2798-6e0c-4ce9-8c79-2d7998d00001",
        "trace_id": "5eeac30d-aee7-47eb-8ef1-5273f6550001",
        "correlation_id": "f3c24c8f-4b5d-4f2a-8722-1abfae8a0001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.create",
        "issued_at": "2025-12-11T06:22:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.create",
          "capability_id": "com.powerx.plugins.base.template.create"
        },
        "payload_ref": "{\"name\":\"MCP Template\",\"description\":\"由 CRUD 示例创建\",\"content\":\"## generated\"}",
        "signature": "stub-signature"
      }'
```

```bash
# update（缺省字段沿用现值）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "808d7ce8-a171-4b6c-87dd-62a513740001",
        "trace_id": "fa03a0c7-7578-4a1c-8d72-5b3f7d240001",
        "correlation_id": "358681c8-52cb-43ad-b683-2f75de420001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.update",
        "issued_at": "2025-12-11T06:23:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.update",
          "capability_id": "com.powerx.plugins.base.template.update"
        },
        "payload_ref": "{\"template_id\":<templateID>,\"description\":\"updated via MCP\",\"content\":\"## patched body\"}",
        "signature": "stub-signature"
      }'
```

```bash
# delete（删除模板）
curl -X POST http://127.0.0.1:8078/api/v1/admin/runtime/sessions/<sessionID>/invoke \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
        "message_id": "aa0278b2-bc64-4b25-8974-b6f9077c0001",
        "trace_id": "6f4f8d62-e892-47ab-a6ee-070f4ad90001",
        "correlation_id": "eafdd91e-45ff-4c35-9d33-b5f7d2e00001",
        "tenant_uuid": "00000000-0000-0000-0000-000000000001",
        "tool_scope": "agent.template.delete",
        "issued_at": "2025-12-11T06:24:00Z",
        "metadata": {
          "session_id": "<sessionID>",
          "intent": "template.delete",
          "capability_id": "com.powerx.plugins.base.template.delete"
        },
        "payload_ref": "{\"template_id\":<templateID>}",
        "signature": "stub-signature"
      }'
```

返回值说明：

- list → `Page` 结构（`list` + `total`），可作为 read/update 的输入。
- read → 只读不写库。
- create → 新增一条记录，可被 Workflow 立刻消费。
- update → 当某个字段未提供时自动沿用当前值，方便局部修改。
- delete → 返回 `{"deleted": true, "template_id": <id>}` 并实际删除模板。

### 本地运行集成测试

想快速确认 compose / audit / quality 三个场景是否会按预期写入模板数据库，可直接运行随仓库提供的 Go 测试：

```bash
cd skeleton/backend/go-gin
go test ./internal/services/integration/...
```

如需只跑其中一个场景，可附加 `-run TestCapabilityInvokerCompose`（或 `...Audit`、`...QualityDistribute`）。测试使用内存 SQLite，无需额外配置，就能模拟文档中的 payload 是否会落库、产生 SSE 事件。

## 3. 订阅 SSE / WebSocket

PowerX 宿主会把插件整体挂载到 `/api/v1` 前缀之下，因此本地访问 SSE/WS 也要带同样的前缀。

- SSE：
  ```bash
  curl -N \
    -H "Authorization: Bearer <token>" \
    "http://127.0.0.1:8078/api/v1/mcp/sse?session_id=<sessionID>"
  ```
- WebSocket：使用 Insomnia/Postman 或 `wscat` 连接 `ws://127.0.0.1:8078/api/v1/mcp/ws?session_id=<sessionID>`。

订阅成功后连接会保持打开，屏幕没有任何输出属于正常现象——只要再发一次 Heartbeat/Invoke，就能在 SSE/WS 终端看到对应的 `session.heartbeat`、`invoke.completed` 等事件；每隔 25 秒还会有 `ping` 用于保活。SSE/WS 只是“观察窗口”，真正触发 Workflow 的步骤仍然是上文提到的 `/invoke` 接口。

事件格式：

```json
{
  "session_id": "6e7f...",
  "type": "invoke.completed",
  "payload": {
    "status": "accepted",
    "trace_id": "cd4f..."
  },
  "timestamp": "2025-12-09T02:25:00Z"
}
```

> 提示：SSE/WS 走根路由且继承全局中间件，如需在联调工具中携带 token，可追加 `Authorization` Header。

当你在同一链接里测试多个 Workflow，SSE payload 的 `intent` 字段会分别显示 `template.compose`、`template.audit` 或 `template.quality_distribute`。可以在 Insomnia/`curl -N` 中过滤出单个 intent，以确认意图识别是否路由到了正确的 Workflow。

## 4. PowerX 调度方式

1. 插件通过 `px-plugin capabilities submit` 上报 catalog 时，`contracts/exposure/mcp-tools.json` 与 `agent-streams/*.yaml` 会随 `export` 产物一并上传。
2. PowerX Agent Hub 根据 MCP Tool Manifest 生成工具卡片：
   - 客户端向 PowerX 申请调度 → 宿主与插件通过 `/api/v1/admin/runtime/sessions/register` 建立会话。
   - 宿主负责保存 session_id 并在 Agent 执行时调用 `/invoke`，再根据结果推送 SSE。
3. 插件开发者无需在 PowerX 侧部署独立 MCP Server；每个插件实例都携带自己的 `/mcp/sse` 与 `/mcp/ws`，通过 Stream Broker fan-out 给订阅者。

## 5. 常见排障

| 现象 | 排查 | 解决方案 |
|------|------|----------|
| Register 返回 500 | `runtime_assignment_id` 为空或重复 | 确认 payload 并检查数据库 `runtime_ops_mcp_sessions` 是否成功写入 |
| Invoke 403 | Grant Matrix 未授权 | 更新 `capabilities/catalog.json` 中 `metadata.protocols.agent_tool.scope`，或在数据库覆盖 `grant_matrix_overrides` |
| SSE 无事件 | 会话未完成 ACK 或 Invoke 失败 | 查看 `/api/v1/admin/runtime/sessions/<id>` 日志；必要时开启 `POWERX_LOG_LEVEL=debug` |

> 建议：在 CI 中加入 `go test ./internal/services/admin/runtime_ops` 以及 `skeleton/backend/go-gin/tests/integration/mcp_agent_mode_test.go`，保证 MCP 与 REST Agent 可以同时调度同一能力。
