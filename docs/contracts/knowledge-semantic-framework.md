# Framework Go 语义索引合同对齐

日期：2026-10-08。本轮范围为 Go Framework 和 Skeleton Lab，开发通过本地 `replace` 使用新源码，尚未发布新 Go/npm 版本；.NET 的开发改动不计入本轮语义完成状态。

提供方定义：[Core 合同](../../../../PowerX/docs/contracts/knowledge-semantic-host.md)、[Core 对齐清单](../../../../PowerX/docs/contracts/knowledge-semantic-framework-alignment.md)。

## 使用边界

- `SemanticKnowledgeProvider` 为强类型扩展：能力查询、配置冻结、严格语义查询和 visibility。当前 Local memory adapter 明确返回不支持，不回退 lexical；原本 Local 业务查询保持自己的合同。
- API Key 的 Host simulation 使用正式 tenant REST。Delegated 使用 `/api/v1/tenant/invocations`，`preferred_protocol=core_internal`，固定 capability、固定 `INVOKE` 和 `core://knowledge/retrieval` / `core://knowledge/documents`，调用方不能自选转发参数。
- SDK 保留完整 query text、四类 filters、required_generations、逐源分数/方向、多分块、不可变来源与三个独立 generation。没有候选结果缓存或业务 fit score。
- Go 来源定位用 Unicode codepoint，`SourceText` 基于完整不可变源和 checksum 读取 `[]rune[start:end]`，不使用 byte 或 UTF-16 下标。

## 文档与重建

`KnowledgeDocument` 新增 `Indexing`、`Artifacts`、`ExternalRef`、`IdempotencyKey`。`HostIngestion` 继续承载正式 snake_case 入库配置；Source Version 保留 opaque 字符串，不转换成整数。

`DocumentRebuildSettings.Indexing` 支持语义重建：reuse_snapshot 禁止新 ingestion/indexing；apply_config 至少提供一个明确配置，对语义文档必须提供 indexing，由 Core 校验。DocumentID 非空只发送 rebuild_document/文档 REST，不扩大范围。同次网络重发保留键，修改请求或失败后开始新执行使用新键。

任务回读保留 artifact_count、vector_count、bundle_generation、effective_config.indexing 和逐文档 document_configs，不只读取第一项。Core queued/running 不能映射成已完成。

## 调试页面

- 入库页读取真实 semantic capabilities，区分 configured/ready/indexed，显示 active/pending 模型。用户明确选择 lexical_chunks、semantic 或 hybrid。
- 可以提供手写联调画像文本，随原文生成强类型 test artifact/source refs；不负责业务画像抽取。SHA256 使用完整 UTF-8 文本，不加前缀，位置为 Unicode codepoint。
- 检索页提供 semantic/hybrid、结构化前置过滤、可选 corpus generation 要求；分别展示 distance、similarity、text rank、RRF、来源区间和版本技术详情。
- 重建弹窗区分两种配置模式、当前/待发布模型以及内容角色；不会隐式推进整个空间。任务详情显示真实配置与产物数量。
- visibility false 成功后清空当前候选，不用缓存保留被撤销资格。

## 发现的合同差异

Core 对齐清单把任务 operation 写为 get_index_job/get_index_job_chunks，但实际 `provisioning_capability_invoker.go` 接收 get_job/get_job_chunks。SDK 使用实际操作名，无别名重试。

只发送 payload.body 的文档例子在当前运行链路曾返回 unsupported capability 502；源码允许且校验固定 method/endpoint。SDK 现在显式发送编译时确定的 INVOKE/core endpoint，仍没有 raw proxy 或 REST fallback。补齐后真实 STS 写入回归通过。

## 验收证据

[真实 SDK 记录](evidence/knowledge-framework-semantic-20261008.json)：73 条记录，其中 63 条为 Framework SDK 发出的真实 HTTP 请求，8 条为 Core 原始非法 DTO/Registry/Admin 隔离负例，1 条为真实 STS Exchange，1 条为数据库 pgvector 读回。初始 640/80 bundle 的实际向量数为 3、维度 1024。

覆盖原文/画像、实际分块与多块来源、完整自然语言和过滤、单篇/空间重建、失败旧索引保护、模型 pending/原子发布、跨租户、只读 Key、即时撤销、版本冲突和来源损坏。临时夹具与凭据已清理，证据不含认证凭据。模型故障明确失败；测试没有把向量结果转成业务适配分。

重跑方式见 [独立验收模块](../../tests/acceptance/knowledge-semantic-sdk/README.md)。Admin 用户 JWT 正向验收属于 Core，本轮仅验证真实 STS 不能进入 Admin 平面。

SDK 真调用不替代插件运行进程或浏览器验收。插件 8078 需加载新 Handler；浏览器连接工具若被现有 Chrome profile 占用或 CUA fetch 失败，应明确记录视觉验收缺口，不能以 build 代替。源码、合同测试和真实 SDK 验收分别记录。

## 插件本地启动

在 `skeleton/backend/go-gin` 中启动：

```bash
POWERX_PROXY=0 POWERX_PROVIDER_MODE=local go run ./cmd/plugin
```

此时插件使用 Local Provider；Knowledge Lab 使用已配置的 Gateway API Key 联调 Core。页面选择 Host 调试路线不会改变普通插件业务的 Provider。

不要仅用 `env -u POWERX_PROXY` 切换模式：配置加载器会为未设置的变量重新读取 `.env`，而进程显式设置的 `POWERX_PROXY=0` 优先于文件配置。

重启前先停止原插件进程。若终端只有 `exit status 1`，查看 `logs/runtime.log`；`listen tcp :8078: bind: address already in use` 表示旧进程仍占用端口，可通过以下只读命令确认：

```bash
lsof -nP -iTCP:8078 -sTCP:LISTEN
```

本轮未替用户停止或启动后台进程。2026-10-08 的检查与当前代码测试已通过，前端为 62 项单元测试及生产构建；页面视觉验收仍需使用加载新 Handler 的插件实例。
