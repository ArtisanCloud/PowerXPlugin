# PowerX Plugin SDK (dev-kit)

当前目录预留给开发体验相关工具（CLI API 封装、Playwright helpers、workspace 示例等）。

Nuxt 平台 layer 已经迁移到 `framework/frontend/nuxt`，如果需要引用运行时能力，请依赖
对应的 framework 包：

```json
"@artisan-cloud/plugin-framework-admin": "^0.0.1-alpha",
"@artisan-cloud/plugin-framework-client": "^0.0.1-alpha"
```

默认从 npm 获取已发布版本。若需在 monorepo 内引用本地源码，可在运行 `px-plugin init` 之前设置环境变量 `POWERXPLUGIN_USE_LOCAL_FRONTEND=1`。

后续在补充新的 SDK 工具时，将以独立子模块或 workspace 形式放在该目录下。
