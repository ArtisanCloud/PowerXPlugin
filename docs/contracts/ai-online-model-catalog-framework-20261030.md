# Framework 本地在线模型目录交付（2026-10-30）

截图中的「插件本地配置」已补齐在线获取、候选分类、持久化和新型号保存链路；PowerX 来源新增候选列表刷新及可配置的管理页入口。更新目录、连接探测、实际试跑保持独立，目录更新不会更换活动模型或写入表单参数。

## 已实现

- 公共 Go `runtime/ai/modeldiscovery` 对齐 Core 的 11 个 Provider 协议入口，保留原生型号及 tag、分页、明确能力、超时和上游状态／trace，拒绝重定向、残缺结果和隐式重试。
- 本地快照按租户、环境、Provider、App 隔离，绑定地址与凭据身份；支持带 revision 的人工分类、并发比较并交换、失败保留旧值和连接变化失效。有效新候选能被选择、保存并由显式 Ollama 试跑使用。
- 服务端通过 `local_ai.credentials` 的显式 UUID 引用解析环境变量中的 Key，匹配租户／环境／供应商／地址，不借用 Core Key。浏览器只接收公开地址默认值，不接收原始密钥。
- Nuxt 模型旁增加「更新在线模型」；弹窗按已匹配／待确认分区，显示地址、协议、更新时间和能力。错误关闭弹窗后仍可见；切换范围及迟到请求不会混入旧列表。没有在线协议的供应商继续使用已声明的静态配置，更新操作明确不可用。
- Skeleton、两套 CLI 模板、主配置示例、内嵌合同导出、Schema、fixture 及生成产物验证已同步。

正式操作、配置和接口见 [本地合同](ai-local-model-catalog.v1.md)、[OpenAPI](ai-local-model-catalog.v1.openapi.yaml)和[共同样例](fixtures/ai-local-model-catalog.v1.json)。Core 在线同步仍只允许用户管理员；Framework 的 PowerX 刷新只读取既有 typed 候选，不调用 Admin 同步路由。相关 Core 正文与[来源摘要](ai-online-model-catalog-source.json)已同步。

## 可消费开发包

本次改动尚未提交、未公开发布；基线 `32aaf0bd33ecda9860384484b55aa1d092ad27fc` 本身不包含本次实现。以下包为当前工作区构建的匹配开发包；目录中的 20261022 是既有打包系列名称，本次构建日期为 2026-10-30。

| 产物 | 精确引用 |
|---|---|
| Go SDK | `v0.0.27-dev.aisettings.a74d13887f3cd123` |
| Nuxt Client | `0.0.14-dev.aisettings.9d2245a4ce0866ac` |
| CLI | `v0.0.0-aisettings.627bff290d0068f6` |
| 完整引用与摘要 | [references.json](../../artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/references.json) |
| 独立生成插件 | [generated-consumer](../../artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/generated-consumer) |

Go 模块为 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`，GOPROXY 为 `file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/go-proxy,https://proxy.golang.org,direct`，GONOSUMDB 使用该模块路径。Nuxt 包引用为 `file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/artisan-cloud-plugin-framework-client-0.0.14-dev.aisettings.9d2245a4ce0866ac.tgz`。生成器为 `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/px-plugin`；生成时显式使用 `--framework-version v0.0.27-dev.aisettings.a74d13887f3cd123` 和 `--framework-client-ref 'file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-cc87e64fe4ca10be/artisan-cloud-plugin-framework-client-0.0.14-dev.aisettings.9d2245a4ce0866ac.tgz'`。

实际生成消费使用 `GOWORK=off`，没有 Framework replace。生成插件清单、后端构建、公共 Go SDK 真实解码／协议回归、合同副本及打包 TypeScript 检查均通过。已有插件还须同步自己的设置页、后端桥接、配置与迁移入口；只换 SDK 不会自动覆盖插件自有页面。当前公开 SDK／CLI 的默认版本不等于这个未发布开发包。

## 本地运行步骤

1. 在插件 Go 后端目录，使用实际配置路径执行单项迁移：`CONFIG_PATH=/path/to/config.yaml go run ./cmd/database migrate-ai-online-catalog`。该命令仅创建目录表／RLS，并将型号字段扩展到 512，不 seed 或重写活动配置。
2. 按原方式让插件后端加载新代码，前端加载匹配的 Client、页面和 locale。若未迁移，页面会明确提示，目录更新与保存禁用；独立连接探测仍可使用。
3. 选择「插件本地配置」和供应商，填写地址及必要的服务端凭据引用，点「更新在线模型」。确认目录与能力后，再明确选择型号并保存。
4. 如需 PowerX 管理站跳转，设置前端 `NUXT_PUBLIC_POWERX_CONSOLE_BASE`；未设置时显示原因，不猜测 3030 或从 API 端口推导。

本轮没有对用户数据库执行迁移，也没有重启 8077／8078／3131／Ollama 等日常服务。只启动了隔离 PostgreSQL 和 3158 临时预览，均已关闭并确认端口释放。

## 验收与现场边界

[验收证据](evidence/ai-online-model-catalog-framework-validation-20261030.json)：10 个 Go 目标包中 9 个有测试且通过、1 个无测试但编译通过，共 178 个通过事件（含子用例）；4 个包 race、8 个 vet 目标、隔离 PostgreSQL race／RLS／迁移、18 个前端单元用例、36 个浏览器场景、严格类型检查、Nuxt 构建、合同／模板检查及独立生成消费均通过。浏览器使用受控 API，不能替代真实供应商或安装态验收。

真实本机检查已执行：`http://127.0.0.1:11434/api/tags` 返回 200、4 个型号；但 `qwen3.5:cloud` 的 `/api/show` 返回 HTTP 410。SDK 明确返回 `AI_MODEL_DISCOVERY_UPSTREAM_FAILED`，保留失败型号、实际地址和上游状态，因此**真实完整目录刷新尚未通过**。没有跳过错误型号、下载／删除模型、修改活动配置或触发生成。该现场依赖需在 Ollama 侧核实后重新验收。

| 范围 | 结果 |
|---|---|
| Go 目录 SDK、Go Skeleton、本地凭据／快照、Nuxt、CLI 合同与模板 | 源码与受控消费完成 |
| 本地 Ollama 新原生型号显式试跑 | 受控完整链路通过；真实目录刷新受上述 410 阻挡 |
| 本地云供应商目录 | 协议与凭据隔离受控验证完成，真实账号待验收 |
| 本地云供应商推理执行器 | 未新增；保持原来的明确不支持状态 |
| .NET／FastAPI 在线目录适配 | 未实现，不计入本轮交付 |
| 用户库迁移、常驻服务加载、真实云账号／安装态 | 待部署方执行与联调 |
| 提交 SHA／公共版本发布 | 本轮未执行 |
