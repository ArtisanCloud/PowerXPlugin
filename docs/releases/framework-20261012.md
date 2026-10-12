# Go Framework v0.0.25 发布准备说明

准备日期：2026-10-12。当前状态：本地代码与测试准备，尚未发布 v0.0.25。

| 交付 | 版本 | 本次状态 |
|---|---|---|
| Go Framework | v0.0.25 | 待 CI、合并及发布 tag |
| Nuxt framework-client | 0.0.12 | 沿用已发布版本 |
| Nuxt framework-admin | 0.0.11 | 沿用已发布版本 |

Go 模块为 `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`。本说明按已发布 v0.0.24 到当前代码的差异整理；覆盖主联系人、Unicode、AI 模型元数据和知识内容导览。Skeleton 和模板变更属于消费方接入示例，升级 SDK 不会自动替换插件内的这些代码。本次不发布 npm 或 .NET 包。

## 主联系人管理

新增独立 `contactfw.PrimaryContactManager`、`SetPrimaryContactInput`、`PrimaryContactResult`，从启动时选定的 `Runtime.PrimaryContactManager()` 获取实现。原六项 `Store` 不扩展；Local 适配器由插件注入，Delegated 使用固定 `core_internal` 合同。

设主调用固定能力 `com.corex.customer.contacts.primary_manage`、`INVOKE core://customer/contacts` 和 operation `set_primary`。`expected_primary_contact_uuid` 必须存在：nil 拒绝，明确的空字符串表示当前无主引用。响应核对客户、目标联系人和前一个引用，拒绝错误归属、协议变更或 fallback。

Framework 识别 `CONTACT_PRIMARY_EXPECTED_REQUIRED`、`CONTACT_PRIMARY_CHANGED`、`CONTACT_PRIMARY_NOT_ACTIVE`、`CONTACT_PRIMARY_OPERATION_REQUIRED`、`CONTACT_PRIMARY_CHANGE_REQUIRED`、`CONTACT_PRIMARY_OPERATION_FAILED` 和 `CONTACT_REQUEST_INVALID`。业务码、实际 HTTP 状态、Trace、request ID 和原始 cause 保留给消费方。

普通联系人创建不能携带 `primary`，普通编辑不能通过角色改写主引用。停用当前主联系人需先更换主引用；客户创建仍按现有原子合同建立主联系人，不增加额外设主调用。订单导入时选择联系人继续使用既有查询/管理接口。

Skeleton Local 示例在同一事务中检查客户/联系人归属、active 状态和预期旧引用，维护唯一 primary 角色、membership 主引用及审计。审计失败回滚，普通编辑共享同一锁。客户回读从 membership 获取主引用；缺少 membership 明确失败，不读取旧缓存兜底。

调用示例、授权字段和 Core 配合事项见 [主联系人对齐合同](../contracts/contact-primary-framework-alignment.md)。

## Unicode 校验

Customer/Contact 姓名上限为 128 个 Unicode codepoint，邮箱 255、电话 32；拒绝控制字符和非法 UTF-8。Customer 更新在 JSON 编码前检查原始字符串，防止 Go 将非法 UTF-8 替换后绕过校验。ASCII tag 规则及 Customer external identity subject 的协议 byte 上限继续按各自合同执行。

## AI 模型元数据

`powerx/ai.LLMModel` 新增 `Modality string` 和 `Capabilities []string`，保留 Core 模型列表中的结构化声明。消费方可据 `modality=llm`、能力包含 `text_generation` 识别文本生成模型，无需从模型名称猜测用途。模型能力声明不代替 Core 调用授权。

客户端测试使用 Core 的 `tenant_profile` 来源枚举，并验证两个新增字段及来源透传。Skeleton Local 模型目录也返回 `llm` / `["text_generation"]`，保留其 `source=local`；两份 Go 模板同步此实现。

## 知识内容导览

新增内容导览 Query/Result DTO、可选 `ContentOverviewService` 和 Delegated/Host 适配，固定调用 `com.corex.knowledge.content_overview.service_read` / `core://knowledge/content-overview` / `list_articles`。响应校验文章来源、已发布原文预览、语料 generation 与分页，并保留请求/调用 Trace。

本项属于 SDK 合同接入。Local 实现须由消费方提供；未实现时明确返回 unsupported。真实 Core 凭据、正式数据库和浏览器内容导览验收仍未完成，本版本不声明这些验收已通过。

## 本地验证与待验收项

2026-10-12 提交前验证：

- Go Framework 全模块测试。
- Skeleton `runtimeexample`、`services/customer`、`admin/customer` 包测试，覆盖 Local 模型声明、设主事务、业务冲突和 HTTP 错误信息。
- 模板同步检查和 Git diff 格式检查。

主联系人/Unicode 合同及 Local 事务相关 race 检查已在 2026-10-11 通过；本次 AI 元数据增量由上述 2026-10-12 测试覆盖。本地测试使用测试替身和 SQLite 事务夹具，不代表真实 API Key/STS 授权、PostgreSQL 联测或浏览器验收完成。

2026-10-12 只读核对 Core 源码，`selectInvokeErrorTemplate` 仍未把 `CONTACT_*` 加入业务 HTTP 状态透传分支，调用入口可能把业务冲突包装为 HTTP 502。Framework 保留实际状态及业务码；Core 团队需补状态透传，并在目标实例验证 400/409/500、Trace 和精确授权。

当前分支 CI 的生成项目检查使用显式 Framework 源码快照，不能作为 v0.0.25 已发布或可下载的证明。包含本次提交的 GitHub CI、真实 Go proxy 下载、无本地替换的生成项目构建均待后续完成。

## 发布与消费方升级

1. 在现有 `009-consume-powerx-capability` 分支提交代码和本说明，推送后验证对应提交的 CI，由用户创建/合并到 develop 的 PR。
2. 在验收通过的合并提交创建并推送 Go 子模块 tag `framework/backend/go/v0.0.25`。Go 模块发布由该 tag 提供版本；仓库现有 `v*` Release workflow 不会由此 SDK tag 自动触发。
3. 验证 Go proxy 可下载 v0.0.25，关闭 workspace、移除本地 Framework replace，运行生成后端检查。CLI 和 Skeleton 已预置 v0.0.25，发布后补齐正式依赖校验记录。
4. 消费方按自己的发布流程更新依赖、构建并重启插件。AI Craft 接入与真实业务验收由其团队执行，Core 精确授权由 Core 团队完成。

发布成功后，在消费方后端目录执行：

```bash
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.25
go mod tidy
go test ./...
```

正式验收需确保 `GOWORK=off` 且该模块没有本地 `replace`。保留本地替换时，编译使用本地源码；已运行的进程需重新构建和重启才能加载修改。遇到 `CONTACT_PRIMARY_CHANGED` 应回读客户并让操作者再次确认，不自动覆盖旧引用。

截至本说明编写，只准备发布内容和本地提交；不宣称已推送本次提交、已通过对应远端 CI、已发布 tag 或已重启服务。
