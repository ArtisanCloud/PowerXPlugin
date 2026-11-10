# Framework SDK 2026-11-10 发布说明

发布日期：2026-11-10。Go、Nuxt Client 和 Nuxt Admin 已发布，并完成公共依赖的独立消费验证。

| 交付 | 版本 | 获取方式 |
|---|---|---|
| Go Framework | v0.0.29 | `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go` |
| Nuxt Client | 0.0.15 | `@artisan-cloud/plugin-framework-client` |
| Nuxt Admin | 0.0.14 | `@artisan-cloud/plugin-framework-admin`，peer Client 固定为 `0.0.15` |

SDK 对应 [PR #96](https://github.com/ArtisanCloud/PowerXPlugin/pull/96) 的 develop 合并提交 `ebbf423af65de561a4f1dd7776558551bad01f9b`。Go tag 为 `framework/backend/go/v0.0.29`；Nuxt 的包内容来自同一提交，两个包的 npm `latest` 已更新。后补的默认依赖和发布记录不改变 SDK tag。

## 本次主要更新

- AI：补齐模型目录、连接测试、本地模型目录、媒体任务与可选图像输入限制；后端和前端以结构化合同消费能力与错误。入口见 [AI 连接测试](../contracts/ai-connection-test-framework.md)、[本地目录](../contracts/ai-local-model-catalog.v1.md) 和 [模型图像限制](../contracts/ai-media-model-limits-framework.md)。
- LLM 流式调用：通过 `hostcontract.InvokeStream` 调用正式 `com.corex.ai.llm.stream` 能力，校验 `done.ok=true`，再产生统一 `final`、`end` 事件。修复过时测试帧，不放松生产协议。见 [流式合同](../contracts/ai-llm-stream-service-v1.md)。
- 用户会话：HTTP、WebSocket、SSE 的结构化失效报告绑定当前会话快照；Admin 能力调用复用共享客户端。宿主确认与草稿清理边界见 [会话终止接入](../contracts/plugin-session-termination-framework.md)。
- 运行时：对齐 Event Fabric、TaskQueue v2、服务授权和错误透传，补齐模板与生成插件的接入示例。见 [Event Fabric](../contracts/event-fabric-framework.md) 和 [TaskQueue v2](../contracts/runtime-taskqueue-v2.md)。
- 客户认证：包含 PR #95 的微信小程序委派登录接入与校验。使用时需要相应 Core 能力、授权和渠道配置。
- 工具链：更新脚手架依赖选择、插件 ID 校验和本地 Publisher 基础。CLI 源码变化随仓库提供，本次发布对象为上述三个 SDK 包。

Skeleton 和模板中的 adapter、路由、locale 及配置仍由插件拥有。升级 SDK 不会自动替换既有插件内的这些文件。

## 验证结果

- [PR CI](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/42739204986)：14 项全部通过。
- [develop CI](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/42741485256)：最终 attempt 2，14 项全部通过。
- 公共 Go proxy 下载 `v0.0.29`，通过 `sum.golang.org` 校验；模块来源为上述合并提交，校验值与发布前冻结候选一致。
- npm 精确版本 metadata、tarball integrity 与发布前候选一致；Admin 的 Client peer 为 `0.0.15`。
- 仓库外生成新插件，关闭 Go workspace、无 Framework 本地 replace、无源码 symlink，使用独立 Go/npm 缓存，从公共源安装三个精确版本。生成后端测试、打包 AI 合同测试、后端构建、TypeScript 导出检查和 Nuxt 生产构建全部通过，临时消费者已清理。

机器可读结果见 [发布证据](../contracts/evidence/framework-publication-20261110.json)。

## 消费方升级

后端在自身模块目录执行：

```sh
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.29
go mod tidy
GOWORK=off go test ./...
```

Nuxt 管理端在自身项目目录执行：

```sh
npm install --save-exact @artisan-cloud/plugin-framework-client@0.0.15 @artisan-cloud/plugin-framework-admin@0.0.14
npm run build
```

正式依赖验收需移除该模块的本地 `replace` 与前端 `file:`/workspace 引用。按插件自己的发布流程提交 lockfile、构建和部署。

`LLMStream` 需要 Core 支持正式 `core_internal` 流式能力，并显式授予服务调用权限；原 REST 权限不能代替此授权。Host Bridge 需要 session v2 快照和 `reportSessionInvalid` 能力；缺失时明确失败。既有插件需按接入合同调整自身 adapter、身份绑定和草稿管理，不增加旧协议兜底。

本次验证覆盖发布、安装、合同与生成插件构建。Core/插件的部署、真实模型、微信设备登录、正式数据迁移与生产业务验收仍由对应消费方完成。本次未发布 CLI 二进制或 .NET NuGet，未变更 Core、业务数据库或用户运行中的服务；并行 Publisher 工作区改动不属于本次 SDK 包。
