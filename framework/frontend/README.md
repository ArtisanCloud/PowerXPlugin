# PowerX Framework Frontend

此目录承载平台级前端运行时（Nuxt Layer、客户端桥接等）。

- `nuxt/framework-admin` 暴露管理端 Layer 与 `definePowerXAdminConfig` 辅助。
- `nuxt/framework-client` 提供 `$powerxApi` 等客户端封装。

插件在引用时统一使用已发布版本（当前建议 `^0.0.1-alpha`）。在 monorepo 内调试本地 Layer 时，可在执行 `px-plugin init` 前设置 `POWERXPLUGIN_USE_LOCAL_FRONTEND=1`，此时脚手架会写入 file: 引用；若 unset，则默认指向 npm 版本。开发工具、workspace 示例仍保留在 `sdk/`，但必须显式依赖此框架层。
