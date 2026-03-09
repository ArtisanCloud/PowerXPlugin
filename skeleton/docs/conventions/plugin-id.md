# Plugin ID 命名规范（统一版）

## 目标
统一插件 ID、路由前缀、Audience、前端桥接配置，避免出现 `/_p/...` 路径错配导致的 404/白屏。

## 规范
- 插件 ID 必须使用：`com.powerx.plugins.<name>`
- 禁止使用旧前缀：`com.powerx.plugins.<name>`
- `<name>` 建议仅使用小写字母、数字和中划线/下划线。

## 必须对齐的位置
- `plugin.yaml`
  - `id`
  - `runtime.env.POWERX_PLUGIN_ID`
  - `runtime.env.POWERX_SECURITY_JWT_AUDIENCE`（形如 `plugin:<pluginId>`）
  - `frontend.admin.routes.*` 中的 `/_p/<pluginId>/admin/...`
- `make-files/common.mk`
  - `PLUGIN_ID` 默认值
- `web-admin/nuxt`
  - `nuxt.config.ts` 默认 `pluginId`
  - bridge 相关默认值（如 `useHostBridgeAdapter.ts`、`powerx-bridge.ts`）
  - 任何硬编码 `_p/<pluginId>/...` 页面路径
- 示例/测试/文档中的插件 ID 示例

## 验证
执行：

```bash
make plugin-id-check
```

检查项：
- 禁止残留 `com.powerx.plugins.`
- 校验 `plugin.yaml` 的 `id` 以 `com.powerx.plugins.` 开头

## 迁移建议
1. 先改 `plugin.yaml` 的 `id`
2. 全局替换代码中的旧前缀
3. 重新构建并重装插件
4. 用 `make plugin-id-check` 与页面联调验证
