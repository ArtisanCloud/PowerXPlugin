# Router 导航验收

## 目标

Core 发送当前插件完整挂载路径。Framework 去掉 Nuxt app.baseURL，再让实际 Router 执行，并将完成 Promise 交还 Core。Core 比较实际地址后确认；不能用整页 location 或 iframe src 修改代替菜单导航。

## 接入步骤

1. **配置挂载点**：安装构建设置 `NUXT_APP_BASE_URL=/_p/<plugin-id>/admin/`。观察 Nuxt app.baseURL 与当前 frame 地址一致；不一致时修正构建／启动配置。
2. **注册 Router**：使用 `createPowerXRouterNavigator(router, config.app.baseURL, window.location.origin)` 作为桥接的 navigator，获得取消订阅方法。预期 bridge:ready 的 navigationReady 为 true；缺失时 Core 明确报告不支持。
3. **检查取消**：新导航／退出会中止旧 signal。适配器在异步 Router 守卫之后再次检查，并卸载临时 beforeResolve 守卫；页面自有异步任务也需遵守取消信号或共享会话。
4. **释放生命周期**：Vue 应用卸载、HMR 或组件适配器处置时取消认证及 Router 注册。旧 adapter 不得清除新 adapter 的消费状态。

接口示例（注册在插件应用中）：

```ts
const bridge = initPowerXBridge({
  pluginId,
  onAuthToken: message => session.apply(message, contextAdapter),
  navigator: createPowerXRouterNavigator(router, config.app.baseURL, window.location.origin),
});
// 在对应应用／组件生命周期结束时调用 bridge.stop()。
```

## 页面操作

| 动作与入口 | 成功判定 | 失败处理 |
| --- | --- | --- |
| 在 Core 插件菜单中由介绍页进入模板 CRUD，再回介绍页 | Router 路由切换，frame 文档未重载，认证状态仍对应当前用户 | 核对 actual path 与 navigation:result，不重置 src |
| 直接打开 `/_p/com.powerx.plugins.base/admin/admin/templates/framework-lab?tab=customer#details` | 内部路由为 `/admin/templates/framework-lab?tab=customer#details`，无重复挂载前缀 | 检查 app.baseURL、query、hash 和 Root 权限 |
| 快速点击两个菜单，或在异步守卫期间退出 | 新导航胜出，旧操作被取消，不回到旧页面 | 查看 requestId 与 signal，检查临时守卫处置 |
| 使用浏览器后退／前进 | 实际 Router／历史与当前页面一致 | 真实浏览器验收，不能仅依赖 memory history |
| 发送其他插件或外部地址 | 明确拒绝，不打开其他地址 | `PLUGIN_NAVIGATION_INVALID` |

## 自动化与边界

合同测试使用真实 Vue Router memory history，验证 encoded query、hash 和 Core navigation:result；也覆盖异步守卫取消。真实 history、首次刷新和图片／下载仍须浏览器验证，当前状态见 [总览](guide.md)。
