# AI 在线模型目录

在「设置 → 模型配置」选择 Provider，保存连接地址与 Key，再点击 Model 旁的「更新在线模型」。Core 使用当前租户、环境已保存的连接查询清单。有未保存的连接修改时，页面会提示先保存，避免误用旧地址或旧 Key。

`default` 是 `dev` 的历史别名。页面、HTTP／gRPC 配置接口、在线目录与运行时使用同一套读取规则：先读取同租户的 `dev` 对应记录，缺失时读取旧 `default` 记录。规范环境已有记录时整条优先，不拼接旧 Key、地址或参数；也不跨到生产环境或其他租户。旧密文仍按记录实际存储的环境解封。

读取和在线更新不迁移凭据。用户正常保存配置时写入 `dev`，继承的旧密钥先按原环境解封，再按目标环境重新密封，旧记录保留。模型画像、默认路由和旧在线目录快照同样兼容读取；API 返回规范环境名称。显式凭据轮换会分别处理两份物理环境的密文，不混用 RSA 作用域。

管理表单中的 Base URL 留空表示使用当前 Provider／接入方式的默认地址，清空后会在字段下展示该地址。连接测试、试跑与保存都遵守这一规则，不再从旧凭据补回自定义地址。保存空地址时保留“使用默认地址”的含义，后续运行与在线目录按同一规则解析；没有默认地址的服务必须填写地址。服务内部仅更新凭据时，省略 `base_url` 字段仍表示保留旧地址，与显式提交空值不同。单纯查看或编辑表单不会切换运行中的连接，成功保存后才使用新配置。只读连接探测日志记录经校验的实际目标 URL，不记录 Key。

成功结果保存到本地数据库的租户配置表；弹窗显示实际获取地址、协议、最近获取时间及匹配数量。无需修改源码 YAML 或安装目录。不创建模型画像、不更换活动模型、不改参数、不下载模型、不发送 prompt。

## 各 Provider 的获取方式

| Provider | 查询方式 | 鉴权／分页 |
| --- | --- | --- |
| OpenAI | Base URL 下的 `GET models`，默认 `/v1/models` | Bearer Key，保留 Organization |
| OpenRouter | Base URL 下的 `GET models?output_modalities=all` | Bearer Key，读取明确输出模态 |
| DeepSeek | Base URL 下的 `GET models`，默认 `/models` | Bearer Key，读取明确输出模态 |
| Moonshot | Base URL 下的 `GET models`，默认 `/v1/models` | Bearer Key，不根据型号名称猜用途 |
| Hugging Face Router | Base URL 下的 `GET models`，默认 `/v1/models` | Bearer Key，读取明确输出模态 |
| OpenAI Compatible | 保存的兼容服务器的 `GET models` | Key 可选，已保存时会传递 |
| Qwen 国内／国际 | 同一 origin 的原生 `GET /api/v1/models` | Bearer Key；处理 `output.total/output.models` 分页 |
| Anthropic | 原生 `GET /v1/models` | `x-api-key`、`anthropic-version`；处理 `has_more/last_id/after_id` |
| Google Gemini | 版本路径下的 `GET models`，默认 `/v1beta/models` | `x-goog-api-key`；处理 `nextPageToken`；读取 `supportedGenerationMethods` |
| Ollama | 实例 `GET /api/tags`，逐型号 `POST /api/show` | 原样保留 tag，读取 capabilities，不加载模型 |

固定协议路径会保留自定义网关前缀。Qwen 将兼容路径后缀转换为原生目录路径；原生目录可能要求所在区域的 workspace 专属域名及相应目录权限。旧共享域名或第三方兼容网关以实际响应为准。

此轮未接入 Agnes、Volcengine、Hunyuan、Baidu、Coze、ComfyUI、Hash、Sentence Transformers 及 Azure deployment 模式的在线目录。接口返回 `AI_MODEL_SYNC_UNSUPPORTED`，保留本地目录。上游接口失败时显示错误码、实际查询地址、上游 HTTP 状态和可用 trace 信息。

## 匹配和使用

- 「已匹配」包括精确匹配本地目录的型号、供应商明确声明且本地有驱动的文本／向量候选，以及管理员确认过的候选。静态型号的 defaults、limits、operations 仍由现有目录负责。
- 只有 ID、无法确认用途的新型号进入「待确认」。文本／向量分区可勾选这些型号，明确「加入候选」后进入对应下拉框，不自动选中。媒体型号仍需要正式 operation 合同，不能凭型号 ID 自动发布能力。
- 在线刷新原子替换当前连接快照，保留同一连接、仍在线型号的人工分类；失败不覆盖旧值，不以静态样例冒充同步成功。本地静态目录不删除。
- 关闭弹窗后选择型号，再按原流程保存模型配置。实际生成可用性通过单独的「试跑」验证，目录同步不触发连接测试或试跑。

快照按租户、环境、Provider、App 保存，并绑定连接身份指纹。换 Key 或地址后，旧候选不参与读取；其他租户／环境不能继承。Key 不写 URL、日志、目录快照，指纹不进入 API 响应。

云 Provider 的管理页与既有 `list_settings_models` typed 读取共用快照。Ollama 原有实例实时发现语义保留：普通下拉读取仍反映实际实例，在线同步另存带匹配明细的快照。

## 管理 API

逻辑路由 `POST /admin/agents/models/sync`，API 前缀沿用 Core 配置。只允许用户 JWT 的 Root 或当前租户 owner/admin，拒绝 API Key、STS 和普通成员。它是管理配置入口，已列入能力审计 ignore；不新增插件服务授权或扩大共享 Profile。

```json
{"env":"dev","provider":"openai","app":"","modality":"llm","action":"refresh"}
```

确认候选使用同一路由，提交响应 revision 和原样型号 ID：

```json
{"env":"dev","provider":"openai","app":"","modality":"llm","action":"assign","revision":"返回的 revision","models":["已发现的新型号 ID"]}
```

每批最多确认 1000 个候选。写入和回读在同一事务完成。确认候选校验 revision，防止覆盖新目录。Body 不接受任意 endpoint、headers、method、Key 或身份字段。日志记录 action、tenant、env、provider、app、actor、数量及 revision。

目录查询使用 `capability_registry.default_http_timeout_seconds`，缺省 20 秒；调用方更早的 deadline 仍优先。它与 LLM 推理期限分开。最多 100 页、10000 个型号、每页 8 MiB；拒绝重定向、畸形及不完整分页，完整获取后才保存。

## 验收与恢复

更新失败时看弹窗错误码和上游地址；检查已保存连接的 Key、区域、目录权限及接口支持，然后重试。不会自动替换活动模型；如不需要新候选，保持原型号即可。改变连接身份后旧快照自动失效。

目录请求的 401／403 对前端返回 `AI_MODEL_SYNC_AUTH_FAILED`（Core HTTP 422，避免触发用户 JWT 刷新），详情保留 `source`、`provider`、`protocol` 与 `upstream_status`。供应商明确通知旧版免费 Key 已停用、需绑定 GitHub 时，仅返回固定分类 `upstream_reason=legacy_free_key_retired`，页面提示获取并保存新版 Key；不转发供应商原始错误消息、Key 或账号信息。失败日志带同一请求上下文及上述诊断字段。连接测试结果属于当前表单，切换环境／模态／Provider、修改配置或离开页面后失效，旧请求不得回写成功或失败提示。

受控验收覆盖协议路径、鉴权、分页、原样 ID、匹配／分类、持久化回读、运行配置接受候选、跨租户／环境／模态隔离、Key 变更失效、上游和写入失败保留旧值，以及桌面／窄屏弹窗与无自动推理请求。真实账号的目录权限、网络可达性和生成支持，需在重启后的 Core 用对应连接分别验收。

官方协议参考：[OpenAI](https://developers.openai.com/api/reference/resources/models/methods/list)、[OpenRouter](https://openrouter.ai/docs/api/api-reference/models/list-all-models-and-their-properties)、[DeepSeek](https://api-docs.deepseek.com/api/list-models/)、[Hugging Face](https://huggingface.co/docs/inference-providers/hub-api)、[DashScope](https://www.alibabacloud.com/help/en/model-studio/list-models)、[Anthropic](https://platform.claude.com/docs/en/api/models/list)、[Gemini](https://ai.google.dev/api/models)、[Ollama](https://docs.ollama.com/api/tags)。
