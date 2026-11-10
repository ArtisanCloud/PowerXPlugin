# Event Fabric 服务 Runtime

应用合同：`event_fabric.service.v1`。完整接法与验收边界见仓库 `docs/contracts/event-fabric-framework.md`。

- `Client.ResolveTopics`：用目标租户实际服务凭据，只读解析发布或消费方向的业务 key。
- `NewTenantHostRuntime`：解析并严格校验所有映射后装配独立租户 Runtime。
- `RuntimeSlot.Switch` / `Close`：单个租户上下文切换时取消旧请求、关闭订阅和 Runtime，失败后不保留旧租户。
- `Runtime.TopicBindings`：取得物理映射副本，不能修改运行中的路由。
- `NewHostRuntime`：仅接受明确凭据租户及其静态映射，不支持无租户通配。

业务 key、原事件信封、稳定事件／幂等身份和提交后 ACK 规则保持。调用方负责业务事务与持久去重。没有 Admin、WSBus、本地队列、System 凭据或拼接 Topic 的兜底。

单测使用受控 HTTP/SSE 服务。真实 Core 安装、凭据授权、业务插件升级与部署联调分别验收。
