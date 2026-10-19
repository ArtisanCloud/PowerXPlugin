# 插件接入与 CLI／页面一致性

2026-10-13，本地生成 `com.powerx.regression.alpha`、`com.powerx.regression.beta` 两个独立插件。在生成目录使用 `GOWORK=off` 和显式 local Framework replacement，编译插件入口与回归 API，并把同一正式 Service integration 验收放入生成后端执行，均通过。

生成物检查覆盖回归注册、模型／迁移、SSE、配置、薄页面、插件 ID 路由、locale 和本地化 API 模块。`sync:templates --check` 对 scaffold 与 embedded CLI 模板通过。两个生成项目的 Nuxt 均独立构建并连接各自生成后端完成 Chromium 验收：连续三次执行、业务失败定位、协作取消、刷新历史与 CLI 同 API 报告。每个 consumer 各六批次、active_templates=0，事件 source_plugin 对齐各自插件 ID。

页面和 CLI 对同一本地实际 API 执行模板场景，均保存独立批次，由同一 Export 生成 v1 报告。四个必要判定维度（定义／断言、缺失 key、verdict、cleanup_state）一致，均 passed／succeeded；对象 UUID 与时间是各批次独立值，不能按字节相等比较。CLI `--wait` 仅消费 SSE，最终读取一次完整快照，没有轮询。

```bash
(cd tools/cli && REGRESSION_GENERATED_TEST=1 go test ./internal/templates -run RegressionGenerated -count=1)
python3 scripts/testing/regression-browser.py
```

证据在 `artifacts/regression/local-validation.log`、`cli-business-report.json`、`browser-business-evidence.json`。该验证使用当前未发布的本地 Framework；外部插件直接消费正式模块、两个生成插件的安装态宿主鉴权，仍须在发布或安装环境就绪后另验。不修改其他插件仓库。

## 可重复的生成页面验收

```bash
python3 scripts/testing/regression-generated.py
```

最新验证通过：脚本自动生成两个临时工程，分别安装自己的依赖、独立构建 Nuxt，连接各自生成的 Go 后端，完成页面／CLI 六批次验收。Framework 前端使用本地 npm pack 的发布 allowlist，后端使用显式本地 replacement；没有发布 registry 包。

为复现本机 npm 11.3 的 peer resolver 错误修复，脚本下载并解包隔离 npm 11.6.2，通过其真实 CLI 路径执行安装，不修改全局工具，不使用 `--legacy-peer-deps` 或共享 node_modules。Pinia 3.0.4 已在 Skeleton／生成模板显式声明。npm 上游相关问题：[npm/cli #9787](https://github.com/npm/cli/issues/9787)。本次实际安装、构建与业务验收均通过；依赖安装仍有本机 Node 24.2 的 engine 警告，CI 使用 setup-node 的 Node 24。

对应日志：`generated-local-validation.log`、`generated-validation.log`、`consumer-alpha-install.log`、`consumer-beta-install.log`、各自 build／browser 日志。报告位于 `artifacts/regression/consumer-alpha/`、`consumer-beta/`。两个工程完成后自动删除；两组后端／Nuxt 正常退出，端口均释放。
