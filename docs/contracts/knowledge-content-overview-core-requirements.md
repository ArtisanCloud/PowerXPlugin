# 知识检索页内容导览：Core 交付要求

日期：2026-10-09。目标页面：`http://127.0.0.1:3131/powerx/knowledge-lab/:space_uuid/playground`。

用户需要在提问前知道空间文章的主题。本地知识库与 PowerX 继续共用 `KnowledgeSearchWorkspace` 和 `KnowledgeContentOverview`；两条数据链路保持各自的权威源。

## 当前事实与插件交付

- 当前 Core tenant Knowledge Host 路由已有空间、检索、文档提交/删除、模型索引和单任务快照读取；尚未发布空间文档目录/内容导览的读取合同。此结论来自 2026-10-09 只读核对 `backend/internal/transport/http/openapi/knowledge_space/routes.go`，没有修改 Core。
- Plugin 的 `usePowerXKnowledgeWorkspace` 未启用内容导览 accessor，也不会请求未经交付的 Core 地址。PowerX 页面显示简洁的未接入说明，现有检索独立可用。
- Local accessor 已读取本地授权的空间文档 API，返回真实标题和原文前 320 个 Unicode codepoint。它是目录及原文预览，不宣称这些文章全部已可检索，不把原文截取称为 AI 摘要。
- 共用组件支持概要、主题、文章目录、原文预览、来源文章绑定的推荐问题、显式分页和失败重试。点击问题只填入输入框，不自动检索。固定的 Framework 示例问题已移除；未知空间不会出现其他空间的文章。
- `app/composables/knowledge/contentOverview.ts` 是前端内部 view model，**不是已发布的 Core wire contract**。Core 的正式路径、schema、字段和权限仍需 Core 确认；Framework 之后做 typed 映射，不以 view model 假装 Core 已交付。

## Core：先交付最小可用的只读目录

1. 提供受租户和对象授权约束的 typed Host 读取合同；接受空间 UUID、检索模式 semantic/hybrid、要求的 corpus generation，以及分页参数。最终 HTTP 路径及 core_internal binding 由 Core 写入正式合同/OpenAPI。不可使用用户 Admin JWT 接口替代 API Key/STS 服务身份合同。
2. 返回当前模式、指定 corpus generation 下**实际可检索且当前调用者有权读取**的文章。目录、total 和内容预览必须应用与 retrieval 一致的租户、文档 visibility 和其他对象授权过滤。没有索引或无查询资格的另外六份验收样本不能被当前五篇语义文章的目录、统计或概要混入。
3. 每篇至少提供 Core 文档 UUID、可读标题、原文预览及对应原文版本/来源定位。UUID 作为引用；预览须来自该文档实际已发布语料，可与 source checksum/version/codepoint 区间核对。不要求在每次读取时调用 LLM。
4. 返回过滤后的 total、明确的 next cursor/终页语义、实际 corpus generation。分页使用稳定排序和快照约束，约定语料或权限变更时的 HTTP/机器错误码。服务端在读取时复核授权；generation 不匹配不得用最新语料或旧缓存静默替代。
5. 无文章返回合法空目录。401、403、404、版本冲突和上游依赖失败须区分，保留 Trace/request ID；请求失败不能返回空成功。
6. 明确 API Key 和 STS 的精确 capability/action/selector。若复用既有读取权限，请写明映射并验收；不要求无依据的重复授权或扩大到 document.manage。

上述最小合同交付后，Plugin 就能展示真实文章目录、原文预览和“围绕这篇提问”，不依赖 AI 概要能力。

## 可选的概要、主题与推荐问题

这些字段只在 Core 有真实、可验证的产物时返回，缺失时不能填入模板答案或虚构主题。

- 空间概要和主题必须仅涵盖本次调用可检索、可阅读的文章；不能用跨租户或空间全量概要泄露隐藏文档。
- 推荐问题须绑定来源 Core 文档 UUID、对应版本/generation，描述文章确实覆盖的内容。移除、隐藏或失去查询资格的文章不能继续贡献问题或概要。
- 标明产物来源、生成/发布状态、适用语料版本和语言。缺失或失效需明确表示；不在 Plugin 里从自由文本猜测 Core 结构化元数据，不用检索采样结果冒充完整目录。

## Framework 和 Plugin 后续责任

- **Framework Go / .NET**：在 Core 正式交付后新增 typed 目录 service/accessor、请求/响应 DTO、Core client 和 delegated adapter，保留精确 HTTP/错误码/Trace。Local 的目录读取由消费插件实现，不从插件访问 Core DSN，不添加 raw proxy 或降级到 Local 的路径。
- **Plugin 后端**：增加受保护的 Lab 内容导览端点，通过 Framework 的 typed accessor 读取。浏览器仍只持有 Plugin 登录身份；PowerX API Key/STS 留在服务端，不接受浏览器指定 Core URL、租户覆盖或 capability。
- **Plugin 前端**：将正式 Host DTO 映射到现有 view model，启用 PowerX `contentOverview` accessor。共用组件按检索 mode/generation 刷新，清除旧空间/旧版本数据；单独的目录失败不阻断已就绪的检索。

## Core 交付与验收清单

请提供正式合同、OpenAPI、capability/binding 映射、最小真实响应及脱敏证据。真实 API Key 与 STS 都应覆盖：

- dev 当前五篇可检索文章的标题、预览、分页总量与检索来源一致。
- 跨租户、无文档权限、hidden/non-queryable、旧 generation、分页期间语料/visibility 变更。
- 空目录、无权和上游失败的区别；Trace 可关联日志。
- 若交付概要/推荐问题，逐条可溯源到当前授权文章；缺失产物明确缺失。

交付后由 Framework/Plugin 接入并完成真实浏览器复验。当前内容导览的 PowerX 数据接入不能标记为完成；Core 不需要为本次前端修改重启或重建索引。
