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

## 2.1 日志落点（先确认再联调）

1. 两种模式日志都在“插件后端进程”输出（stdout/stderr），不是默认写到底座日志。
2. 推荐统一落盘：

```bash
export RUNTIME_LOG_FILE=./tmp/runtime-backend.log
POWERX_PROXY=0 ./backend/cmd/plugin/plugin 2>&1 | tee "$RUNTIME_LOG_FILE"
# 或
POWERX_PROXY=1 PX_GATEWAY_BASE_URL=http://127.0.0.1:8077 ./backend/cmd/plugin/plugin 2>&1 | tee "$RUNTIME_LOG_FILE"
```

3. 容器场景可用：

```bash
docker logs -f <container_name> | tee "$RUNTIME_LOG_FILE"
```

## 3. 连接地址矩阵

1. Host（底座）：`ws://127.0.0.1:8077/api/ws?authorization=Bearer%20$HOST_TOKEN`
2. Standalone（插件）：`ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN`
3. Standalone Proxy（宿主内嵌）：`ws://127.0.0.1:3030/api/ws?authorization=Bearer%20$USER_TOKEN`

> 注意：宿主内嵌调试必须连接宿主 `/api/ws`，不要连接 `/_p/<plugin>/api/ws`。

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
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/ws-bus/publish \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"id":"demo","progress":25}}'
```

预期：先 `ack`，再收到 `event`。

### Step 3：通知探针（可选但推荐）

用统一探针验证 UI 是否真的消费到事件（而不是仅靠发送后主动拉取）。

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/notifications/test \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"message":"ws probe"}'
```

预期：右上角通知铃铛未读角标增加，通知面板出现新事件。默认 topic 为 `_topic.notify.tenant.{tenant_uuid}`。

## 6. standalone + proxy（`POWERX_PROXY=1`）

### 标准流程（插件侧入口）

1. 插件入站调试都打插件 API（Bearer）。
2. 插件出站到底座按 `gateway.auth_scheme` 选择契约凭证（Bearer 或 ApiKey）。
3. 先通过插件接口创建 topic（插件代理到底座 `admin/event-fabric/topics`）。
4. 再执行 `grant`（插件代理到底座 `admin/runtime/ws-bus/grant`，仅绑定 ACL）。
5. 最后执行 `publish`，并在 WS 连接上验证收到 `event`。

### Step 0：准备出站凭证

1. 宿主模式下确认 PowerX 已注入 STS client 与 gRPC upstream 环境变量。
2. standalone 本地联调如需 ApiKey，基于带目标 topic 权限的 profile 创建/轮换 API Key。
3. standalone ApiKey 配置：

```bash
export PX_GATEWAY_AUTH_SCHEME=apikey
export PX_GATEWAY_API_KEY=<your_api_key>
```

### Step 1：通过插件接口创建 topic（打 8078）

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/event-fabric/topics \
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
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/ws-bus/grant \
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
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/ws-bus/publish \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"_topic.template.update","payload":{"id":"demo","progress":25}}'
```

### Step 5：验收

1. 插件日志应看到：
   - `gateway_auth_scheme` 与你的配置一致（`bearer` / `apikey`）
   - `outbound_token_source` 与凭证来源一致（如 `sts` / `PX_GATEWAY_API_KEY`）
   - 不再出现 `outbound_token_source=request_bearer_passthrough`
2. 订阅端先收到 `ack`
3. 执行 Step 4 后收到 `event`

## 7. 快速排障

1. `404`：路径不是 `/api/ws`
2. `401`：凭证无效，或 `PX_GATEWAY_AUTH_SCHEME` 与实际提供的凭证不匹配
3. `403 topic not allowed`：profile/ACL 未授权该 topic
4. 只有 `ack` 无 `event`：topic 不一致、未先 `grant`，或权限快照未轮换
5. `grant/publish` 失败且提示 topic 不存在：先走 `event-fabric/topics` 创建 topic
6. 铃铛显示“已连接”但无通知：检查是否订阅了 `_topic.notify.tenant.{tenant_uuid}`，并确认 token 中 `tid` 与 publish 租户一致

## 8. 职责边界（必须遵守）

1. 插件本体：只声明“需要哪些 topic + actions”。
2. Proxy：负责“topic 注册、profile 权限绑定、key 轮换、grant/publish 调用”。

## 9. 避免 403 的硬规则

1. topic 必须先存在于 `event_topics`。
2. API Key 快照权限必须覆盖该 topic。
3. 权限变更后必须使用轮换/新建后的新 key。
4. `grant` 不创建 topic，只做授权绑定。

## 11. 页面统一联调（framework 功能测试页）

1. 页面按钮统一调用：`POST /api/v1/admin/runtime/ws-bus/test-flow`
2. 页面诊断关键字段：
   - `Grant/Publish`
   - `flow_mode`（如 `host_plus_local_echo`）
   - `echo_ok`
   - `diag.sub_sent / ack_ok / event_ok`
3. 判断打通标准：
   - `Grant=ok`
   - `Publish=ok`
   - `ack_ok=true`
   - `event_ok=true`

## 10. Proxy 权限失败闭环（US3 验收）

目标：形成“有限重试 -> 超限工单 -> 暂停 -> 运维恢复 -> 再触发”的统一流程。

### Step 1：构造权限失败并触发重试

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/dispatches/dispatch-us3-001/retry \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"error_code":"AUTH_FORBIDDEN","error_message":"topic not allowed"}'
```

预期：前两次返回 `202`，第三次返回 `409`。

### Step 2：暂停并创建恢复工单

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/dispatches/dispatch-us3-001/pause \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"paused_job_id":"job-us3-001"}'
```

预期：返回 `201`，拿到 `ticket_id`。

### Step 3：校验恢复权限边界

```bash
# 非 ops/admin（应失败）
curl -sS -X POST "http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/tickets/$TICKET_ID/resume" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"operator_role":"viewer","operator_id":"qa-user","reason":"try-resume"}'

# ops/admin（应成功）
curl -sS -X POST "http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/tickets/$TICKET_ID/resume" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"operator_role":"ops","operator_id":"ops-user","reason":"permission fixed"}'
```

预期：第一条 `403`，第二条 `200` 且 `ticket_status=resolved`。

### Step 4：恢复后再触发

恢复成功后再次执行 retry，预期回到 `202`（重试窗口被重置），并可继续按标准 WS 联调流程验证 `ack -> event`。
