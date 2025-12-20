# Quickstart - PowerX 通用能力插件消费

1. **声明所需能力**
   - 在 `skeleton/plugin.yaml` 的 `requiredCapabilities` 中增加 CoreX 能力 ID（如 `com.corex.media.assets.manage`）。
   - 运行 `px-plugin capabilities plan --manifest ./skeleton/plugin.yaml`，确保 Registry 中存在并已授权。

2. **获取或刷新工具凭证**
   - 宿主模式：运维在部署时注入 `PX_GATEWAY_BASE_URL`、`PX_PLUGIN_TOOL_TOKEN`、`PX_TENANT_UUID`。
   - Skeleton 模式：执行 `px-plugin login --manifest ./skeleton/plugin.yaml`，并将生成的 `PX_GATEWAY_BASE_URL`、`PX_TOOL_TOKEN` 写入 `skeleton/.env.local`。

3. **初始化 Gateway Client**
   - Go backend：在 `packages/backend` 中调用 `powerxgateway.NewClient(cfg)`，并通过 DI 注入到需要调用 Core 能力的 Service。
   - Nuxt admin：在 `packages/admin/plugins/powerx-gateway.client.ts` 暴露 `usePowerXCapability()`，读取 runtimeConfig。

4. **发起调用**
   ```go
   resp, err := gateway.Invoke(ctx, integration.InvokePayload{
       CapabilityID: "com.corex.media.assets.manage",
       Action:       "Create",
       Payload: map[string]any{"assetName": "demo.pdf"},
   })
   ```
   ```ts
   const invoke = usePowerXCapability()
   const data = await invoke('com.corex.media.assets.read', 'List', { pageSize: 20 })
   ```

5. **观测与日志**
   - 框架自动记录 `capabilityId`、`tenantUUID`、`traceId`、耗时；若触发限流会写入 `rateLimitExceeded` 事件。
   - 如需手动验证，执行 `scripts/capabilities/run-from-package.mjs --manifest ./skeleton/plugin.yaml --cap com.corex.media.assets.manage --action Create`。

6. **Mock / 降级（Skeleton）**
   - 无法访问 Dev Gateway 时，运行 `npm run dev -- --use-mock=media` 或设置 `PX_USE_MOCK=media`，框架会返回内存数据并在日志中提示 Mock 模式。

7. **文档与速查**
   - 参考 `docs/plan/009-consume-powerx-capability.md` 与本 spec，了解对应能力动作、常见错误码及 Token 刷新指引。
