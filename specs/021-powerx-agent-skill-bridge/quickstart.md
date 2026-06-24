# Quickstart — PowerX Agent Skill Bridge Framework 对齐

## 1. 前置条件

1. 已切换分支：

```bash
git branch --show-current
# 预期：021-powerx-agent-skill-bridge
```

2. PowerX 底座已具备 Agent Session/Stream API。
3. 插件运行模式已配置 delegated 或 standalone dev。
4. Go 1.24 与前端开发环境可用。

## 2. 插件 Skill Manifest 示例

最小示例：

```json
{
  "skill_id": "mediax.video_rebuilder.cn",
  "provider": "com.powerx.plugin.mediax-studio",
  "version": "1.0.0",
  "title": "视频智能重构",
  "description": "根据视频链接和模板要求创建视频自动化重构任务",
  "intent_examples": [
    "帮我重构这个 shorts",
    "用篮球模板处理这个视频"
  ],
  "input_schema": {
    "type": "object",
    "required": ["urls"],
    "properties": {
      "urls": {
        "type": "array",
        "items": {"type": "string"}
      },
      "template_hint": {
        "type": "string"
      }
    }
  },
  "executor": {
    "type": "capability",        "capability": "creation.video_automation.ingest"
  }
}
```

## 3. Skill 发现验收

启动插件后执行：

```bash
curl -X GET "$PLUGIN_BASE_URL/api/v1/plugin/skills" \
  -H "Authorization: Bearer $PLUGIN_RUNTIME_TOKEN"
```

预期：

1. 返回 `items[]`。
2. 每项包含 `skill_id/provider/version/description/input_schema/executor`。
3. 重复 `skill_id + version` 或缺少必填字段时启动或接口返回明确错误。

## 4. Skill Schema 验收

```bash
curl -X GET "$PLUGIN_BASE_URL/api/v1/plugin/skills/mediax.video_rebuilder.cn/schema" \
  -H "Authorization: Bearer $PLUGIN_RUNTIME_TOKEN"
```

预期返回 input/output schema。

## 5. Executor 成功调用验收

```bash
curl -X POST "$PLUGIN_BASE_URLPowerX Capability Invocation" \
  -H "Authorization: Bearer $PLUGIN_RUNTIME_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "skill_id": "mediax.video_rebuilder.cn",
    "version": "1.0.0",
    "input": {
      "urls": ["https://example.com/video.mp4"],
      "template_hint": "篮球模板"
    },
    "context": {
      "tenant_uuid": "tenant_xxx",
      "user_uuid": "user_xxx",
      "agent_id": "agent_xxx",
      "session_id": "session_xxx",
      "message_id": "message_xxx",
      "skill_id": "mediax.video_rebuilder.cn",
      "trace_id": "trace_xxx",
      "channel": "web",
      "locale": "zh-CN",
      "capability": "creation.video_automation.ingest"
    }
  }'
```

预期：

```json
{
  "success": true,
  "skill_id": "mediax.video_rebuilder.cn",
  "status": "queued",
  "message": "已创建视频重构任务",
  "task_id": "task_xxx",
  "trace_id": "trace_xxx"
}
```

## 6. Fail-fast 验收

缺少 `tenant_uuid`：

```bash
curl -X POST "$PLUGIN_BASE_URLPowerX Capability Invocation" \
  -H "Authorization: Bearer $PLUGIN_RUNTIME_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "skill_id": "mediax.video_rebuilder.cn",
    "input": {"urls": ["https://example.com/video.mp4"]},
    "context": {
      "trace_id": "trace_missing_tenant"
    }
  }'
```

预期：

```json
{
  "success": false,
  "error": {
    "code": "skill.plugin_context_missing",
    "message": "tenant_uuid is required"
  }
}
```

capability 不匹配：

```json
{
  "error": {
    "code": "skill.plugin_capability_mismatch"
  }
}
```

## 7. Framework Client 调用 PowerX Agent SSE

本地 Chat 或后端调试入口应调用 PowerX：

```bash
curl -N -G "$POWERX_BASE_URL/api/v1/agents/stream/sse" \
  -H "Authorization: Bearer $POWERX_ACCESS_TOKEN" \
  --data-urlencode "agent_id=agent_xxx" \
  --data-urlencode "session_id=session_xxx" \
  --data-urlencode "q=帮我用篮球模板重构这个视频：https://example.com/video.mp4"
```

预期事件：

```text
event: intent
event: plan
event: node_start
event: node_end
event: final
event: end
```

Framework Client 应统一解码为：

```json
{
  "type": "node_start",
  "trace_id": "trace_xxx",
  "session_id": "session_xxx",
  "plan_id": "plan_xxx",
  "node_id": "node_xxx",
  "payload": {}
}
```

## 8. 插件调试 Chat 验收

打开插件调试 Chat 页面并发送：

```text
帮我用篮球模板重构这个视频：https://example.com/video.mp4
```

通过标准：

1. 页面请求 PowerX Agent API，不请求插件领域业务 API。
2. PowerX Agent Runtime 命中当前插件 Skill。
3. 插件 executor 收到完整 `PluginSkillInvocationContext`。
4. 页面展示 PowerX Agent Stream 的执行过程和最终结果。
5. 日志可按 `trace_id/session_id/skill_id/plugin_id` 串联。

## 9. 配置校验

delegated 模式必须存在：

```text
PX_GATEWAY_BASE_URL
PX_GATEWAY_AUTH_SCHEME=bearer
POWERX_STS_CLIENT_ID
POWERX_STS_CLIENT_SECRET
```

缺失任一项时，Framework Client 初始化必须失败。

禁止 delegated 模式读取：

```text
PX_TOOL_TOKEN
PX_GATEWAY_API_KEY
```

## 10. 回归命令建议

```bash
cd framework/backend/go
go test ./runtime/skills ./runtime/powerx/agent ./runtime/powerx/sts
```

Skeleton 验证：

```bash
cd skeleton/backend/go-gin
go test ./internal/skills ./internal/transport/http/plugin/skills
```

前端 E2E：

```bash
cd skeleton/web-admin
npm run test:e2e -- agent-skill-bridge
```

## 11. 本次实现回归记录（2026-06-08）

已执行：

```bash
cd framework/backend/go
go test ./runtime/skills ./runtime/powerx/agent ./runtime/powerx/sts

cd skeleton/backend/go-gin
go test ./internal/skills ./internal/transport/http/plugin/skills
```

结果：全部通过。前端 E2E 已新增 `skeleton/web-admin/nuxt/tests/e2e/agent-skill-bridge.spec.ts`，用于验证本地 Chat 请求 PowerX Agent SSE 且不直连插件 executor。
