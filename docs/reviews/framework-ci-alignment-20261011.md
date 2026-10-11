# Framework 已发布版本对齐与 Skeleton 身份链路、CI 验证

目标分支：develop。提交分支：fix/framework-ci-alignment-20261011。

develop 当前使用 Go v0.0.22 和旧版 Nuxt 鉴权接线。升级至已发布的 Go v0.0.24、Nuxt client 0.0.12、admin 0.0.11 后，补齐消费方身份与 Runtime 接口迁移，并让分支 push 即触发 CI。

## 代码变化

- Go Framework 使用已发布 v0.0.24 的源码：修复姓名解析成员缺租户、租户数字状态解码和 key 映射；增加凭据租户核验、正式请求身份及 WS 身份复核；同步已发布的 typed Runtime/Host 合同与测试。
- Nuxt client 统一 bridge v2、已验证宿主会话、结构化鉴权错误分类、Router 导航及 WS/SSE 会话绑定。admin Scheduler 从正式 client 包导入，并声明 client peer dependency。
- Skeleton 在 JWT 后、RBAC 前验证 Core 请求身份；启动时绑定正式 resolver，WS 在收发消息和订阅时复核身份及授权，退出时关闭生命周期。补齐 UUID 身份及访问凭据到期时间。客户鉴权消费方迁移已移除的构造器，改用模式绑定的正式 Runtime 接口。
- 同步 CLI 默认版本、Skeleton 依赖和 lockfile、两个模板副本。修复模板同步的 Vue 插值转义与 admin/client 版本参数替换；生成项目直接编译真实发布依赖，不使用仓库 workspace 或本地 Framework replace。
- CI 支持 fix/framework-* 的 push、PR 和手动触发；frontend 增加 7 个鉴权/传输合同测试文件，backend 增加生成项目编译检查。旧工具 token 禁用检查只扫描实现代码，允许文档描述禁止使用的名称。

## 本地验证

- Framework Go、Skeleton Go、CLI 全模块测试通过。
- 前端 7 个文件、122 个测试通过；完整 Skeleton Nuxt 生产构建通过。
- 生成后端在 GOWORK=off、无本地 Framework replace 条件下编译和测试通过。
- 模板同步、agent 路径检查、git diff --check 通过。
- 远端 CI 结果以当前分支 HEAD 对应的 GitHub Actions run 为准。

## 范围与发布顺序

不包含 .NET Scheduler、知识库 UI、make dist target 和暂停验收的内容导览适配；这些工作保留在原开发工作区。未修改 Core 或 AI Craft，未重启用户服务。

Go v0.0.24、client 0.0.12、admin 0.0.11 已在前一步发布。本次提交不重复发布或移动既有 tag。之后的发布应先提交分支并通过 CI，经 PR 合并后再发布新的版本。

用户自行向 develop 创建 PR；本次不创建或合并 PR。
