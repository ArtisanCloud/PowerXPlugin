# WebSocket 联调手册（插件侧：Host / Standalone / Proxy）

## 1. 目标

用同一协议验证三种模式：

1. Host（直连底座）
2. Standalone（插件本地 WS）
3. Standalone Proxy（插件转发到底座）

本手册只覆盖 WS 链路（`grant/publish/subscribe`），不覆盖业务 `emit` 主链路。

## 2. 前置条件

1. 后端默认强鉴权（必须携带有效凭证）
2. topic 已在插件声明层与执行层同步：
   - 声明层：`plugin.yaml.events.topics[]`
   - 执行层：`config/event_fabric.yaml`
3. topic 已在底座 `event_topics` 存在，且主体具备权限
4. 模式明确：
   - standalone：`POWERX_PROXY=0`
   - standalone + proxy：`POWERX_PROXY=1`

## 3. 连接地址矩阵

1. Host（底座）：`ws://127.0.0.1:8077/api/ws?authorization=Bearer%20$HOST_TOKEN`
2. Standalone（插件）：`ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN`
3. Standalone Proxy（插件入口）：`ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN`

## 4. 协议消息

客户端：`subscribe` / `unsubscribe` / `ping`  
服务端：`welcome` / `ack` / `error` / `event`

## 5. standalone（`POWERX_PROXY=0`）

### Step 1：连接并订阅插件 WS

```bash
wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN"
```

```json
{"type":"subscribe","topics":["_topic.template.update"]}
```

### Step 2：插件 publish

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/internal/ws-bus/publish \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"id":"demo","progress":25}}'
```

预期：先 `ack`，再收到 `event`。

## 6. standalone + proxy（`POWERX_PROXY=1`）

### 标准流程（插件侧入口）

1. 插件入站调试都打 `:8078`（Bearer）。
2. 插件出站到底座按 `gateway.auth_scheme` 选择凭证（推荐 ApiKey）。
3. 先通过插件接口创建 topic（插件代理到底座 `admin/event-fabric/topics`）。
4. 再执行 `grant`（插件代理到底座 `internal/ws-bus/grant`，仅绑定 ACL）。
5. 最后执行 `publish`，并在 WS 连接上验证收到 `event`。

### Step 0：准备 proxy 凭证

1. 在 PowerX 准备好带目标 topic 权限的 profile（`permission_ids`）。
2. 基于该 profile 创建/轮换 API Key（权限为快照，profile 变更后必须换 key）。
3. 配置插件出站凭证：

```bash
export PX_GATEWAY_AUTH_SCHEME=apikey
export PX_GATEWAY_API_KEY=<your_api_key>
```

### Step 1：通过插件接口创建 topic（打 8078）

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/internal/event-fabric/topics \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "namespace":"_topic.template",
    "name":"update",
    "payload_format":"json",
    "versioning_mode":"backward",
    "max_retry":5,
    "ack_timeout_seconds":30
  }'
```

预期：`success=true`；如果 topic 已存在，底座返回冲突也可视为已满足前置条件。

### Step 2：插件 grant（打 8078）

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/internal/ws-bus/grant \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topics":["_topic.template.update"]}'
```

### Step 3：连接 WS 并订阅（打 8078）

```bash
wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN"
```

```json
{"type":"subscribe","topics":["_topic.template.update"]}
```

### Step 4：插件 publish（打 8078）

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/internal/ws-bus/publish \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"id":"demo","progress":25}}'
```

### Step 5：验收

1. 插件日志应看到：
   - `gateway_auth_scheme=apikey`
   - `outbound_token_source=PX_GATEWAY_API_KEY`
2. 订阅端先收到 `ack`
3. 执行 Step 4 后收到 `event`

## 7. 快速排障

1. `404`：路径不是 `/api/ws`
2. `401`：凭证无效或插件出站仍在走 Bearer
3. `403 topic not allowed`：profile/ACL 未授权该 topic
4. 只有 `ack` 无 `event`：topic 不一致、未先 `grant`，或权限快照未轮换
5. `grant/publish` 失败且提示 topic 不存在：先走 `internal/event-fabric/topics` 创建 topic

## 8. 职责边界（必须遵守）

1. 插件本体：只声明“需要哪些 topic + actions”。
2. Proxy：负责“topic 注册、profile 权限绑定、key 轮换、grant/publish 调用”。

## 9. 避免 403 的硬规则

1. topic 必须先存在于 `event_topics`。
2. API Key 快照权限必须覆盖该 topic。
3. 权限变更后必须使用轮换/新建后的新 key。
4. `grant` 不创建 topic，只做授权绑定。
