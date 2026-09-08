# 场景：delegated 装配与安装验收

从[主指南](guide.md)进入本文。目标是插件业务消费与 local 相同的 Framework contract；不是让插件再次实现 Core HTTP 客户端。

## 1. 固定三类身份

| 凭证/上下文 | 谁验证、用途 | 禁止行为 |
|---|---|---|
| 插件入站用户凭证 | 插件 HTTP middleware，限制谁能发起业务操作 | 直接拿用户 JWT 代替 Core STS |
| STS service token | Core 校验插件、tenant、capability 与实际 grant | 从浏览器接收 STS，或把固定 token 写进源码 |
| Customer JWT | Customer Auth/Membership 的第二凭证，Core 验证 customer 与 tenant | 仅传 customer UUID、以 STS 代替 customer 身份 |

**动作**：bootstrap 使用现有已配置的 STS provider，向 typed client 注入可刷新 `Token(ctx)`；Skeleton 示例位于 `skeleton/backend/go-gin/internal/grpc/client/sts_token_provider.go` 及 `cmd/plugin/main.go`。独立插件不能导入 Skeleton 的 `internal` 包，应复用自己的宿主连接装配，或生成模板中的等价组件。

**预期**：token 获取失败时明确失败，业务出站不会改用另一凭证。**失败处理**：检查宿主连接、token audience/租户/插件身份；不打印原始 token 或 secret。

`BaseURL` 使用部署配置中的 Core Gateway 地址，推荐只填写 origin（例如本机开发 Core 的 `http://127.0.0.1:8077`，远程部署使用相应 HTTPS 地址），不要填 admin 页面 URL、插件 App Proxy 前缀或自行添加业务路径。不假定所有客户端拥有相同的超时默认值，应根据具体 SDK 配置和操作设置 timeout/context。

## 2. 按需声明能力并装配

**动作**：从主指南 §6.3 获取已确认能力，将完整 ID 放到实际 manifest 引用的 catalog 中。例如 Notifications：

```yaml
capabilities:
  required:
    - com.corex.capabilities.grant_status.read
    - com.corex.notifications.create
```

仓库示例清单位于 `skeleton/plugin.yaml`，它引用 `skeleton/plugin.d/capabilities.yaml`；独立插件使用自己的发布清单，不照搬路径和全部 required。不要改用旧 `requiredCapabilities` 或猜测 `consumes` 会自动同步授权。

bootstrap 顺序：

1. 校验可信 mode 和必要配置。
2. 构造 typed Core clients（主指南 §6.1.1），复用同一个 STS provider。
3. 加载并校验实际清单的 required；用 Capability client 调 `capability.RequireGrants(ctx, checker, required)`。它不授予权限，也不是业务调用的替代授权层。
4. 将 local/delegated adapters 传给模块 Factory，取得本部署必需 accessor；有错不启动该必需业务。
5. 业务只消费取得的 interface，继续传递请求 context。

完整代码与本地合同测试见 [examples/bootstrap.go](examples/bootstrap.go)。示例的 required 参数必须来自已校验清单，不是用户请求；示例只装配 Notifications，不能代替插件所有必需模块的预检。

**预期**：缺发布、缺 registration、缺 grant 均不会被当作“可用”。**失败处理**：修正声明及授权流程，不能删除真正需要的 required 绕过检查。启动检查成功后 Core 仍在每次业务请求检查实时授权，插件不得永久缓存为 allow。

## 3. 模块专有边界

- IAM：目录使用 STS；`ResolveIdentity` 另有被解析的用户 Bearer 输入，不能将目录 STS 本身当作人类身份。可信 tenant 参数仅作合同上下文/一致性约束，不由用户选择，也不序列化为 Host tenant 覆盖。
- Customer：Core Auth 的 Register/Login 使用 `Channel: "shopify_storefront"` 和 `CustomerCredential{Type: "shopify_customer_access_token", Value: ...}`；Core 必须配置可用 verifier。外部 identity resolve 不是用户登录，其最小结果不提供可靠 roles。Validate/Membership 使用 `WithCustomerCredential`；示例的 `ValidateCustomer` 演示请求级转交。SDK 的 local 注册字段不保证 delegated 支持；不要把 local 密码登录 UI 直接宣称兼容 Core Shopify 登录。
- Agent：按主指南 §7.2 执行独立 append/invoke/events/cancel；不再调用旧 `/agents/sessions`、`/agents/stream/sse`。Session STS-only，API Key 无法代替。订阅是已存在 invocation 的结果流，不是执行入口。
- Media：上传票据不是上传完成；按票据传输文件后再 complete，读取 variant 使用 variant_uuid，不暴露内部 object key。
- Knowledge：异步写入返回 job，不把 202/queued 当作索引完成。显式查询任务状态是合同操作，不是给 SSE 增加轮询降级。
- Plugin Release：Core 托管 signing key；调用方不能上传任意公钥自证。创建 job/session 不代表安装完成。
- Skills/Notifications/Plugin Runtime：仅主指南列出的有限操作；不等于完整 Skill 管理、通知中心或插件生命周期管理。

**预期**：无未声明 admin 路径、自由 tenant 覆盖或额外身份信任。**失败处理**：模块特有错误保留其 HTTP 状态、reason_code/trace（若有），由插件 locale 显示可恢复错误；不要把 `err.Error()` 直接作为用户文案。

## 4. 安装及真实验收（可与 local 开发分开排期）

**动作**：由部署人员确认 Core 所需合同已部署、capability 已发布且 tenant registration 存在；安装/升级/重新启用插件后核对当前实例实际 grants。仅重启插件不代表更新 grants，本文不要求插件团队擅自 seed、迁移或提权 Core。

先执行主指南 §7.3 的插件 status probe，预期返回模式与选中 adapter 状态；再调用有明确测试对象的只读业务操作。status 成功不是业务验收；写操作须单独确认对象、清理方式和风险。

记录每次请求的时间、插件与 Framework 版本、模式、capability、操作、trace（若有）、HTTP 状态和 reason_code，至少覆盖：

- 同租户已授权真实对象成功；无凭证/错误凭证拒绝。
- 有效凭证但未授予能力拒绝；跨租户资源拒绝或按合同隐藏存在性。
- 上游失败保留错误，不改查 local 表。
- 清单能力撤销后实际调用拒绝，旧 token 不能继续访问；grant-status 与真实调用一致。
- 模块专有场景：同名 ambiguous、任务失败、流终止/断连、显式取消等按消费范围选择。

API Key 仅对明确支持它的客户端/Host 操作做独立开发验证，不把远程 API Key client 注入 local adapter，也不把 API Key 结果当作安装态 STS 证据。Core P2 尚未完成核验的操作映射，按覆盖台账列为待确认，禁止自行拼接未知 capability。

**失败处理**：先定位是插件入站鉴权、Factory 缺依赖、STS 交换、Core grant 还是业务对象错误，再交给对应团队。无真实凭证时可完成 HTTP 合同测试并标记“代码验证通过”，不要编造安装态通过记录。
