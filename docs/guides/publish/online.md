# 插件在线发布与市场上传指南

核对日期：2026-11-09。本文区分当前源码与开发者发布中心规划，不声明真实市场上传、SDK 发布或宿主安装已经验收。

统一规范见[开发者发布中心与市场对接方案](../../marketplace/plugin-developer-publishing-center-plan.md)及[实现设计与命令迁移](../../marketplace/plugin-publisher-implementation.md)。市场侧维护同版文档；正式 publisher OpenAPI 尚待 P01 冻结。

## 1 当前能力

| 入口 | 当前源码行为 | 使用边界 |
| --- | --- | --- |
| `make dist` | 构建并组织安装目录，写产物版本和 target，检查 Go 二进制目标 | 在生成插件根目录执行；Framework 仓库会转发到 skeleton |
| `make package` | 先执行 dist，再压缩版本目录为 ZIP | 可生成本地产物，不会上传或送审 |
| Go `px-plugin package` | 独立打包为 `.px-plugin/build/<时间戳>/package.tar.gz` | experimental，布局与 Makefile 不同，尚未统一为新发布引擎 |
| Go `px-plugin dist` | 输出 experimental 提示 | 当前没有实际打包实现 |
| Go `px-plugin publish` | 发送 JSON 至 `/internal/plugins/releases`，携带本地文件 URI | 不向开发者市场接口传送真实文件字节，不用于新发布中心 |
| TypeScript publish | 本地生成发布 ID、上传地址和回执 | 不代表已签名、已上传或已进入市场审核 |
| Go `px-plugin publisher plan/serve` | 本地 UUID 计划及独立会话服务，实际命令见[本地契约](../../contracts/plugin-publisher-local-v1.md) | 已实现基础及 F03 首个隔离构建配置，见[构建契约](../../contracts/plugin-publisher-build-v1.md)；真实 Linux/Nuxt 运行、市场上传、送审和页面代理尚未验收 |
| 市场开发者网页 | 浏览器登录后上传文件、管理草稿及提交审核 | 使用现有会话、CSRF 和归属校验；批准与上架分开 |

对应源码：[构建规则](../../../skeleton/make-files/build.mk)、[Go publish](../../../tools/cli/cmd/publish.go)、[HTTP 客户端](../../../tools/cli/internal/publish/client.go)、[TypeScript 流水线](../../../tools/cli/src/lib/publish/pipeline.ts)。

## 2 当前准备及手工上传

在已安装依赖、工具链和配置就绪的生成插件根目录，构建单个平台包。例如在 Linux 执行器中：

```bash
make dist VERSION=1.2.0 PLATFORM=linux TARGET_ARCH=amd64
```

确认本次 dist 的插件 ID、版本、target、后端及迁移程序正确，前端完整运行时也适用于 Linux。当前目标脚本检查 Go 构建信息，不表示已实现全部原生 Node 依赖验证。macOS 上交叉编译 Go 成功不能直接作为完整 Linux 包验收。

需要使用现有 ZIP 打包入口时执行：

```bash
make package VERSION=1.2.0 PLATFORM=linux TARGET_ARCH=amd64
```

此命令会再次触发 dist，不是对先前冻结包的无副作用重试。F03 新发布引擎已直接压缩独立任务 staging，配置限制及真实运行验收边界见[构建契约](../../contracts/plugin-publisher-build-v1.md)。当前需核对最终 ZIP 的实际字节、大小和 SHA256，避免把不同构建当成同一次重传。

之后使用已配置市场的开发者网页：

1. 登录具有有效开发者身份的市场账号，选择已经登记且归属自己的插件。
2. 上传选定包及更新说明，明确 OS 和架构。当前文件上限为 2 GiB；包内插件 ID 和版本由市场解析。
3. 核对网页中的版本、平台包和更新说明。上传登记的是私有草稿，不会自动进入公共目录。
4. 资料齐备后按[审核与上架指南](marketplace-review.md)提交审核；人工批准后仍须显式上架。

本地文件 URI、宿主 API Key 和插件业务登录态均不能替代市场上传及市场身份。

## 3 开发者发布中心的目标流程

以下为待实现流程，不是当前可运行命令：

```text
本地版本计划 → 预检查 → 构建 → 冻结打包 → 完整目标校验
                                           ↓
                   独立市场发布凭据 → 上传真实 ZIP 字节
                                           ↓
                            市场登记私有草稿 → 状态回显
                                           ↓ M2
                          提交审核 → 人工批准 → 显式公开上架
```

页面及 CLI 将复用 Go 引擎，页面通过开发同源代理访问回环执行器。生产构建不注册发布页面和执行接口。首期支持一个已登记插件、Linux 执行器的 `linux/amd64`、正式 SemVer 和 stable。

市场 API root 下的 `/publisher`、`powerx.plugin-publisher/v1` 及发布令牌仍属待实现机器协议；需要正式 OpenAPI、scope、预算和 fixture 就绪后接入。不把旧 `publish create/deploy`、内存 Plan 记录或本地回执当作该链路。

## 4 失败恢复与完成判据

- 本地构建完成只表示产物就绪；传输达到 100% 后显示市场校验中。
- 市场成功要求实际包已保存、版本及 Artifact 已登记、操作结果可靠保存，并能查询核对。
- 网络超时、取消上传、丢响应后进入结果待核对；先查询原 operation，再核对版本及包摘要。
- 必要重传使用原 operation UUID、同一冻结文件和同一请求内容；不换 key 覆盖包，不重新压缩后冒充原请求。
- 首期恢复任务及服务端结果，必要时整包重传；不提供字节级断点续传。
- 新协议不可用时明确禁用远端动作，保留本地构建与导出；可显式进入市场网页，不自动调用旧内部接口或更换凭据。

当前 Go/TypeScript publish 尚未具备以上新协议恢复能力。幂等和恢复行为由后续 F04/P05 实现验收。

## 5 交付阶段及验收

M0 冻结本地工具及正式机器契约；M1 完成单平台真实上传和 A01—A14；M2 完成资料、完整制品集合和 A15—A18；M3 扩展多平台及 beta；M4 接入同引擎 CI。

必须从独立生成插件消费实际 CLI/Framework 包或可追踪快照，记录版本、摘要、源码状态、目标及操作 ID。真实上传、人工审核、公开下载和 PowerX 安装启用分别验收，不能相互替代。

## 6 相关文档

- [统一方案及十二项对齐结论](../../marketplace/plugin-developer-publishing-center-plan.md)
- [实现设计、文件范围及命令迁移](../../marketplace/plugin-publisher-implementation.md)
- [审核与上架](marketplace-review.md)
- [离线产物与隔离安装边界](offline.md)
- [本地安装](local-install.md)
- [能力契约与注册](capabilities/registration.md)
