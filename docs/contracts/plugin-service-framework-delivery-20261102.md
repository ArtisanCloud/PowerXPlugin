# Framework 服务错误与 Gateway SSE 交付（2026-11-02）

本轮完成 Go 通用 Gateway、Capability 客户端的类型化错误，受控 Gateway SSE，以及 AI Runtime 的实际接线。权威正文和 Schema 已同步 Core；来源与摘要见 `plugin-authorization-source.json`，使用方式和边界见 `plugin-authorization-framework.md`。

## 可消费的开发快照

- Go SDK：`v0.0.29-dev.pluginservice.ad2129efb4087712`。
- CLI：`v0.0.0-pluginservice.b9aabc521150368b`。
- SDK 本地代理：`file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-plugin-service-20261102-2ff6526b92e5f5d5/go-proxy`。
- 生成器二进制：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-plugin-service-20261102-2ff6526b92e5f5d5/px-plugin`。
- 完整引用与构建摘要：`/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-plugin-service-20261102-2ff6526b92e5f5d5/references.json`。
- 当前 Git 基线：`32aaf0bd33ecda9860384484b55aa1d092ad27fc`；本轮改动未提交、未发布。

在消费项目的 Go 后端目录使用匹配快照：

```sh
GOPROXY='file:///private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-plugin-service-20261102-2ff6526b92e5f5d5/go-proxy,https://proxy.golang.org,direct' \
GONOSUMDB='github.com/ArtisanCloud/PowerXPlugin/framework/backend/go' \
go get 'github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.29-dev.pluginservice.ad2129efb4087712'
```

SCRM 当前的本地 `replace` 已指向本工作区；保留该声明时使用工作区源码。部署应固定匹配 SDK 引用。本轮未修改 SCRM 的 go.mod、业务参数或服务配置。

## 完成范围

| 范围 | 状态 |
|---|---|
| Go Gateway／Capability 错误与安全恢复信息 | 已实现并受控验证 |
| Gateway／Capability 受控 SSE、AI LLMStream 实际运行时 | 已实现并受控验证 |
| Skeleton、Host Contract 诊断、CLI 合同与生成模板 | 已同步；独立消费构建通过 |
| Nuxt | 本轮无 UI／协议修改；打包引用保留当前客户端代码 |
| .NET | 未同步，不计入本轮全语言完成 |

错误不依赖正文字符串解析。Core trace 与插件 trace 分开保留，恢复字段和版本诊断使用类型化值；`retryable` 不自动触发重放。SSE 固定使用 Core 的 `/tenant/invocations/stream`，不能由业务输入指定路由、身份头或 context；取消、超时和回调失败关闭上游，不换凭据、不重连。

## 验收与剩余联调

验收记录：`docs/contracts/evidence/plugin-service-errors-stream-framework-validation-20261102.json`。覆盖 API Key／STS、409／503 业务错误、运行版本诊断、SSE 成功、授权失败、非 SSE、重定向、取消和回调关闭，以及独立生成项目构建。测试使用受控 HTTP 目标，不代表现有安装态已完成这些调用。

Core 安装运行版本、发布／租户 registration／实时 grant，以及 SCRM 真实服务身份和消息链路仍需双方部署后联调。目标插件的服务票据验证客户端不在本轮补齐范围内。用户常驻服务、数据库和授权均未改动。
