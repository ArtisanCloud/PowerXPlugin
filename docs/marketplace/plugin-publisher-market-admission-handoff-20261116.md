# Base 发布包准入问题：交给 Market 的修复与联调任务

**2026-11-17 更新：本文保留 Nov16 的历史失败及参考方案。Market 已自行修复并交付，Framework 未修改 Market；同一原包已通过 Framework 网页上传和送审。当前接续任务见[人工审核协作步骤](plugin-publisher-base-review-coordination-20261117.md)。**

日期：2026-11-16。协议仍为 `powerx.plugin-publisher/v1` / `1.0.0-rc.2`，本问题不要求升级协议或发布 Framework SDK/CLI。

**当前分工：Framework 完成自己的发布页面、构建、客户端适配和后续页面复验；Market 维护者修复市场准入代码，并交付可联调的隔离市场。** 本文和参考补丁保存在 Framework 仓库，供市场侧 Review、实施和回传结果。Framework 交付问题和参考补丁，Market 维护者 Review 并修改自身仓库。

## 1 已完成与当前阻塞

| 环节 | 已核实结果 | 责任方与下一步 |
| --- | --- | --- |
| 发布页面 | 自动准备并连接本地执行器，可在页面选择版本、启动 Linux 容器构建和绑定发布凭据 | Framework 已实现并完成隔离页面验证；用户发布流程不需要额外运行 publisher CLI 或填写配对变量 |
| Base 安装包 | 实际 `make dist` 完成，Linux/amd64 ZIP 通过 Framework 准入；解压该 ZIP 后，在 Linux 容器启动其中的 Nuxt，HTML 与实际 JS 资源读取通过 | Framework 保留原冻结包，后续复验使用同一 SHA256 |
| 授权 | 在真实隔离市场网页签发凭据，经发布中心密码输入框绑定至 macOS 系统凭据库 | 本轮测试凭据与临时环境已清理；下轮使用市场提供的新隔离交接信息 |
| 上传 | 页面发起真实上传，发送完整 ZIP 后收到市场 `package_invalid` | Market 修复 P04 资源引用误判并交付结果 |
| 送审及后续 | 本轮真实 Base 包尚未取得上传成功回执；未完成该包的送审、驳回重提、批准或上架 | 市场修复交付后，Framework 接续页面验收；审核和上架动作由市场侧负责 |

完整事实和截图见 [GUI 验证记录](../contracts/evidence/plugin-publisher-gui-validation-20261116.json)。此前受控包 M2 的通过记录是独立历史证据，不能替代本轮真实 Base 包的上传与审核结果。PowerX 安装启用、正式市场公开下载、Linux Secret Service 和独立已发布 SDK/CLI 消费仍分别保留验收边界。

## 2 交接输入及失败证据

| 项目 | 本轮值 |
| --- | --- |
| 插件 | `com.powerx.plugins.base` / PowerX Base Plugin |
| 源码声明版本 / 候选包版本 | `0.1.4` / `0.1.5`；源码版本未被改写 |
| 目标 | `linux/amd64`，Go `CGO_ENABLED=0`，Node Linux/x64 |
| ZIP 大小 / 文件数 | `60,868,645` 字节 / `544` |
| ZIP SHA256 | `e7a9f5ef7d883633d2826011946c5f4201a2ef7ba77cc6ba1be5d08190c342f3` |
| 冻结包 | `skeleton/.px-plugin/publisher/work/7352fbba-b6ca-481d-9699-8650f5e4a52b/artifact.zip`（相对 Framework 仓库根目录）；原包在本机保留并被 Git 忽略，跨机器转交需另行复制和核对摘要 |
| 上传入口 | `POST <market_api_root>/publisher/plugins/com.powerx.plugins.base/package`，`market_api_root` 已含 `/api/v1` |
| 客户端上传任务 UUID | `5278f126-cbc3-4ae6-830c-660239466cfc` |
| 本轮 operation UUID | `d1405a57-d0ca-4274-bc26-87a2739584d2` |
| 市场 request UUID | `2211121f-0f3e-4aaa-859f-22986b52aa47` |
| 返回 | 客户端 `PUBLISHER_MARKET_REJECTED`，市场原因 `package_invalid`；任务终态 `failed`，发送字节数与 ZIP 大小相同，成功回执为空 |

旧隔离 API/console 地址已关闭，临时数据库、对象桶、凭据和执行器均已清理。本轮失败记录保留在验收 JSON 中，不再依赖已清理的临时工作区。客户端未取得可用于后续送审的 Version/Artifact UUID，不把本地构建成功当作市场版本已创建。

## 3 原因和建议修复

市场位置：[backend/internal/publisher/admission/runtime.go](../../../../PowerXPluginMarket/backend/internal/publisher/admission/runtime.go) 的 `references()`，由 `validateRuntime()` 对清单及 YAML 资源调用。

Nov16 失败时的实现递归检查所有字符串，把以 `schema/` 开头的字符串统一解释为文件引用。例如真实契约中：

```yaml
provides:
  - id: schema/input/com.powerx.plugins.base.template.create.v1
    path: schema/input/com.powerx.plugins.base.template.create.json
    kind: input
    version: 1.0.0
```

`id` 是 Schema 标识，`path` 才是资源路径。包内存在 `contracts/schema/input/com.powerx.plugins.base.template.create.json`；将 `id` 变成 `contracts/schema/input/com.powerx.plugins.base.template.create.v1` 并查找文件，会误报 `package_invalid`。真实输入可核对 [template.create 契约](../../contracts/capabilities/com.powerx.plugins.base.template.create.yaml)。

建议市场侧按契约字段区分标识和文件引用：

1. 对文件引用字段执行存在性和路径检查。参考实现覆盖 `path`、`schema`、`schema_ref`、`template`、`endpoint`、`channel`、`payload`、`handler`、`imports`、`descriptor`、`input`、`output`、`entrypoint`；对象和数组继续按结构递归，字段集合由市场侧结合正式契约 Review。
2. `id` 和说明文本不作为文件引用；实际路径继续使用现有 `hasPath()` / `safePath()` 校验，并保留 `./` 相对路径的规范处理。
3. 保留缺失引用、路径逃逸、链接、重复/冲突条目、manifest 256 KiB、ZIP 50000 条目、解压预算、压缩比、ELF/Node 目标及版本等独立准入检查。修复的是引用语义，其他门禁继续生效。
4. 使用原契约与原冻结 ZIP 完成复验，保持包大小和 SHA256 不变。

[参考补丁](../contracts/evidence/plugin-publisher-market-schema-reference-fix.patch) 仅供市场维护者 Review：修改 `backend/internal/publisher/admission/runtime.go`，新增 `backend/internal/publisher/admission/schema_reference_test.go`。市场原仓库未应用该补丁。补丁基于本轮核对的文件摘要：

```text
原 runtime.go SHA256:
a692a9a4ead0e7cd90d9cd0ec8e3a4461e3145483a005ce6e7e9ac27ef4370ad
私有副本中的建议 runtime.go SHA256:
572dd3b71097d1ddad2469924f9f1a896d789c5334cb0f7b8df65114e4562b13
```

若市场代码已经变化，请按当前实现 Review 后修改，不直接套用旧上下文。Framework 已在私有副本中复现“原代码拒绝同一真实 ZIP”，并验证“建议修复接受同一 ZIP”；建议修复同时通过该副本内的市场原有 admission 测试、Schema ID 回归测试及 race 检查。这不代表市场原仓库或运行服务已修复。

## 4 市场侧任务及交付清单

由 **Market 维护者** 完成代码修改、市场回归、隔离实例准备及市场文档同步。

必须覆盖的回归至少包括：

| 场景 | 预期 |
| --- | --- |
| `provides[].id` 为 `schema/...v1`，对应 `path` 的 JSON 存在 | 接受标识，校验真实路径 |
| 删除 `path` 对应文件，保留相同 `id` | 拒绝 |
| `schema: contracts/missing.json` 等嵌套引用不存在 | 拒绝，保留原有 `missing nested contract` 回归 |
| 引用带 `./`，或 `imports` 数组中有缺失项 | 正确规范路径并拒绝缺失项 |
| 原有 ZIP/manifest/Go/Node 安全和预算用例 | 原有门禁保持有效 |
| 本文指定 SHA256 的 Base 原包 | 独立准入通过；真实上传返回可靠回执及可查询的 Version/Artifact UUID |

以下是市场维护者的开发检查，不属于插件开发者的网页发布步骤：

```bash
# 在 Market/backend 下，使用项目要求的 Go 工具链
GOWORK=off go test -race ./internal/publisher/admission
GOWORK=off go vet ./internal/publisher/admission
```

回传 Framework 时，请给出：

- 修改文件清单、提交或工作树摘要、回归命令及结果；不要求为此先发布新版协议。
- 同一 Base ZIP 的独立准入及真实上传结果，包含包摘要、大小、operation/request UUID，以及服务端返回的 Version/Artifact UUID。
- 新的隔离 `api_root`、批准的 `console_base`、授权入口、测试插件归属、环境有效期及维护人；敏感登录信息使用本机私密引导文件，不写入聊天、文档、截图或日志。
- 真实 discovery 能力和凭据权限。上传需要 `plugins:read/packages:write`；送审另需 `publications:read/publications:submit`，并开放相应 `package-upload/idempotent-writes/publication-submit` 能力。
- 市场侧审核维护者负责后续驳回、批准、明确上架的操作，记录对应申请和结果。
- 同步 Market 仓库的 `docs/marketplace/plugin-developer-publishing-center-plan.md` 与 `plugin-publisher-implementation.md`：将“等待 Framework 跨仓库代修确认”改成“Market 负责修复交付，Framework 接续客户端验收”。本轮 Framework 只维护自己的文档副本，市场副本的同步由市场侧交付确认。

## 5 Framework 收到交付后的接续任务

由 **Framework 维护者** 在自己的开发页面完成以下操作并记录证据；发现客户端问题时修改 Framework 自身实现。

1. 读取市场的新隔离交接信息，通过页面添加连接、绑定凭据并检查能力。日常 Web Admin 若仍加载旧模块，由该日常环境的维护者重新运行原开发任务；Framework 不擅自重启日常服务。
2. 选择已就绪的 Base `0.1.5` 构建，复核本文的冻结 ZIP 摘要与大小，使用同一包执行页面上传。
3. 本轮旧 operation 已有明确的失败终态；修复后复验创建新的上传任务和 operation，保持 ZIP 字节不变。若遇到的是丢响应或结果未知，则先查询并恢复原 operation，不能以新任务绕过未知结果。
4. 取得可靠上传回执和 Version/Artifact UUID 后，读取市场资料选项与该版本的完整实际制品集合，用户确认后提交审核。`expected_artifacts` 使用市场返回的全部制品 UUID、平台、SHA256 和大小。
5. 配合市场维护者刷新待审核、已驳回、审核通过待上架、已上架四种状态；驳回重提创建新任务、operation 和 Publication UUID，原申请资料、制品集合及回执保留，旧链接继续指向旧申请。补充本轮真实 Base 包的截图与请求记录。
6. 下载、PowerX 安装和启用按各自验收条件另行执行。只有取得对应证据才更新验收结论；保留历史受控包证据，SDK/CLI 发版仍在插件版本验收之后。

## 6 可直接转交 Market 的说明

> Framework 发布中心已真实构建 Base 0.1.5 Linux/amd64 完整包，并验证包内 Nuxt 启动。页面上传完整发送 60,868,645 字节后，P04 返回 package_invalid，request UUID 为 2211121f-0f3e-4aaa-859f-22986b52aa47。原因是 admission/runtime.go 的 references() 把 provides[].id 中的 schema/...v1 当作文件路径；该条目的 path 所指 JSON 实际存在。请市场侧按引用字段修复语义并补回归，保留全部包安全门禁。参考补丁已在私有副本通过市场 admission 测试和同一真实 ZIP 校验，未应用到 Market 原仓库。请按本文第 4 节交付修复、隔离入口及验证结果；Framework 收到后继续自己的网页上传、送审和状态验收。
