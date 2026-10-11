# Go Framework 发布记录

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
