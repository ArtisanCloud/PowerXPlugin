# Framework 2026-10-11 发布内容

| 交付 | 版本 |
|---|---|
| Go Framework | v0.0.24 |
| Nuxt framework-client | 0.0.12 |
| Nuxt framework-admin | 0.0.11 |

Go 版本包含 IAM 姓名解析的凭据租户核验、完整租户 DTO 的数字状态转换、正式请求身份与 WS 身份复核 SDK，以及已具备合同测试的知识库创建、配置快照、重建和语义检索适配。暂停验收的内容导览适配不包含在本次版本中。

Nuxt client 使用 Core iframe bridge v2，统一验证后的会话状态、结构化鉴权错误、Router 导航和 WS/SSE 会话绑定。只对明确的用户 token 到期错误请求新宿主会话；服务凭据、身份票据、403 和上游故障保留原会话边界。

Nuxt admin 的 Scheduler composable 改为从正式 client 包导入，并声明 client ^0.0.12 的 peer dependency，修复 tarball 安装后无法解析仓库相对路径的问题。

升级还需检查插件自身的接线：复制到插件内的鉴权逻辑不会随依赖升级自动替换。接入步骤见 [iframe bridge v2 指南](../guides/features/plugin-session-bridge-v2/guide.md) 和 [请求身份合同](../contracts/plugin-request-identity-framework.md)。SDK 合同测试与包构建通过不代表某个部署环境的浏览器登录、授权撤销和知识库业务流程已验收。

发布准备已验证：Go 全模块测试；IAM、Realtime、Knowledge 的 race 测试；CLI 全模块测试；Nuxt 客户端相关 90 个测试；两个包的 tarball 打包、安装及 Nuxt 4.2.0 生产构建；模板同步检查。

Go tag 使用 framework/backend/go/v0.0.24；npm 发布的默认渠道为 latest。远端发布和完整性核验结果单独记录在 [发布证据](../contracts/evidence/framework-release-20261011.json)。

Go 与两个 npm 包均已发布并完成远端核验；npm latest 分别指向 client 0.0.12 和 admin 0.0.11。通过公开 registry 安装的 27 个包文件与发布提交一致，独立 Nuxt 生产构建通过。
