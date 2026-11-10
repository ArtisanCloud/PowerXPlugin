# Framework 本地在线模型目录 v1

「插件本地配置」由插件后端直接查询供应商目录；「PowerX 底座配置」继续通过 typed Core 目录能力读取候选。两者不切换业务运行通道。在线获取不是连接测试，也不是模型试跑。

## 本地使用流程

1. 选择环境、模态和供应商，填写服务地址。需要密钥时，填写服务端预先配置的凭据引用，不填写原始 Key。
2. 点击模型旁的「更新在线模型」。弹窗显示获取地址、协议、更新时间，并分为「已匹配」「待确认」。只有这个显式操作会请求供应商并保存目录快照。
3. 对用途未知的型号，可在文本或向量分区显式加入候选；媒体生成能力不能按型号名称推断。每批最多确认 1000 个，确认请求必须带返回的 revision。
4. 关闭弹窗后明确选择型号，再保存配置。原生 ID 和完整 tag 原样保存；更新目录不自动选择型号、修改参数、创建 profile、生成内容或下载模型。

编辑地址或引用后离开输入框，只回读插件数据库中的当前连接候选，不请求供应商。换租户、环境、供应商、App、地址或密钥后，旧动态候选不能进入新范围。旧请求不能回写新页面状态。

当前本地执行器支持 Ollama；云供应商目录可获取和保存，实际云模型试跑仍须等待对应本地执行器实现。列表可见不表示模型已获生成权限、存在足够余额或已验证实际生成能力。

## 供应商与结果

目录协议对齐 Core：OpenAI、OpenRouter、DeepSeek、Moonshot、Hugging Face、OpenAI Compatible、Qwen 国内／国际、Anthropic、Gemini 和 Ollama。Ollama 使用 GET `/api/tags` 及逐型号 POST `/api/show` 获取明确能力；这些调用不加载或下载模型。未实现的供应商明确返回 `AI_MODEL_SYNC_UNSUPPORTED`。

完整分页结束后才写入快照：最多 100 页、10000 个型号、每页 8 MiB；默认总期限 20 秒，可通过 `local_ai.catalog_timeout_seconds` 设置，最大 120 秒。拒绝重定向、畸形响应和不完整分页，不自动重试，不用静态样例冒充在线更新成功。

快照按租户、环境、Provider、App 隔离并绑定连接指纹。刷新保留同一连接中仍在线型号的人工分类；地址或密钥改变后旧动态候选失效。失败、过期 revision 或并发竞争不会覆盖旧快照。Ollama 刷新后的候选反映实际实例，已移除型号不能继续作为可保存的新候选；原活动配置仍保留，需用户明确更换。

`uuid` 是稳定快照引用；`revision` 是不透明 UUID，只用于比较并交换，不是条目数量或可递增版本。接口不返回凭据、连接指纹、数据库 numeric id 或供应商原始错误正文。上游机器分类、实际失败地址／型号、HTTP 状态和可用 trace／request_id 保留。上游 401／403 对页面返回 HTTP 422 `AI_MODEL_SYNC_AUTH_FAILED`，避免把供应商鉴权失败当成用户登录过期。

## 服务端凭据

使用主配置的 `local_ai.credentials` 声明显式绑定，密钥通过服务器环境变量解析：

```yaml
local_ai:
  catalog_timeout_seconds: 20
  credentials:
    - reference: "22222222-2222-4222-8222-222222222222"
      tenant_uuid: "11111111-1111-4111-8111-111111111111"
      environment: development
      provider: openai
      endpoint: https://api.openai.com/v1
      api_key_env: PLUGIN_AI_OPENAI_KEY
      # organization_env: PLUGIN_AI_OPENAI_ORGANIZATION
      # anthropic_version: "2023-06-01"
```

引用是稳定 UUID，绑定租户、环境、供应商和地址。浏览器不能选择环境变量名、替换租户身份或把密钥发给另一地址。没有匹配绑定、密钥环境变量为空或范围不匹配时明确失败，不使用 Core／网关／其他租户凭据。本地 Ollama 和 OpenAI Compatible 的 Key 可以为空；显式提供引用时仍必须成功解析，不会忽略无效引用。

服务权限由插件认证上下文中的 Root 或当前租户 `role_owner`／`role_admin` 决定，前端禁用不是授权依据。HTTP body 不接受 `tenant_uuid`、`source`、`api_key`、`headers`、`method`、prompt 或自由代理字段。

## 接口与迁移

- POST `admin/ai-settings/catalog/local-online`：`action=refresh|assign`，合同见 [OpenAPI](ai-local-model-catalog.v1.openapi.yaml)，共同样例见 [fixture](fixtures/ai-local-model-catalog.v1.json)。
- GET `admin/ai-settings/catalog/models?source=local`：携带 `environment`、`modality`、`provider`、`app`、`endpoint`、`credential_ref`，只读取当前连接的静态／有效动态候选。
- 公共 Go 目录发现模块为 `runtime/ai/modeldiscovery`；Nuxt Client 导出 `createLocalModelCatalogClient` 和正式输入／结果类型。

部署前，在插件 Go 后端目录使用对应配置执行单项迁移：

```bash
CONFIG_PATH=/path/to/config.yaml go run ./cmd/database migrate-ai-online-catalog
```

该命令创建 `local_ai_online_catalogs`，配置 PostgreSQL 强制 RLS，并将既有 `local_ai_settings.model_key` 扩展为 512；不重写活动配置、不 seed、不重置数据库。SQLite 支持同一目录功能，使用显式租户过滤。未迁移时返回 `AI_LOCAL_CATALOG_MIGRATION_REQUIRED`；页面说明所需动作，连接探测仍独立可用。

迁移通过函数 `RollbackAIOnlineCatalog` 提供回滚：只删除新目录表，不删除活动配置，也不缩短已经扩展的型号字段。回滚会丢失目录快照，应由部署方明确安排。

## PowerX 来源

「刷新模型列表」只调用现有 `list_settings_models`／媒体 typed 目录读取。在线同步和候选分类保留在 Core 管理页，插件 API Key／STS 不调用 Admin 同步路由。管理页跳转地址通过前端 `NUXT_PUBLIC_POWERX_CONSOLE_BASE` 明确配置；未配置时显示原因，不猜测管理站端口，也不从 API 地址推导。
