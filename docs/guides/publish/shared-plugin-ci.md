# 插件公共构建工作流

业务仓库在自己的 Actions 运行中调用 `.github/workflows/reusable-plugin-ci.yml`。工作流在 GitHub 托管的 Ubuntu 24.04 x64 Runner 上，为 `linux/amd64` 生成经过 Publisher 校验的插件包。产物用于交给 XDocker 核对和隔离安装验收。

## 调用约定

| 输入 | 约定 |
| --- | --- |
| `source_sha` | 40 位小写 Git SHA，必须与调用方本次运行对应的提交及实际检出一致 |
| `plugin_version` | Publisher 支持的稳定 SemVer，最长 50 字符，可带构建元数据 |
| `mode` | 当前支持 `build`，省略时默认为 `build` |
| `request_id` | 1 至 128 个 ASCII 字符，以字母或数字开头，允许字母、数字、点、下划线和连字符。同一运行内，每次调用使用独立请求 ID |

插件根目录须提供 `plugin.yaml`、Publisher 兼容的 `make dist`、`backend/go.mod`、`backend/go.sum` 及 `web-admin/package-lock.json`。工作流先准备锁定依赖，Publisher 从 Git 干净源码构建并冻结安装包。后端需要支持 `CGO_ENABLED=0`、`-trimpath` 和计划版本注入，详见 [Publisher 构建契约](../../contracts/plugin-publisher-build-v1.md)。

使用 Framework 模板的插件应在提交构建源码前运行 `make plugin-yaml-sync`，并提交生成的 `plugin.d/` 清单变化。已同步的清单在构建时再次生成后应保持相同内容，否则 Publisher 会以 `PUBLISHER_BUILD_SOURCE_CHANGED` 拒绝冻结包。

工具链固定为 Go `1.24.13`、Node.js `22.23.2`。Runner 安装固定的 ripgrep `14.1.1` 供 `make dist` 校验使用，并独立检出 CLI 源码，用 `go build -mod=readonly -trimpath -buildvcs=false` 构建 `px-plugin`，注入版本 `ci.10c98e4eada350d296d106ff5ad805192380eaa8`。执行前用下列 SHA256 校验 Linux/amd64 CLI 二进制。 [CLI 源码提交](https://github.com/ArtisanCloud/PowerXPlugin/tree/10c98e4eada350d296d106ff5ad805192380eaa8) · [ripgrep 发行](https://github.com/BurntSushi/ripgrep/releases/tag/14.1.1)

```text
d1c8c046ecf91913617eaeacbfcdacd5e9997aae192509287e56eccbfa0d9882
```

## 电商调用示例

在电商仓库现有 `.github/workflows/ci.yml` 的 `on` 下加入手动入口，保留已有的 `push` 和 `pull_request` 配置。GitHub 从默认分支上登记该入口。触发时选择目标分支，并填写该分支本次运行应对应的完整提交 SHA。

```yaml
on:
  workflow_dispatch:
    inputs:
      source_sha:
        description: Expected full Git SHA of the selected branch
        required: true
        type: string
      plugin_version:
        description: Plugin package version
        required: true
        type: string
      request_id:
        description: Request identifier (optional for a manual run)
        required: false
        type: string
```

在同一文件的 `jobs` 下加上以下调用。原 `build` 作业继续运行 `make ci-all`。将占位符替换成 Framework 审核后的**完整工作流提交 SHA**，再提交调用方。

```yaml
  package:
    if: github.event_name == 'workflow_dispatch'
    needs: build
    permissions:
      contents: read
    uses: ArtisanCloud/PowerXPlugin/.github/workflows/reusable-plugin-ci.yml@<FRAMEWORK_WORKFLOW_COMMIT_SHA>
    with:
      source_sha: ${{ inputs.source_sha }}
      plugin_version: ${{ inputs.plugin_version }}
      mode: build
      request_id: ${{ inputs.request_id || format('github-{0}', github.run_id) }}
```

## 交给 XDocker

成功后，从电商运行的 **Artifacts** 下载 `plugin-build-<request_id>-attempt-<run_attempt>`。它包含 Publisher 生成的原始 `artifact.zip` 和 `build.json`。后者是 `build run` 响应中的 `.data`，保留原包大小和 SHA256。运行摘要记录插件源码、工作流实际提交、CLI 版本和摘要。

把运行与 attempt 链接、插件源码 SHA、Framework 工作流 SHA 和两份文件交给 XDocker。使用 `verify-plugin-build` 时，将文件放在私有目录，并将 `build.json` 权限设为 `0600`。首单验收应覆盖业务 CI 通过和两份文件可下载。拒绝路径用另一个有效 SHA 作为预期提交，确认本次运行失败且没有插件包制品。
