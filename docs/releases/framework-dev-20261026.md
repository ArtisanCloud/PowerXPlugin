# Framework 与 px-plugin 开发版本（2026-10-26）

这套版本用于生成新插件并使用当前 Framework 能力。它是本机可追踪的开发快照，尚未对外发布；不等于 npm latest 或 GitHub 正式 Release。

本次先提交并推送 `025-framework-business-regression` 分支源码，由维护者创建 PR 合并到 `develop`。后续只有在最新 `develop` 的 CI 通过后，才基于该提交打版本 tag 并发布；当前分支不打发布 tag，也不向 npm 发布。

| 组件 | 版本 |
| --- | --- |
| Go Framework | `v0.0.27-dev.20261026.1` |
| Framework Client | `0.0.14-dev.20261026.1` |
| Framework Admin | `0.0.13-dev.20261026.1` |
| px-plugin | `v1.0.4-dev.20261026.1` |

Framework 包、Skeleton、scaffold、CLI 内嵌模板与生成器默认版本一起对齐。Admin 精确依赖同批 Client 开发版本，并声明 Node 类型 peer；生成项目显式安装 `@types/node`，不依赖其他包间接提供全局类型。CLI 支持 `--framework-admin-ref`，并接受 `PX_PLUGIN_FRAMEWORK_VERSION`、`PX_PLUGIN_FRAMEWORK_ADMIN_REF`、`PX_PLUGIN_FRAMEWORK_CLIENT_REF` 三项显式环境配置；命令行参数优先。`--install-deps` 安装失败会以失败退出，不能再把未完成安装报成初始化成功。

AI Settings 使用 Core 正式的 typed `core_internal` 目录合同；新项目同时获得该合同与 fixture。AI Media v2、通知卡片、图片 derivatives 等当前 Client 导出随实际 tarball 交付。

同步规则明确排除了后端测试源码；本轮清除了过去残留的 42 份测试模板及其 42 份 CLI 副本。同步检查会拒绝再次夹带这些过期文件。生产者的 Framework/Skeleton 测试继续保留并执行。

模板中的旧 `plugin`／`migrate` 二进制及其副本也已移出，原文件保存在验收目录的 `retired-template-binaries/`。同步、打包和生成检查会拒绝构建产物进入新插件；新项目从当前源码构建自己的程序。

## 生成新插件

推荐使用本机已有的 Node `22.23.2`。脚手架的第三方直接依赖固定为仓库 lockfile 当前版本，包括 Nuxt `4.4.2`、UI `3.3.7`、i18n `10.2.3`。这避免原 `^4.2.0` 范围解析到 Nuxt `4.6`／Router `5`，与 UI `3` 的 Router `4` 要求冲突。实际安装与工具链版本见验收记录。

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
source artifacts/framework-release-dev-20261026/activate.sh
px-plugin version

# 进入你准备存放新插件的工作目录，并替换插件 ID。
cd /path/to/plugins-workspace
px-plugin init com.powerx.plugins.example --install-deps
```

激活脚本只影响当前 shell：选择这批 CLI、设置 Go 本地包源及两个 npm tarball 引用，不写全局 Go/npm 配置。后续在新终端开发该插件时，也先激活这批依赖。若使用 nvm，可先运行 `nvm use 22.23.2`。

也可以直接用本批生成入口：

```bash
bash /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/artifacts/framework-release-dev-20261026/init.sh \
  com.powerx.plugins.example --install-deps
```

开发包位于 `artifacts/framework-release-dev-20261026/`，精确目录、源码哈希和包校验和见其中的 `bundle.json` 及 `references.json`。生成项目依赖这里的包文件，请保留该目录。跨机器使用前需要分发完整开发包并重新生成依赖引用，或另行正式发布这组版本；不能直接从公共源安装尚未发布的版本。

## 复现与验收

```bash
python3 scripts/release/prepare-framework-dev.py --cli-version v1.0.4-dev.20261026.1
python3 scripts/release/validate-framework-dev.py /absolute/path/to/bundle
npm run sync:templates -- --check
```

打包器不发布、不打 tag、不安装全局 CLI。校验器在仓库外生成插件，使用独立 Go 模块缓存、`GOWORK=off`、无 Framework 本地 `replace`，验证真实 npm tarball 安装、合同、全后端编译、公开 TypeScript 导出与 Nuxt 生产构建，结束后删除临时项目和缓存。

最终结果见 [验收记录](../contracts/evidence/framework-dev-20261026.json)。Core 的已交付合同来自 PowerX 维护方；本轮不修改 Core、授权数据或用户服务，也不把受控合同测试标成真实 STS／安装态页面端到端验收。
