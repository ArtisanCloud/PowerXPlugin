# Framework 2026-10-19 发布记录

状态：正式发布执行中。Go v0.0.26、client 0.0.13、admin 0.0.12 的源码与本地消费验证已完成，正在提交发布源码并验证 CI；远端 tag／npm 结果另行记录。

| 交付 | 候选版本 | 正式发布入口 |
|---|---|---|
| Go Framework | v0.0.26 | `framework/backend/go/v0.0.26` Git tag |
| Nuxt framework-client | 0.0.13 | npm `@artisan-cloud/plugin-framework-client` |
| Nuxt framework-admin | 0.0.12 | npm `@artisan-cloud/plugin-framework-admin` |

## 本次内容

Go 包包含 IAM 连续授权基线、Media 派生图响应映射、通知导航／回执与 API 前缀对齐、WS 最终发送复核，以及业务回归运行时。各接口的 local 权威实现仍由插件负责，缺失依赖明确失败，不增加本地旁路。

client 增加业务回归 API、事件校验与公共类型导出；admin 增加对应工作台、运行详情和 composable。admin 依赖 client `^0.0.13` 与 valibot `^1.2.0`。现有 Nuxt peer 要求保持 `^4.2.0`。

CLI 默认值与 Skeleton Go 依赖更新为发布版本。本次提交发布 Go／TypeScript SDK、版本引用与合同文档；工作区其余 Skeleton／scaffold 功能修改、.NET 及业务工具不并入该发布提交，也不发布 CLI。升级 SDK 不会自动改写已有插件的内部源码。

验证时发现现有模板目录还包含 `backend/go-gin/bin/plugin`、`bin/migrate` 两份旧二进制，生成器会复制它们。消费测试从生成源码编译，未运行这两个文件；本轮不发布 CLI。CLI 单独发布前需要清理模板构建产物并验证生成目录。

## 候选包与验证

候选包、校验和、源码清单、消费工程和验证日志保存在仓库 `artifacts/framework-release-20261019/`（已忽略，不提交二进制）。[验证记录](../contracts/evidence/framework-release-preparation-20261019.json) 保存版本、SHA256、检查结果、失败尝试与正式发布待办。

| 包文件 | SHA256 |
|---|---|
| `powerxplugin-framework-go-v0.0.26.zip` | `8f12664af5fc6b17958cfd2b05643789698f2a67abdfff0a6739d95e38406d08` |
| `artisan-cloud-plugin-framework-client-0.0.13.tgz` | `418c9c7d5fa65eb3684efe964a0b469b4e6fee1b47db9c35e7b03b50786b967f` |
| `artisan-cloud-plugin-framework-admin-0.0.12.tgz` | `a443fbb2ba8c057a1e58add08820e56a95ec469c7e4c436f71b8495df079b55c` |

Go 包的 480 个源码文件、client 的 15 个文件和 admin 的 19 个文件均与当前工作区逐项核对。打包时工作区已有未提交功能修改，候选包代表当时源码快照。正式发布以提交和 tag 为准；后续发布文字调整不代表实现变化。

| 检查 | 结果 |
|---|---|
| Go Framework 全量、关键合同与 runtime race 测试 | 通过 |
| Skeleton 后端、CLI 全量测试 | 通过 |
| 新 CLI 生成默认版本与模板一致性 | 通过 |
| 生成插件消费 Go 本地 module proxy，bootstrap／WS race 测试 | 通过，无本地 Framework replace |
| client regression 测试 | 4 项通过 |
| Skeleton 前端单元测试 | 21 个文件、197 项通过 |
| 仓库外工程安装实际前端 tarball，公共类型导出 | 通过，无源码链接 |
| 仓库外工程 `npm ci` 与 Nuxt 4.2.0 生产构建 | 通过，Node 22.23.2／npm 10.9.8／TypeScript 5.9.3 |
| 根合同检查、CI guardrails、模板同步、diff 格式 | 通过 |
| `npm run lint` | 命令成功，但当前为占位脚本，不能视为静态检查通过 |

Go 测试工具链为 go1.26.7 darwin/arm64；本轮未另测 Go 1.24 最低版本。首次 npm 安装遇到内部依赖树错误，之后的 npm 11.6.2 安装生成了缺少 commander 的锁文件；最终以 Node 22.23.2 自带 npm 10.9.8 修复消费工程锁文件并完成干净安装与构建，未使用忽略 peer 依赖参数。失败日志全部保留。最终安装仍有 vue-i18n、unplugin-vue-router、glob 弃用提示，构建仍有动态导入与 chunk 体积警告，未在打包任务中扩展为依赖升级。

Go 本地 module proxy 包只用于验证当前源码快照，使用隔离 module cache；不代表版本已出现在公共 Go proxy。npm tarball 在隔离消费工程安装和构建；不使用源码 workspace 或源码目录链接冒充发布包验证。

正式发布前，需要确认发布提交包含所列源码和版本文件，并以通过 CI 的准确提交创建 Go tag。npm 按 client → admin 顺序发布。发布后另外执行公共 Go proxy／npm registry 消费验证。

## 后置验收

真实 Core service grant、Core 前端已读同步、IAM 初始化、插件完整预览接管、业务 worker 以及安装环境联合验收仍按各合同分别完成。本次不发布 .NET 包，不操作用户服务、真实数据库或服务授权。
