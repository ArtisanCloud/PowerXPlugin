# Framework 2026-10-19 发布记录

Go Framework v0.0.26、framework-client 0.0.13、framework-admin 0.0.12 均已正式发布，公共 Go proxy／npm registry 的正式依赖消费验证通过。

| 交付 | 版本 | 状态 |
|---|---|---|
| Go Framework | v0.0.26 | [已发布](https://github.com/ArtisanCloud/PowerXPlugin/releases/tag/framework/backend/go/v0.0.26) |
| Nuxt framework-client | 0.0.13 | [已发布，latest](https://www.npmjs.com/package/@artisan-cloud/plugin-framework-client/v/0.0.13) |
| Nuxt framework-admin | 0.0.12 | [已发布，latest](https://www.npmjs.com/package/@artisan-cloud/plugin-framework-admin/v/0.0.12) |

发布源码提交为 `d1fc3b25a881fb9e0bc2477f6f2cf3673456133b`，Go tag 为 `framework/backend/go/v0.0.26`。[GitHub CI](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/39148704898) 的 9 项检查全部通过，覆盖 Go、Skeleton、生成后端、前端、CLI、合同、smoke 与 regression。

## 本次内容

Go 新增 IAM 连续授权基线批量合同、typed Core client、严格响应校验与 Registry／Bundle 入口；WS 在最终发送前复核身份与 topic；Media 响应映射可选图片 derivatives 状态及变体信息；通知增加业务导航、独立已读回执、统一 Core API 前缀与错误分类。此外新增业务回归运行时、场景／断言／证据合同及可选 GORM 存储。

client 新增业务回归 API、事件校验及公共类型；admin 新增工作台、运行详情及 composable。admin 的 client peer 为 `^0.0.13`，valibot peer 为 `^1.2.0`，Nuxt peer 为 `^4.2.0`。

发布提交同步 CLI 默认版本、Skeleton 依赖与相关前端模板。本次只发布 Go／TypeScript SDK；其余工作区中的 .NET、Skeleton/scaffold 业务实现、业务工具与 AGENTS 修改未并入发布提交。SDK 升级不会覆盖插件已生成的内部实现，插件仍需显式接入启动装配及所选 provider。

## 验证结果

- 远端 CI：Go 1.24.13 linux/amd64、Node 22.23.2，9 项检查全部通过。
- 公共 `proxy.golang.org` 下载 Go v0.0.26，`sum.golang.org` 校验通过；Origin 对应发布提交。
- 发布 Go 模块的 480 个源码文件与发布提交逐文件一致。模块校验和为 `h1:KfX72k7QMeo3Ut9UgeKrUNlBr1T8Sto6hzyG8yPFwKg=`。
- 关闭 workspace、无 Framework 本地 replace、关闭源码快照，生成后端使用正式 v0.0.26 完成测试。
- 发布前本地 Go 全量／专项 race、Skeleton／CLI、client 4 项与前端 197 项测试通过；实际前端 tarball 的仓库外安装、公共类型导出及 Nuxt 4.2.0 构建通过。
- 公共 npm 包的 15／19 个文件与发布提交逐文件一致，完整性校验与此前通过构建的包一致；latest 分别指向 client 0.0.13、admin 0.0.12，alpha 保持 0.0.9。
- 仓库外工程从正式 npm registry 安装确切版本，确认无源码链接、公共类型完整；Node 22.23.2／临时 npm 12.2.0／Nuxt 4.2.0 的生产构建通过。临时消费工程、工具链与独立缓存已清理，保留锁文件与日志。
- lint 命令当前仍为占位脚本，不能据此声称完成静态检查。

[正式发布证据](../contracts/evidence/framework-publication-20261019.json) 记录 tag、发布提交、CI、公共源校验和 npm 结果。[发布前验证记录](../contracts/evidence/framework-release-preparation-20261019.json) 保留初始测试和失败尝试。本地包与日志保留在被忽略的 `artifacts/framework-release-20261019/`，不作为正式版本的分发入口。

首轮 CI 因漏带 Skeleton 前端依赖声明，仍将旧 admin 0.0.11 与新版 client 组合，安装失败。补齐依赖、锁文件、模板规则，并对齐已验证的 Node 版本后，当前提交的完整 CI 通过。npm 本机旧凭据查询返回 401，经网页重新登录为 matrix-x；发布操作另由 registry 要求二次认证，首个发布认证失效后重新完成验证；两个 npm 发布命令均成功。版本处理期间首次正式安装遇到尚未传播的 metadata，随后 npm 10.9.8 又触发依赖树内部 edgesOut 错误；使用临时 npm 12.2.0 与独立缓存完成安装和构建，未使用 force 或 legacy-peer-deps。npm 12 默认脚本审批策略保持原样，未统一放行依赖安装脚本。

## 升级

Go 消费方在后端目录执行：

```bash
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.26
go mod tidy
go test ./...
```

正式依赖验收必须关闭 workspace 并移除 Framework 本地 replace。前端在项目目录安装这两个正式版本：

```bash
npm install @artisan-cloud/plugin-framework-client@0.0.13 @artisan-cloud/plugin-framework-admin@0.0.12
```

两个 npm 源码 tag 分别为 `framework/frontend/nuxt/framework-client/v0.0.13` 与 `framework/frontend/nuxt/framework-admin/v0.0.12`，均指向相同的发布提交。

## 后置验收与发布边界

真实 Core service grant、IAM 初始化、B01–B24 安装态验收、插件本地授权实现、delegated worker、完整预览接管及业务阅读游标仍需各方分别接入验收。

本次不发布 .NET 或 CLI，不操作用户服务、真实数据库或 Core grant。现有 CLI 模板目录还包含旧的 `backend/go-gin/bin/plugin`、`bin/migrate` 构建产物；CLI 单独发布前需要清理并检查生成目录。本次测试从源码编译，没有运行这两份旧二进制。

正式发布证据在 SDK tag 推送后补录，后补文档不包含在 SDK tag 的源码提交中。
