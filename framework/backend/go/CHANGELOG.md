# Go Framework 发布记录

## v0.0.26 — 2026-10-19

发布模块 tag 为 `framework/backend/go/v0.0.26`。发布状态与正式依赖验证见本版本发布记录。

- 新增独立 IAM 连续授权基线批量合同、严格响应校验、typed Core client 和可选 Registry／Bundle 入口；local 权威存储继续由插件注入。
- WebSocket 在最终发送前复核当前身份与 topic，并阻断到期、撤权、取消及迟到结果。
- Media Host 响应保留可选标准图片 `derivatives` 快照，区分原图上传与预览处理状态；既有 Service 接口保持不变。
- 通知支持可选业务导航、动作响应和独立已读回执；通知创建／回执共享配置的 Core API 前缀，本地参数、上游合同和未配置客户端分别返回明确错误。
- 新增业务回归运行时、场景／断言／证据合同及可选存储适配；配套 client/admin 使用各自 npm 版本。
- 更新 SDK 的生成默认版本；插件仍需接入启动装配与所选 provider，升级 SDK 不会覆盖插件已经生成的内部实现。

本地打包、正式发布、真实 Core grant 和安装态验收分别记录，不能互相替代。当前候选包及验证边界见 [2026-10-19 发布准备](../../../docs/releases/framework-20261019.md)。

## v0.0.25 — 2026-10-12

发布模块：`github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`。

- 新增独立 `contactfw.PrimaryContactManager.SetPrimary` 和对应 DTO，通过固定 `core_internal` 能力更换主联系人；原六项 `Store` 接口保持不变。
- 识别 `CONTACT_PRIMARY_*` / `CONTACT_REQUEST_INVALID`，保留 Core 实际 HTTP 状态、Trace、request ID 和原始错误；普通联系人创建/编辑不能绕过设主操作。
- Customer/Contact 姓名按 Unicode codepoint 校验，拒绝超长、控制字符和非法 UTF-8；更新请求在 JSON 编码前校验，避免非法编码被替换后漏检。
- AI 模型 DTO 保留 Core 的 `modality` 和 `capabilities`；测试使用正式 `tenant_profile` 来源枚举。
- 新增知识内容导览的类型、可选 Provider 接口及固定 Host 客户端，校验文章预览、语料版本和分页响应。本项真实 Core／浏览器验收仍待完成。
- Skeleton Local 设主事务维护 membership 主引用、角色和审计，保护当前主联系人；Local 模型目录补齐文本生成声明。相关 Go 生成模板已同步。

本版本已通过 `framework/backend/go/v0.0.25` 发布，指向 CI 通过的 develop 合并提交 `fea5e58a`。Go proxy 下载、发布源码核对和无本地替换的生成项目测试通过。CLI/Skeleton 已预置 v0.0.25；消费方升级后需重新构建插件。Core 精确授权和业务 HTTP 状态透传仍需目标环境验收。

详细变更、测试记录与升级步骤见 [v0.0.25 发布说明](../../../docs/releases/framework-20261012.md)。v0.0.24 的已发布内容和证据见 [上一版发布说明](../../../docs/releases/framework-20261011.md)。

## v0.0.23 — 2026-10-07

发布模块：`github.com/ArtisanCloud/PowerXPlugin/framework/backend/go`。

### 接口与 Runtime

- 对齐 PowerX IAM 成员数字状态：Delegated transport 接收数字，公开成员模型保留 `"1"`/`"2"`；容错批量解析按 Core 精简响应接收。状态解析错误和授权错误分别处理。
- 提供 Customer/Contact 的强类型 local/delegated 接口，覆盖客户资料、联系人及外部身份查询、绑定、新建并绑定。
- 扩展 Knowledge Host 的强类型配置目录与空间创建接口，校验 UUID、配额和响应结构；入库、任务和查询沿用固定 Host 能力调用。
- 对齐 Metadata 标签更新 DTO，包括多语言和资源类型字段。
- 提供 Agent/Skill 管理接口及启动时选定 local/delegated 的 Runtime Factory；具体 local 持久化与执行由插件 adapter 实现。
- 补充 Runtime Identity 客户端、能力调用错误中的 reason/trace/request 信息，以及 Cache、TaskCenter 等 Runtime 合同与测试。
- 补齐独立 Go 模块的依赖校验记录，支持关闭仓库 workspace 后构建和测试。

### 插件升级

在插件后端 Go 模块目录执行：

```bash
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.23
go mod tidy
go test ./...
```

如果插件使用本地 `replace` 或 `go.work` 引用 Framework，运行时仍会使用本地源码。验收发布版本时应在独立 checkout 中关闭 workspace，并移除该模块的本地 replace。

成员公开状态为字符串数字，`"1"` 表示启用、`"2"` 表示停用。插件使用 Framework 的成员接口，不自行解析 Core 的数字状态。

### 验收边界

本版本通过 Go Framework 全量测试与独立模块检查。测试通过不替代插件自己的真实 API Key/STS、capability grant、数据库迁移和安装后验收。升级后需重新构建并重启插件；Core 授权、能力版本锁或缺失 Host 合同仍会明确返回错误。

前端 npm 继续使用 `framework-client 0.0.11`、`framework-admin 0.0.10`；本次不发布 .NET 包。
