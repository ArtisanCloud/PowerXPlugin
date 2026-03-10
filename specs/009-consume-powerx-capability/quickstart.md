# Quickstart - PowerX 通用能力插件消费

1. **声明所需能力**
   - 在 `skeleton/plugin.yaml` 的 `requiredCapabilities` 中增加 CoreX 能力 ID（如 `com.corex.media.assets.manage`）。
   - 运行 `px-plugin capabilities plan --manifest ./skeleton/plugin.yaml`，确保 Registry 中存在并已授权。

2. **获取或刷新工具凭证**
   - 宿主模式：运维在部署时注入 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`（租户由 token `tid` 推导）。
   - Skeleton 模式：执行 `px-plugin login --manifest ./skeleton/plugin.yaml`，并将生成的 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN` 写入 `skeleton/.env.local`。

3. **初始化 Gateway Client**
   - Go backend：在 `packages/backend` 中调用 `powerxgateway.NewClient(cfg)`，并通过 DI 注入到需要调用 Core 能力的 Service；同时在插件后端提供受控 API（如 `/api/powerx/capabilities/invoke`）供前端复用。
   - Nuxt 前端：通过 `runtimeConfig.public.powerx` 读取 `apiBase` 与 `capabilityEndpoint`（默认 `/_p/<plugin>/api/v1` + `/integration/capabilities/invoke`，可通过 `NUXT_PUBLIC_POWERX_API_BASE`/`NUXT_PUBLIC_POWERX_CAPABILITY_ENDPOINT` 覆盖），使用仓库内置的 `powerx-capability.client.ts`/`usePowerXCapability` 代理调用插件后端。

4. **发起调用（带完整协议描述）**
   ```go
   resp, err := gateway.Invoke(ctx, integration.InvokeRequest{
       CapabilityID:      "com.corex.media.assets.manage",
       Action:            "Create",
       PreferredProtocol: "rest",
       Payload: map[string]any{
           "method":   "POST",
           "endpoint": "/api/v1/media/assets",
           "headers":  map[string]any{"Content-Type": "application/json"},
           "body": map[string]any{
               "assetName":    "demo.pdf",
               "uploadMethod": "presign_upload",
           },
       },
   })
   ```
   ```ts
   const data = await $fetch('/api/powerx/capabilities/invoke', {
     method: 'POST',
     body: {
       capabilityId: 'com.corex.media.assets.read',
       action: 'List',
       preferredProtocol: 'rest',
       payload: {
         method: 'GET',
         endpoint: '/api/v1/media/assets',
         query: { page: 1, page_size: 20 }
       }
     }
   })
   ```
   > `action` 仅用于区分语义或供 Gateway 做审计，与最终 URL 无直接关联；真正的路由依据是 `payload` 中的协议细节。

5. **观测与日志**
   - 框架自动记录 `capabilityId`、`tenantUUID`、`traceId`、耗时；若触发限流会写入 `rateLimitExceeded` 事件。
   - 如需手动验证，执行 `scripts/capabilities/run-from-package.mjs --manifest ./skeleton/plugin.yaml --cap com.corex.media.assets.manage --action Create`。

6. **Mock / 降级（Skeleton）**
   - 无法访问 Dev Gateway 时，运行 `npm run dev -- --use-mock=media` 或设置 `PX_USE_MOCK=media`，框架会返回内存数据并在日志中提示 Mock 模式。

7. **Capability Lab 调试**
   - Skeleton web-admin（Root）> “开放能力调试”可一键访问 `/powerx/capability-lab` 页面，页面会代理调用插件后端 `/api/v1/integration/capabilities/invoke`。
   - 能力下拉菜单会请求 `/api/v1/admin/capabilities?source=corex`，后端再透过 Gateway 查询 PowerX `/tenant/capabilities`，因此列表与底座开放能力保持同步；未加 `source` 时则继续返回本地 manifest 供内部页面使用。
   - 若在“Mock 模块”输入 `media` 等值，会自动附加 `X-PX-Use-Mock` header 并在响应 `warnings` 中提示 Mock 状态，便于在 Gateway 不可用时复现。
   - 详见 `docs/guides/develop/consume-powerx-capability/README.md#6-skeleton-能力调试页（Capability Lab）`。

8. **文档与速查**
   - 参考 `docs/plan/009-consume-powerx-capability.md` 与本 spec，了解对应能力动作、常见错误码及 Token 刷新指引。
