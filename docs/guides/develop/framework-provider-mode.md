# Framework Provider 模式

插件统一从 [业务模块接入指南](../features/009-consume-powerx-capability/guide.md)接入。本文只解释模式，不假定所有管理页面均支持 delegated。

## 模式与可信配置

- local：Factory 选择插件注入的 local adapter，由插件维护该模式的业务数据。
- delegated：Factory 选择 Framework Core Host adapter，不访问插件 local 业务表，也不调用 Core Admin API 替代 Host 合同。

provider.ModeResolver 使用配置 context.provider_mode 与 POWERX_PROVIDER_MODE；均设置但不一致时失败，均未设置当前默认 local。安装态应显式核对模式；POWERX_PROXY 或浏览器参数不能代替模式确认。详情见主指南 §6.2。

Factory 在启动时绑定模式，不负责创建 Store、获取凭证或授予 capability。必需 accessor 必须在 bootstrap 检查；运行时错误不触发模式切换。

## 页面与 diagnostics 边界

前端调用插件自己的 HTTP API，不持有 Core STS/API Key、不选择数据权威源。两种模式可以保留相同页面路径，但必须确认 handler 真正消费 Runtime；不能由路径相同推断功能可用。

Host Contract Lab 的 status probe 只证明 adapter 装配；真实调用和授权单独验证。framework-lab 用于基础设施诊断。范围见 [覆盖台账](../../contracts/powerx-core-framework-coverage.md)，不把 AI Settings 或 Customer Admin 自动计入 AI Generative / Customer Auth 合同。

## 验证

在 PowerXPlugin 仓库根目录执行：

```bash
go test ./framework/backend/go/runtime/provider ./framework/backend/go/runtime/module -count=1
```

预期模式冲突、缺 adapter、typed-nil 和禁止隐式降级测试通过；不替代插件数据隔离和安装态验收。缺依赖按实际模块机器码返回，不要求所有模块都有同一个 /mode HTTP 路由。
