# 插件打包与市场发布契约

核对日期：2026-11-09。统一发布方案和实现设计是新开发者工具的文档基线；本页描述当前构建事实及迁移目标，不声明签名、扫描或市场发布已经完成。

## 1 当前安装布局

生成插件的 `make dist` 依据有效 Makefile 变量形成 `dist/<version>/`。标准 Go/Nuxt 工程的核心布局为：

```text
dist/<version>/
  plugin.yaml
  backend/bin/plugin
  backend/bin/migrate          # 声明迁移且有对应程序时
  web-admin/.output/
  web-admin/i18n/              # 项目存在时
  plugin.d/
  config/event_fabric.yaml     # 项目存在时
  skills/                     # 项目存在时
```

其他受控资源按 manifest 引用及安装契约校验；路径不能只靠本示意推断。当前 dist 不自动承诺复制所有 contracts、生成 checksums.txt/manifest.json、运行安全扫描或完成签名。需要引用的 schema/资源缺失时应在后续 F03 校验中明确失败并修正包内容，不补造成功报告。

## 2 当前构建与 ZIP

在依赖及配置就绪的生成插件根目录，Linux 执行器示例：

```bash
make dist VERSION=1.2.0 PLATFORM=linux TARGET_ARCH=amd64
make package VERSION=1.2.0 PLATFORM=linux TARGET_ARCH=amd64
```

第二条会再次执行 dist，并输出 `<plugin_id>-1.2.0.zip`，内含单一版本目录。Framework 仓库根入口会转发到 skeleton。版本和 target 写入产物 manifest，源码 manifest 不自动修改；Go 目标检查不等于完整 Nitro 原生依赖校验。

`make release/package-release` 是另一套 target 输出规则，`make package-pxp` 是历史 staging 入口；二者不作为 M1 统一发布引擎。Go `px-plugin package` 另行生成 TAR payload，`px-plugin dist` 当前只有 experimental 提示，不能与 Makefile 入口混同。

## 3 新引擎首个配置：已实现并受控验证

Go CLI 及本地 HTTP 已复用隔离构建引擎；页面尚未接入。首个配置是原生 Linux/amd64、无 CGO 的 Go 后端及纯 JS 前端，以 dist 为基础生成 ZIP，真实 Linux/Nuxt 运行待验。只对已冻结 staging 压缩一次，不再次调用会构建的 `make package`；重试复用同一文件。

新包必须有唯一正式 `id`、SemVer 版本和显式 `target.os/arch`，与计划、实际二进制及完整前端运行时一致。安装根和入口路径沿用宿主契约；链接、重复条目、危险路径、解压预算和执行权限共同验证。

最终压缩文件按实际字节计算 SHA256 和大小，市场独立复算。排除 `.env`、凭据、业务数据、上传文件、仓库元数据和 `.px-plugin/publisher/`。hash、源码提交和 signature.json 占位均不证明有效签名。

## 4 发布、审核与安装

当前可在市场开发者网页上传实际文件，成功登记私有草稿；资料齐备后提交审核，人工批准后显式公开上架。新 `/publisher` 机器接口、独立发布凭据及幂等恢复待实现，不使用内部 API 或本地文件 URI 代替传输。

首期恢复任务和操作结果，必要时同 operation UUID 整包重传，不支持字节级断点续传。同版本同平台不可覆盖；修改已有包需新版本。

真实上传、送审、批准、上架、下载及 PowerX 安装启用分别验收。客户端不能通过 `verified` 字段、假扫描结果或旧回执决定市场审核结果。

## 5 统一文档

- [开发者发布中心与市场对接方案](../../../../marketplace/plugin-developer-publishing-center-plan.md)
- [实现设计、迁移表及文件范围](../../../../marketplace/plugin-publisher-implementation.md)
- [在线发布与当前入口](../../../../guides/publish/online.md)
- [本地安装](../../../../guides/publish/local-install.md)
