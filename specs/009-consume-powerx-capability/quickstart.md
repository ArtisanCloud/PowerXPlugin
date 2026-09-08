# Quickstart：Framework 业务模块接入

**唯一对外操作指南：[Framework 业务模块接入指南（local / delegated）](../../docs/guides/features/009-consume-powerx-capability/guide.md)。** 本文件仅保留需求侧导航，不维护重复代码样例。旧的拼接 Media URL、packages/backend 路径和通用 Gateway 替代业务 contract 示例已撤下。

## 实施顺序

1. 按主指南 §6.0 确认插件实际 Go module 版本/replace，选择所需业务接口；npm 版本不代表 Go SDK 版本。
2. [local 场景](../../docs/guides/features/009-consume-powerx-capability/usecase-local-adapter.md)：插件实现 local adapter 与持久化，Framework 负责 Factory 单选，业务层只依赖 contract。
3. [delegated 场景](../../docs/guides/features/009-consume-powerx-capability/usecase-delegated-runtime.md)：bootstrap 构造 typed Core clients、绑定 STS、检查 required grants，取得必需 accessor。
4. 编译所有生产 adapter 和 test fake，执行主指南的示例及插件自己的业务/数据隔离测试。
5. 按实际消费范围进行安装态验收；Core 合同已部署不代表该插件已有 grant，API Key 结果不能代替 STS。

## 本地文档示例验证

在 PowerXPlugin 仓库根目录执行：

```bash
go test ./docs/guides/features/009-consume-powerx-capability/examples/*.go -count=1
```

预期：local/delegated 选中正确实现，grant 拒绝阻断装配，业务 403 不回退 local。测试使用 HTTP fixture，不是安装态验收。

## 范围与证据

- 规则：[双模式规范](../../docs/guides/develop/framework-dual-mode-business-modules.md)。
- 完成状态：[覆盖台账](../../docs/contracts/powerx-core-framework-coverage.md)。
- 未完成工作：[tasks.md](tasks.md)，安装态、Core P2 与套件插件消费分别跟踪。
- 所有写操作须明确测试对象与清理方式，指南不授权 seed、迁移、重启或自动发布。
