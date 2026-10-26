# Framework 与 px-plugin 开发版本 dev.2

本批修正开发快照的 Go 归档校验和及版本引用。dev.1 是本地快照；不将它直接复用为公开分发版本，也不覆盖原有快照或其模块缓存。

| 组件 | 版本 |
| --- | --- |
| Go Framework | `v0.0.27-dev.20261026.2` |
| Framework Client | `0.0.14-dev.20261026.2` |
| Framework Admin | `0.0.13-dev.20261026.2` |
| px-plugin | `v1.0.4-dev.20261026.2` |

## 归档修复

原打包器遍历磁盘，错误地收进被 Git 忽略的 `coverage.out`，同时排除了 Git 管理的 `.gitkeep`。这些文件都会影响 Go 模块的 `h1`，因此本地 ZIP 与正式 Git 归档的校验和不同。

本批按 Git 的已跟踪及未忽略文件选择工作区输入，保留尚未提交的源代码修改。Python 打包器与 Go 生成物测试使用相同规则；CLI 嵌入目录存在被忽略的缓存、日志或构建产物时明确失败。模板同步清理这些残留，原文件保留在本机验收目录。正常的源代码、启动脚本和旧快照保持原样。

使用 `--canonical-revision` 时，还会逐文件比较 Go SDK 与指定 Git 提交；不一致即停止打包。归档通过 Go 工具在独立模块缓存中读取并生成 `Sum`、`GoModSum`，消费者验收再次核对实际安装的校验和。

dev.2 的 Go SDK 校验和为：

```text
h1:Yq8XFoEAxn0f+8qezJdn6jmK3IyNPLdU/sY88DBxACU=
```

该值与 `abfc0e871049694a19fee98502f9ef6f237cc02e` 的规范 Go 归档一致；该提交的公共 Go proxy 归档也已按文件内容比对。SDK 业务源码未改变。

## 验证命令

```bash
python3 -m unittest discover -s scripts/release -p 'test_framework_archive.py'
python3 scripts/release/prepare-framework-dev.py \
  --cli-version v1.0.4-dev.20261026.2 \
  --canonical-revision abfc0e871049694a19fee98502f9ef6f237cc02e
python3 scripts/release/validate-framework-dev.py /absolute/path/to/bundle
npm run sync:templates -- --check
```

开发包记录升级为 `framework_dev_bundle.v2`，包含 `go_checksums` 与 `go_canonical_git_revision`。新校验器只接受该格式。原 dev.1 的验收记录保留其历史状态。

## 发布顺序

修复先在当前分支提交 PR；合并后等待最新 `develop` 的 CI 通过，再在该提交打 tag。npm 按维护者要求延后发布。tag 工作流构建带版本号的 CLI，并验证归档及独立生成后端；不再调用旧示例中的 Marketplace 发布命令。

本机可通过开发包中的 `activate.sh` 或 `init.sh` 使用 dev.2，CLI 默认依赖与同批包版本一致。该包尚未公开发布，不能将本机验收视为公共 registry 安装或安装态业务端到端验收。

验收结果见 [dev.2 记录](../contracts/evidence/framework-dev-20261026-2.json)。
