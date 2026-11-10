# Framework 只读连接探测交付（2026-10-29）

AI 设置页已将“测试连接”与“试跑一次”分开。测试连接只验证服务入口和可支持的鉴权，不发送提示词、不调用模型、不创建任务、不保存配置；试跑才使用启动时选定的业务运行通道执行实际模型调用。

## 当前行为

| 操作 | 前置条件与结果 |
|---|---|
| PowerX 来源测试连接 | 后端已配置 Core 客户端，供应商目录已加载且选定供应商；不要求选定模型或填写提示词。使用启动时配置的 API Key／STS；本地业务运行模式也可显式探测 Core |
| 本地来源测试连接 | 仅支持 Ollama，要求连接地址；只读 GET `/api/tags`，不要求模型。其他供应商明确禁用／返回不支持 |
| 试跑一次 | 继续要求模型、输入、模态能力和业务通道；PowerX 模型在本地业务通道中仍禁用试跑，并说明原因 |
| 展示结果 | 连接结果与分析／任务／文本试跑结果分别保存；关闭面板、切换分区后保留上下文；连接成功不代表具体模型具有调用权限或生成能力 |

Core 的供应商目录可读但模型发现失败时，仍可执行连接探测。供应商目录不可读时明确显示目录错误。Ollama 成功显示“未验证 API Key”；需要 API Key 的供应商须收到 Core 明确的鉴权验证结果。展示 Core 返回的实际探测环境 `env`，不能把页面环境选择当作 Core 环境覆盖。

## SDK 与请求合同

Go 新增独立 `aisettings.ConnectionTester` 接口及 `Client.TestConnection(ctx, ConnectionTestInput, requestID)`，保留既有设置／生成接口。Nuxt Client 导出 `createAIConnectionClient`、输入和结果类型。

PowerX 调用固定能力 `com.corex.ai.connection.test`，协议 `core_internal`、方法 `INVOKE`、端点 `core://ai/connection`、actor 为 `service_actor`。通过配置 API 前缀下的 `/tenant/invocations` 调用。只接受 `contract_version: 1`、`operation: test_connection`、`modality`、`provider` 和可选原生 `model`；没有凭据、连接地址、环境或生成输入覆盖。

插件页面桥接 `POST admin/ai-settings/test-connection` 的 PowerX 请求现在为：

```json
{"source":"powerx","modality":"llm","provider":"ollama"}
```

本地请求另带 `environment: development|production` 和 `endpoint`。原先把 `inputs`、参数或幂等键传给 TestConnection 的方式已移除；这些字段属于 `quick-call`。页面与对应后端须同时更新，不能只刷新旧页面或只替换 SDK。

SDK 要求网关状态 completed、实际协议 core_internal、显式 fallback_used=false；必需布尔值缺失、身份不匹配或声称已生成均返回合同错误。HTTP 状态、机器码、details、trace、request_id 和原始上游 body 保留。失败不自动重试、不切换凭据或通道、不请求审批。

Skeleton 的 `plugin.d/capabilities.yaml` 和两套生成模板已声明 `com.corex.ai.connection.test`。此声明不授予权限。API Key 仍需 Core 的 `_scope.ai.connection.test`／`test`／`api`／`ai-connection-test` grant；安装态 STS 的 allowed_capabilities 必须精确包含该能力。完整权威定义见 [Core 合同](ai-settings-connection-test.md)和[来源摘要](ai-connection-test-source.json)。

## 可消费开发包

以下为当前未提交源码生成的本地开发包，未公开发布；基线提交 `32aaf0bd33ecda9860384484b55aa1d092ad27fc` 本身不包含本次改动。包目录中的 20261022 是既有打包系列标识，本次构建日期为 2026-10-29。

| 产物 | 精确引用 |
|---|---|
| Go SDK | `v0.0.27-dev.aisettings.a64914411aceb494` |
| Nuxt Client | `0.0.14-dev.aisettings.30b5d140eecf02d8` |
| CLI | `v0.0.0-aisettings.ed21c0b2f7384ffa` |
| 引用与摘要 | [references.json](../../artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/references.json) |
| 独立生成插件 | [generated-consumer](../../artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/generated-consumer) |

Go 模块为 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`，本地 GOPROXY 为 `file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/go-proxy,https://proxy.golang.org,direct`，GONOSUMDB 使用该模块路径。Nuxt 安装引用为 `file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/artisan-cloud-plugin-framework-client-0.0.14-dev.aisettings.30b5d140eecf02d8.tgz`。本地生成器为 `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/px-plugin`；生成时显式传入 `--framework-version v0.0.27-dev.aisettings.a64914411aceb494` 和 `--framework-client-ref 'file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-81226a83a8ade24b/artisan-cloud-plugin-framework-client-0.0.14-dev.aisettings.30b5d140eecf02d8.tgz'`。

实际消费使用 `GOWORK=off`，没有 Framework replace；生成插件已通过清单、后端构建、SDK 合同回归和打包 TypeScript 检查。生成产物实际包含三份 Core 合同及来源摘要，内容与 Core 一致。现有公开 SDK／CLI 尚未包含这次改动；默认旧版本引用不能代替上述匹配引用。已有业务插件还须同步自己的页面、后端桥接及 locale，SDK 升级不会覆盖插件自有代码。

## 验证与本地联调

[验收记录](evidence/ai-connection-test-framework-validation-20261029.json)记录了 7 个 Go 包／153 个测试事件、4 个包 race 和 vet、14 个前端单元用例、20 个浏览器场景、严格类型检查、Nuxt 构建、合同和模板一致性及实际生成插件消费。浏览器使用受控 API；API Key／STS 使用真实 SDK 经过受控 HTTP 服务，不能代替真实安装态。8 种主题／屏宽组合检查了手动主题与系统主题不一致、按钮默认／悬停／聚焦／禁用、文字对比度及横向溢出。

全量 `ci-manifest-align` 检查遇到已有 `parcel.volume` 描述格式与同步器不一致的问题，会删去其既有 exposure／schema 引用。已撤回这些无关生成改动，仅保留本次 required 能力；生成插件的 `manifestcheck --plugin-only` 通过。该全量守卫未标记为通过。

本轮没有重启用户的 8077 Core、8078 插件后端或 3131 Nuxt，没有修改 Core／grants。自建 3157 预览进程及进程组已关闭，端口释放。真实联调按以下顺序进行：

1. 让插件后端加载本次新代码，前端使用匹配页面和 Client；Core 使用包含新能力的构建。
2. 由 Core 检查能力发布、租户 registration，以及现有 API Key 的最小 grant／安装态 STS 能力清单。
3. 页面选 PowerX 来源与供应商，先不填提示词、不选模型，点测试连接；检查实际环境和鉴权结果，并确认没有模型任务。权限拒绝应显示原机器码和 trace，人工修复授权后再重试。
4. 分别验证 API Key Host 与安装态 STS。需要实际试跑时，再使用具备相应能力和通道的配置，单独点击试跑。

| 范围 | 完成情况 |
|---|---|
| Go、Nuxt Client、Skeleton、清单与 CLI 模板 | 源码和受控消费验证完成 |
| .NET 连接探测适配 | 未实现，不计入本次交付 |
| 真实 Core／供应商、安装态 STS | 待用户运行环境联调 |
| 公共版本、提交／发布 | 未执行 |
