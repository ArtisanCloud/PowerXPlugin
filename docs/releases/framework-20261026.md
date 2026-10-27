# Framework 与 px-plugin 正式版本

Go Framework `v0.0.27` 已正式发布，官方 Go proxy、校验数据库及独立消费验证通过。CLI、前端 SDK 与脚手架默认引用统一使用不带后缀的正式版本号。

| 组件 | 正式版本 | 当前状态 |
| --- | --- | --- |
| Go Framework | `v0.0.27` | [已正式发布](https://github.com/ArtisanCloud/PowerXPlugin/releases/tag/framework/backend/go/v0.0.27)，公共源安装通过 |
| Framework Client | `0.0.14` | 源码与包已准备，npm 发布按维护者要求延后 |
| Framework Admin | `0.0.13` | 源码与包已准备，npm 发布按维护者要求延后 |
| px-plugin | `v1.0.4` | 默认依赖已更新，待当前 PR 合并及 develop CI 成功后打正式 tag |

前端 SDK 的两个源码 tag 与 px-plugin 正式 tag 在版本引用修改合并后创建。Go SDK 发布源码为 `9fb5fd3da219ceb8a56d7cc06c9aa19f6c033a24`，该提交的 [develop CI](https://github.com/ArtisanCloud/PowerXPlugin/actions/runs/40322727448) 十项检查通过。

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

`PX_PLUGIN_CLI_VERSION` 只注入构建版本标记；构建前应确认当前源码对应所需 CLI tag。npm 发布完成前，不能将 frontend 的本地 tarball 验收视为公共 npm 安装已完成。

## 验证

正式 Go 模块从 `proxy.golang.org` 下载并由 `sum.golang.org` 校验，独立缓存、关闭 workspace、无 Framework 本地 replace 的安装、AI SDK 合同测试及 `go mod verify` 均通过。公开归档共 510 个文件，与规范 Git 输入一致。

5 项归档回归、CLI 21 个测试包、模板同步与发布路径检查通过。关闭版本环境变量时，CLI 实际生成的默认依赖为上述三个正式 SDK 版本。仓库外生成插件从公共 Go 源安装后，清单校验、后端测试与构建、已安装 AI SDK 合同测试、TypeScript 导出检查和 Nuxt 生产构建全部通过；前端使用配套正式号 tarball，无源码链接。

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
