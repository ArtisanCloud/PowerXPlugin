# PowerX Framework Frontend

此目录承载平台级前端运行时（Nuxt Layer、客户端桥接等）。

- `nuxt/framework-admin` 暴露管理端 Layer 与 `definePowerXAdminConfig` 辅助。
- `nuxt/framework-client` 提供 `$powerxApi` 等客户端封装。
- `nuxt/framework-client/scheduler.ts` 提供统一 Scheduler client，页面通过它调用插件侧 `/admin/runtime/scheduler/jobs` 管理接口。
- `nuxt/framework-admin/layer/app/composables/usePowerXScheduler.ts` 提供 Nuxt 管理端 composable，供插件页面统一创建、查询、暂停、恢复与手动触发 scheduler job。

插件在引用时统一使用已发布版本（当前建议 `^0.0.1-alpha`）。在 monorepo 内调试本地 Layer 时，可在执行 `px-plugin init` 前设置 `POWERXPLUGIN_USE_LOCAL_FRONTEND=1`，此时脚手架会写入 file: 引用；若 unset，则默认指向 npm 版本。开发工具、workspace 示例仍保留在 `sdk/`，但必须显式依赖此框架层。

## Scheduler Client

页面不直接拼接 PowerX 底座 SchedulerService 地址，也不直接调用 `/admin/event-fabric/cron/jobs`。统一使用 frontend framework scheduler client：

```ts
import { createSchedulerClient, usePluginApi } from "@artisan-cloud/plugin-framework-client"

const api = usePluginApi({
  pluginId: "com.powerx.plugins.ai-craft",
  tenantUuid
})
const scheduler = createSchedulerClient(api)

await scheduler.createJob({
  tenant_uuid: tenantUuid,
  owner_type: "plugin",
  owner_id: "com.powerx.plugins.ai-craft",
  name: "sample_progress_50",
  schedule_type: "once",
  schedule_expr: eta50,
  payload: {
    business_action: "sample_progress_50",
    order_id: orderId
  }
})
```

Nuxt 管理端页面可使用 Layer composable：

```ts
const scheduler = usePowerXScheduler({
  pluginId: "com.powerx.plugins.ai-craft",
  tenantUuid
})

await scheduler.listJobs()
await scheduler.pauseJob(jobId)
await scheduler.resumeJob(jobId)
await scheduler.triggerJob(jobId)
```
