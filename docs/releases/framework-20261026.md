# Framework 与 px-plugin 正式版本

Go Framework `v0.0.27`、Framework Client `0.0.14`、Framework Admin `0.0.13` 与 px-plugin `v1.0.4` 的正式版本均已就绪。官方 Go proxy、npm registry 与独立生成插件的安装、测试和构建验证通过，脚手架默认引用统一使用正式版本号。

| 组件 | 正式版本 | 当前状态 |
| --- | --- | --- |
| Go Framework | `v0.0.27` | [已正式发布](https://github.com/ArtisanCloud/PowerXPlugin/releases/tag/framework/backend/go/v0.0.27)，公共源安装通过 |
| Framework Client | `0.0.14` | [npm 已发布，latest](https://www.npmjs.com/package/@artisan-cloud/plugin-framework-client/v/0.0.14) |
| Framework Admin | `0.0.13` | [npm 已发布，latest](https://www.npmjs.com/package/@artisan-cloud/plugin-framework-admin/v/0.0.13) |
| px-plugin | `v1.0.4` | [正式 tag 已推送](https://github.com/ArtisanCloud/PowerXPlugin/tree/v1.0.4)，Release 构建通过，本机 CLI 已升级 |

前端 SDK 两个源码 tag 与 px-plugin 正式 tag 均指向 `4cbeaf6d823bf5196bb06bec4d89bc6f7b38dbd8`；该提交的 [develop CI](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/40339672217) 十项检查及 [CLI Release 构建](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/40342514415) 全部通过。Go SDK 发布源码为 `9fb5fd3da219ceb8a56d7cc06c9aa19f6c033a24`，Go SDK 内容在本批引用调整中保持一致。

## Go 安装

在插件后端目录执行：

```bash
go get github.com/ArtisanCloud/PowerXPlugin/framework/backend/go@v0.0.27
go mod tidy
go test ./...
```

Go 模块校验和为 `h1:vQ5pzmcBX3OnNjh3IM0hLnAfDJQUhJ7WrWqehIHS31U=`；`go.mod` 校验和为 `h1:Xmb97ZEMwaN9kaz32/7GHXR/oQESdc3ILbiCtOHh+2w=`。

## CLI 与脚手架

`px-plugin init` 默认生成 Go Framework `v0.0.27`、Client `0.0.14`、Admin `0.0.13` 的依赖。Admin 的 Client peer 固定为同批 `0.0.14`。Skeleton、CLI 模板生成测试及锁文件同步更新。

`PX_PLUGIN_CLI_VERSION` 只注入构建版本标记；构建前应确认当前源码对应所需 CLI tag。本机全局命令与仓库 `bin/px-plugin` 均已升级到正式 `v1.0.4`，使用正式 tag 的干净 Git 归档编译，Go Framework 来自公共模块。

在新项目的父目录执行：

```bash
px-plugin --version
px-plugin init com.example.new-plugin --backend go-gin --admin nuxt --install-deps
```

已有插件应在自己的仓库显式升级 Go 与 npm 依赖，并运行该项目自身的测试。CLI 升级不会自动覆盖已生成的业务代码。若此前在当前终端激活过开发快照，建议新开终端，避免临时版本环境变量继续覆盖默认值。

## 验证

正式 Go 模块从 `proxy.golang.org` 下载并由 `sum.golang.org` 校验，独立缓存、关闭 workspace、无 Framework 本地 replace 的安装、AI SDK 合同测试及 `go mod verify` 均通过。公开归档共 510 个文件，与规范 Git 输入一致。

5 项归档回归、CLI 21 个测试包、模板同步与发布路径检查通过。关闭版本环境变量时，CLI 实际生成的默认依赖为上述三个正式 SDK 版本。仓库外生成插件从公共 Go 源安装后，清单校验、后端测试与构建、已安装 AI SDK 合同测试、TypeScript 导出检查和 Nuxt 生产构建全部通过；前端使用配套正式号 tarball，无源码链接。

发布后再次使用已安装的本机正式 CLI，在仓库外生成完整插件。Go 与 npm 均使用公共 registry、独立缓存，npm 安装不使用本机保存的凭据。实际安装的 Client `0.0.14`、Admin `0.0.13` 与公开 tarball 的完整性一致，类型检查、后端测试和构建、Nuxt 生产构建全部通过。临时消费目录与缓存已清理。

公共 npm 包包含的 18／19 个文件与对应正式源码逐文件一致，registry SHA-512、tarball SHA-256 与发布前验收包相同；两个 `latest` 分别为 Client `0.0.14`、Admin `0.0.13`。

正式号归档与配套包可以通过以下命令重建、验证；公共 Go 模式使用官方代理和校验数据库，不使用本地 Go proxy：

```bash
python3 -m unittest discover -s scripts/release -p 'test_framework_archive.py'
python3 scripts/release/prepare-framework-dev.py --stable \
  --cli-version v1.0.4 \
  --canonical-revision 9fb5fd3da219ceb8a56d7cc06c9aa19f6c033a24 \
  --output artifacts/framework-release-20261026/bundles
python3 scripts/release/validate-framework-dev.py /absolute/path/to/bundle --public-go
```

旧版本与历史验收记录保留。此次发布不操作 Core、数据库或日常服务。

具体校验和、包摘要及消费结果见 [正式版本验证记录](../contracts/evidence/framework-publication-20261026.json)。
