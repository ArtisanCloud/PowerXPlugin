# 真实 Core 集成验收

状态：**待专用环境**。用户再次确认仍无专用环境，本次继续完善本地交付，不勾选 T045。

当前仓库运行配置没有 regression live profile、测试租户／资源绑定或可用测试凭据。只读检查命令：

```bash
python3 scripts/testing/regression-live-check.py --output artifacts/regression/live-readiness.json
```

实际结果为 `status=blocked`、`reason_code=REGRESSION_LIVE_ENV_REQUIRED`，退出 2。检查脚本缺配置时不会发送请求；配置完整后也只 GET 场景和 profile 目录，不产生业务操作或授予权限。目录可读仅表示 `ready_for_execution`，`core_grant_verified=false`、`business_execution_verified=false`，不能当作 T045 完成证据。

环境负责人需提供以下变量，凭据只保存在指定的环境变量内，不写入文档：

- `REGRESSION_API_BASE`：插件实际 API 基址，包含 API 前缀。
- `REGRESSION_PLUGIN_ID`、`REGRESSION_TENANT_UUID`：专用插件和测试租户。
- `REGRESSION_LIVE_PROFILE_UUID`：服务器注册的 live profile。
- `REGRESSION_LIVE_RESOURCE_TYPE`、`REGRESSION_LIVE_RESOURCE_UUID`：profile 在后端固定绑定的专用资源。
- `REGRESSION_LIVE_TAG_UUID`：既有测试标签。
- `REGRESSION_TOKEN_ENV`：管理端测试凭据的环境变量名。

真正执行时须核对正式 Core 服务授权、绑定／重复绑定／业务读回，以及原标签集合恢复；报告中的来源、租户和资源必须与该配置一致。工作台不自动建立 Core grants，不修改普通业务资源。执行与证据要求见 [live 指南](../../../docs/guides/features/025-framework-business-regression/usecase-live.md)。
