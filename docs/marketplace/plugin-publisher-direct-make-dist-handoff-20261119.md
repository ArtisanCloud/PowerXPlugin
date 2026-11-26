# Base 0.1.6 无 Docker 打包与市场准入交接

发布中心已接通 Apple 芯片 Mac 直接构建。选择版本并点击「开始构建」，Framework 执行 `make dist`，然后复核并冻结 Linux / amd64 ZIP。市场准入修复交付后可继续真实上传。

## Framework 实测

Base 的 Nuxt Layer 在 Framework 仓库里通过本地依赖链接。直接打包时，Nuxt 从 Layer 目录解析 `nuxt/kit`，无法找到插件前端已安装的依赖。`skeleton/make-files/build.mk` 已为 `frontend-build` 设置插件自己的 `node_modules` 搜索路径，在未预设 `NODE_PATH` 的隔离副本重跑成功。

```bash
make dist VERSION=0.1.6 PLATFORM=linux TARGET_ARCH=amd64
```

本次构建使用 Apple Silicon Mac 上的 Go `1.26.7` 和 Node `22.13.0`。Go 交叉编译为 `linux/amd64`，`CGO_ENABLED=0`。前端由本机 Node 构建。`make dist` 的内建检查及 Framework 的冻结、ZIP 复核通过。`go test ./tools/cli/internal/publisher` 通过。

| 项目 | 复验值 |
| --- | --- |
| 插件 / 版本 | `com.powerx.plugins.base` / `0.1.6` |
| 目标 / Node 构建环境 | `linux/amd64` / `darwin/arm64` |
| 文件数 / ZIP 字节数 | `544` / `60,836,752` |
| ZIP SHA256 | `133224c7770bfafbeba3467f112896c72ce3e15e9186c3bfca4056ccde82e855` |
| 本机复验 ZIP | `skeleton/.px-plugin/publisher/acceptance/base-0.1.6-darwin/artifact.zip`，Git 忽略 |

在独立、无外网的 QEMU Linux / amd64 验证机中，对同一 ZIP 核对 SHA256 后解压。后端以本地模式使用临时内存库启动，`/healthz` 返回 200。包内 Nuxt 由 Linux / x64 Node `22.13.0` 启动，宿主管理页和页面引用的 JavaScript 均返回 200。验证范围到进程启动及 HTTP 请求，PowerX 安装和真实市场上传待联合验收。

### 网页实测

独立的 Nuxt 开发服务自动连接 Framework 执行器。在 Docker 未运行的 Apple 芯片 Mac 上，从发布中心创建 Base `0.1.6` 计划并点击「开始构建」。页面以直接构建提交任务，生成安装包后显示「发布包就绪」，点击「复核发布包」通过。验证了桌面与窄屏的状态恢复，以及浅色／深色主题与系统设置相反时的可读性。发布中心单测 68 项和 Go race/vet 通过。

网页生成的 ZIP 为 `60,827,994` 字节，SHA256 为 `eb10042228cc4dea80f85933dbf15d37f1975eaa18d8ce235e815b9943512262`，本机路径为 `skeleton/.px-plugin/publisher/acceptance/base-0.1.6-darwin/web-artifact.zip`，Git 忽略。对该 ZIP 使用当前 Market P04 实际校验器，结果仍为 `package_target_mismatch`。

## Market 需要处理的规则

用上述原始 ZIP 调用 Market P04 的 `admission.Validate`，请求目标为 `linux/amd64`，返回 `package_target_mismatch`。包内 `plugin.yaml`、`powerx-publisher.json` 的目标及 Go ELF 均匹配。拒绝发生在 `validateRuntime` 对 `tools.node_os` 和 `tools.node_arch` 的判断，它目前只接受 `linux/x64`。该字段由 Framework 写入实际构建工具信息。原 Base `0.1.5` 对照 ZIP 仍通过。[Market 准入检查](../../../../PowerXPluginMarket/backend/internal/publisher/admission/runtime.go)

建议 Market 在 `make-dist-pure-go-js/v1` 下单独校验 Node 构建平台，新增经过验证的 `darwin/arm64`，继续用声明目标与 ZIP 内容验证实际运行平台。在临时 Go overlay 中只调整该工具平台条件，同一 ZIP 通过 P04 全量校验。Market 工作树未应用该实验修改。

Market 交付需覆盖以下回归。

- 相同 SHA256 的 Base `0.1.6` 通过 P04，且已有 `linux/x64` 构建包继续通过。
- 不支持的构建平台、清单目标错误、Go ELF 架构错误和夹带原生 Node 模块仍被拒绝。继续执行现有 ZIP 路径、体积、压缩比、清单与引用检查。
- 在隔离市场真实上传同一 ZIP，回传 Version UUID、Artifact UUID 和服务端确认的 SHA256。

Market 交付后，Framework 用已接通的网页流程完成隔离市场上传，核对 Version UUID、Artifact UUID 和同一 ZIP 的 SHA256。协议沿用 `1.0.0-rc.2`。
