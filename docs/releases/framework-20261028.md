# PR #93 后续整合与发布准备

本次在 `025-framework-business-regression` 分支整合已合并的 PR #93，补齐新插件生成、权限声明及本地 Runtime 改动。本文件记录发布准备，不代表版本已发布或已有插件已升级。

## 版本与发布顺序

- Go SDK 目标版本：`v0.0.28`，待 develop 合并且 CI 通过后创建 `framework/backend/go/v0.0.28`。
- px-plugin 目标版本：`v1.0.5`，待 Go SDK 公共下载验证通过后创建 CLI tag。
- 前端 SDK 引用保持 Admin `0.0.13`、Client `0.0.14`；本次没有新的 npm 包发布。
- 新 SDK 校验和通过当前 Git 源码生成的模块归档计算；保留已发布 `v0.0.27` 的原校验和。发布前还需比较最终 Git 归档，发布后需从公共 Go 代理验证。

## 行为变化

- 新插件的 Skill provider、种子数据、能力调用和后台清单统一使用生成时指定的插件 ID。Skills 与后端、清单一样由 Skeleton 同步到两套模板，不再复制固定基础插件 ID。
- 回归模块统一使用 `regression.runs:read`、`regression.runs:invoke`、`regression.runs:manage`，API 声明绑定对应业务权限；页面为 `regression.page:read`，菜单为 `menu.console.regression:view`。
- TaskBus 传递显式 API Key 鉴权配置；缺少 Key 或鉴权失败时明确失败，不切换到 STS 或用户凭据。
- 日志输出遵循显式配置，与业务 Provider 模式分离；文件输出初始化失败即中止启动，文件模式的 Fatal 错误同时输出终端摘要。
- SDK 包含 tenant/owner 作用域的 Scheduler service 读取客户端；此新增客户端不改变现有 Skeleton Scheduler 默认调用路径。
- CI 与 Release 均校验实际 Skeleton 权限声明，打包回归验证安装包中的 AI provider catalog。

## 现有插件升级

更新模板或已生成插件时，需同步回归模块的清单、后端路由及前端权限判断，并重新提交权限声明。旧的 `regression:read/invoke/manage` 不作兼容；已有自定义角色需由管理员按业务需求授予新的业务、页面及菜单权限。本次代码变更不自动修改运行环境的角色或授权数据。

## 验证边界

本地验证覆盖 Go Framework、Skeleton、CLI、隔离的生成插件、权限声明、模板同步、模块校验和、打包、前端单测及浏览器受控业务回归。浏览器测试使用内存 SQLite 和注入测试身份，包含桌面、窄屏及浅深主题；测试进程和端口在结束后关闭。

真实 PowerX 安装、正式 STS/API Key 授权和公共仓库下载验收仍需单独记录，不能用受控测试或 CI 成功替代。原有 live 回归验收记录和 Core 补丁不属于本次提交。
