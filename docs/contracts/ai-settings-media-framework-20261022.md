# Framework AI 设置页与旧注册对齐交付（2026-10-22；错误合同补齐 2026-10-23）

本轮修复设置页 TestConnection／QuickCall 对所有模态调用 LLM 的问题。目录与生成 SDK 继续提交 typed body，endpoint、method 由 Core 注册决定。运行通道在启动时绑定，模型目录来源不改变执行通道。

## 已实现

| 范围 | 结果 |
|---|---|
| Go 设置接口 | 按 llm／vlm／image／video／audio_tts 分别调用对应 Service；VLM 返回分析，生成返回原 Core 任务及 HTTP 202 |
| 本地模式 | 文本测试可使用未保存的 Ollama Profile，透传测试输入与参数；VLM 使用已配置的启动模型；未实现的生成能力明确失败，无 Core fallback |
| Nuxt 设置页 | 模型配置、参数、能力测试分区；VLM 图片输入；能力／模型／通道不满足时禁用操作并说明原因 |
| 任务与重试 | 每次显式新尝试创建幂等键；失败重试保留原请求和键；任务查询／取消使用原 UUID；较旧 revision 不覆盖新状态，无自动轮询 |
| 草稿与状态 | 模态／来源／环境切换保留当前页面会话内的草稿、测试输入、结果和生成尝试；刷新页面不自动重新提交生成 |
| 契约与模板 | 同步 Core 当前合同快照、共享样例及独立的 Framework 旧注册场景 fixture；Skeleton、两套 CLI 模板和生成器内嵌合同同步 |

两个测试按钮都会执行选定能力。生成类测试可能产生供应商费用，页面在提交前说明；异步任务被接收不标记为生成完成。开始新测试是显式操作，可能再次计费。服务凭据保留在后端。

## 消费接口边界

`POST admin/ai-settings/test-connection` 和 `POST admin/ai-settings/quick-call` 现在要求在现有配置字段之外提供 `inputs`；生成能力还要求稳定的 `idempotency_key`。设置页面与对应后端需要一起升级。VLM 模态值为 `vlm`，TTS 页面模态值保持 `audio_tts`。未知或未实现的能力不会改调 LLM。

设置页返回值分别为 `data.text`、`data.type=analysis` 或带 `task_uuid` 的 Core task。目录 SDK、GenerativeService 和公共 Nuxt AI media 接口本轮没有新增方法或改变形状。

## 可消费本地版本（2026-10-23 合同补齐包）

| 产物 | 精确引用 |
|---|---|
| Go SDK | `v0.0.27-dev.aisettings.ea782ad8b674322b` |
| Nuxt Client | `0.0.14-dev.aisettings.1f47c36788746a87` |
| CLI | `v0.0.0-aisettings.cc2f82e8ac7e0000` |
| 引用、摘要及消费证据 | [references.json](../../artifacts/framework-release-ai-settings-20261022-ae21044e9e205ed3/references.json) |
| 生成插件 | [generated-consumer](../../artifacts/framework-release-ai-settings-20261022-ae21044e9e205ed3/generated-consumer) |

Go 本地代理为 `file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-ae21044e9e205ed3/go-proxy`；模块路径为 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`。Nuxt 包引用为 `file:/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-ae21044e9e205ed3/artisan-cloud-plugin-framework-client-0.0.14-dev.aisettings.1f47c36788746a87.tgz`。CLI 二进制为 `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-ai-settings-20261022-ae21044e9e205ed3/px-plugin`。

这是可从当前工作区读取的内容寻址开发包，未公开发布。改动仍在当前分支工作区，未创建实现提交；基线提交为 `f123a2310568bd0e3625e825b4fa419b9cfb4f6c`，不能将它当作包含本轮改动的提交。产物通过 `GOWORK=off`、无 Framework replace 的实际 Go 模块消费；生成清单校验、后端编译、内嵌契约及打包 SDK TypeScript 检查通过。

MediaXStudio 现有固定快照中的 AI SDK 实现文件与当前 Framework 一致。本轮无需仅为 Core 注册修复更换该快照。如果需要采用新设置页，应显式同步对应页面、后端和 locale；只升级 Go/Nuxt 公共 SDK 包不会自动更新插件自有页面。

## 证据与 Core 配合项

[验收记录](evidence/ai-settings-media-framework-validation-20261022.json)保留了 2026-10-22 页面验收：10 个 Go 测试包、132 个测试事件（含子用例）、5 个包 race、vet、19 个前端用例、4 个浏览器场景、Nuxt 构建及实际生成插件消费通过。浏览器使用受控 API fixture，检查桌面／窄屏分区、状态保留、幂等重试、VLM 必需图片、禁用条件、查询取消和 revision 去重。临时预览端口 3147 已释放。

Core 仍负责旧租户注册迁移、版本确认、真实 API Key／STS grants、缓存刷新和部署进程。Framework 不执行 seed、迁移或升级确认，不替换用户服务。真实安装态需在升级后的 Core 构建上分别验证 API Key 与 STS。

### 2026-10-23 升级错误合同补齐

Core 已发布 `capability_version_upgrade_error.v1` 的 10 类权威错误。已同步四份来源摘要，保留 `source_worktree_uncommitted=true`；文档、SDK testdata、CLI 内嵌及新生成项目的样例一致。升级确认错误预期为 `capability.upgrade_binding_mismatch`，`machine_reason_published=true`，HTTP 409 与 `message=capability.upgrade_failed` 保持不变。

真实 `DecodeHTTPError` 已覆盖全部 10 类样例，保留 HTTP 状态、ErrorCode、ReasonCode、trace、request_id、details 及原始 body。`runtime_apply_failed` 保留 `approval_persisted=true` 和 audit_id，运行锁状态仍未知；`audit_finalize_failed` 同时保留 `runtime_confirmed=true`，不能将其解释为完全未执行。`retryable=true` 仅支持修复依赖后人工重新确认，不授权自动审批重试。

本轮 5 个 Go 包、92 个测试事件（含子用例）、3 个包 race、5 个 vet 目标、6 个一致性检查用例（其中 5 个为缺失／漂移／旧预期的失败检查）及模板检查通过。API Key／STS 共 20 组受控错误响应各只发出一次原媒体请求，没有审批或替代通道请求。开发包生成后使用 `GOWORK=off`、无 Framework replace 从模块代理运行 SDK 升级错误回归；生成项目内权威 fixture 的 SHA256 为 `ff70b10ec2099bf96fe026fe1b91d55306e8ef1a5e24942aafc4bb9504a57faf`，与 Core 一致。

本轮未修改 AI SDK 运行时、增加 SDK 方法或改变媒体协议，也未改 MediaXStudio 快照。05／06／08 配置验收可继续。本地开发验证包更新不要求 MediaXStudio 升级消费快照；公开发版和真实安装态验收仍分别记录。

`fixtures/ai-media-v2.json` 和 `fixtures/capability-version-upgrade-errors.v1.json` 来自 Core；`fixtures/ai-media-v2-registration.json` 是 Framework 自有的受控客户端测试场景，不能替代 Core 的数据库迁移或真实租户验收。Go、Nuxt、Skeleton／CLI 本轮交付完成；.NET AI media v2 仍后置，不计入全语言交付。
