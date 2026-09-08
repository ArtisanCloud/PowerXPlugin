# 场景：插件实现 local adapter

入口与通用约束以[主指南](guide.md)为准。本文适用于新插件及已有 local 表的插件；不要求连接 Core，也不提供替代插件真实业务的空 Store。

## 1. 选定合同与依赖

**动作**：完成主指南 §6.0，列出本插件实际消费的模块及操作。业务所需的 contract、Factory accessor 与 delegated 构造入口均在主指南 §6.1。

**命令**：在插件后端目录执行，例如：

```bash
go doc github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/notifications.Publisher
go doc github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/notifications.CreateInput
```

**预期**：依赖可解析，方法及字段与本次接入基线相符。**失败处理**：核对 go.mod/go.work/replace，不通过增加旧签名 overload 或降级依赖绕过。

## 2. 将现有存储包成正式 adapter

建议在插件中分开组织（以下目录是建议，不是 Framework 强制路径）：

```text
internal/adapters/local/<module>/   # 实现 Framework interface，校验身份和隔离
internal/repository/               # 本地存储与事务
internal/bootstrap/                # 一次性选择模式、构造/注入依赖
internal/services/                 # 只依赖 Framework interface
```

**动作**：为每个 adapter 添加编译断言，并将原有 model 转成正式 DTO。例如 `var _ notifications.Publisher = (*LocalPublisher)(nil)`。DTO 所在 `runtime/powerx/*` 包可以被 local 导入；不要复制一份同名 DTO。

必须实现：

- 由已验证 context 提取 tenant/调用主体，按 tenant + 对象 UUID 查询；无身份先拒绝，不执行 SQL。
- 本地事务、唯一约束、并发和幂等；Framework 不负责插件的建表迁移。
- 分页、批量缺项、取消与错误语义按合同处理。不能将未实现方法返回 nil/空集合作为成功。
- 明确转换历史 numeric 关联为 UUID；保存身份引用而非显示名称。local 与 delegated 的历史 UUID 不保证相同，不能仅切 mode 就让原有本地引用自动变成 Core 对象。

**预期**：业务代码不再直接选择本地 Repository 或 Core URL。**失败处理**：暂不支持某项时返回明确的模块错误，并将操作列入插件未支持清单；不嵌入 nil 接口伪造完整实现。必需操作未实现则该模块不能宣称验收通过。

## 3. 启动装配与业务注入

**动作**：在 bootstrap 中使用已解析的 `provider.Mode` 调用对应 `NewRuntime`，立即取得必需 accessor，检查错误，然后将 interface 注入业务 Service。仅构造 Runtime 不代表依赖完整。

完整的可编译函数见 [BuildPublisher](examples/bootstrap.go)：它接收真实 local Publisher，不自行创建本地表；delegated 分支才构造 Core 客户端。该分支只出现在 bootstrap，业务层没有 mode 判断。

装配特例：

- IAM：`Registry.Bind` 一次绑定 Directory/Authz/Context 三项，参数类型是 `contracts.IAMAdapterMode`；不能只传 Directory。
- Media：Factory 需要完整 `Service`，业务可以只拿 `Assets()`；这不是只读 adapter 注入口。
- Agent：仅会话业务可 `NewRuntime(mode, nil, nil, WithSessions(localSessions, nil))`，只取得 `Sessions()`；无需补造生命周期实现。
- Customer：`AdaptersFromLocalStore(store)` 将同一个真实 Store 拆成 Auth/External/Membership；也可显式提供三组 adapter。不要再让业务层读取独立的 customer_auth.mode。
- Knowledge：实现 `KnowledgeProvider`，如实声明 Capabilities；无法支持的操作必须失败，不能伪报支持。

**预期**：业务仅调用例如 `Publisher.Create(ctx, input)`。**失败处理**：缺必需 accessor 使启动失败；可选功能明确不可用，不影响其他已完成模块。

## 4. 验证与旧链路退出

在 **PowerXPlugin 仓库根目录**执行本文档示例测试：

```bash
go test ./docs/guides/features/009-consume-powerx-capability/examples/*.go -count=1
go test ./framework/backend/go/runtime/notifications -run '^ExampleNewRuntime$' -count=1
```

这些测试验证装配和失败边界，不验证你的数据库。随后在**插件后端目录**执行：

```bash
go test ./... -count=1
```

插件至少补：同租户成功、跨租户拒绝、缺身份不落库、选中 adapter 缺失、typed-nil 拒绝、403/上游失败不切模式、分页与异步/幂等边界。对未选中的 adapter 设置调用计数或失败哨兵。

验收时从真实业务 handler 追踪到 Service → Framework interface → local adapter → Repository，不能只测试 Factory。删除旧的业务 mode 分支、Core 直调和 UUID 显示 fallback；同步维护所有 test fake。迁移数据前先备份并验证，本文不授权删除现有表或重置数据库。

将结果按主指南 §8 记录。通过后可标记“该插件 local adapter 已对齐”，不能据此标记 delegated 安装态通过。
