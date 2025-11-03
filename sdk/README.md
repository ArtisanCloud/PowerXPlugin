# PowerX Plugin SDK (dev-kit)

当前目录预留给开发体验相关工具（CLI API 封装、Playwright helpers、workspace 示例等）。

Nuxt 平台 layer 已经迁移到 `framework/frontend/nuxt`，如果需要引用运行时能力，请依赖
对应的 framework 包：

```json
"@powerx-plugin/framework-admin": "file:../../framework/frontend/nuxt/framework-admin"
"@powerx-plugin/framework-client": "file:../../framework/frontend/nuxt/framework-client"
```

后续在补充新的 SDK 工具时，将以独立子模块或 workspace 形式放在该目录下。
