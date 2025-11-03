# PowerX Framework Frontend

此目录承载平台级前端运行时（Nuxt Layer、客户端桥接等）。

- `nuxt/framework-admin` 暴露管理端 Layer 与 `definePowerXAdminConfig` 辅助。
- `nuxt/framework-client` 提供 `$powerxApi` 等客户端封装。

插件在引用时统一指向这里：

```json
"@powerx-plugin/framework-admin": "file:<repo-root>/framework/frontend/nuxt/framework-admin"
"@powerx-plugin/framework-client": "file:<repo-root>/framework/frontend/nuxt/framework-client"
```

开发工具、workspace 示例仍保留在 `sdk/`，但必须显式依赖此框架层。
