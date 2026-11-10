# 插件构建、发布与安装指南

开发者发布中心的统一文档基线为 v0.4（2026-11-09），在 PowerXPlugin 和 PowerXPluginMarket 两仓库同步维护。新增机器协议、页面和发布凭据尚待实现；当前源码检查、正式包发布和运行验收分别记录。

| 任务 | 文档 | 当前范围 |
| --- | --- | --- |
| 理解职责、产品流程和机器协议方向 | [统一对接方案](../../marketplace/plugin-developer-publishing-center-plan.md) | 十二项对齐结论、双方任务及 21 项验收 |
| 领取实现任务、迁移旧命令 | [实现设计与交付范围](../../marketplace/plugin-publisher-implementation.md) | Go 共享引擎、本地执行器、文件范围及 M0—M4 依赖 |
| 创建本地计划、启动工具服务 | [本地工具 v1](../../contracts/plugin-publisher-local-v1.md) | F02 基础已实现，不执行构建或市场操作 |
| 隔离构建与冻结 ZIP | [F03 构建契约](../../contracts/plugin-publisher-build-v1.md) | 首个配置受控验证，真实 Linux/Nuxt 运行待验 |
| 构建并上传市场 | [在线发布](online.md) | 当前 Makefile 与网页流程；新 publisher 明确标记待实现 |
| 提交审核和公开上架 | [审核指南](marketplace-review.md) | 草稿、送审、批准、上架分开；完整集合快照为 M2 任务 |
| 生成及传送离线产物 | [离线产物](offline.md) | 本地构建和导出，不承诺专用离线审核或字节断点续传 |
| 将产物安装到 PowerX | [本地安装](local-install.md)、[安装前检查](preflight-checklist.md) | 安装和启用单独验收 |
| 声明及注册能力 | [能力注册](capabilities/registration.md) | 能力治理不替代市场发布授权 |
| 使用 CLI 开发 | [CLI 指南](../develop/cli-plugin/README.md)、[CLI README](../../../tools/cli/README.md) | Go CLI 当前命令及生成工程流程 |

旧 TypeScript 发布回执、Plan/Deploy 内存记录及内部接口样例不作为真实市场上传、审核、签名或客户部署的证据。当前本地构建优先使用 `make dist`/`make package`；新发布工具采用正式机器契约，不自动回退旧接口。
