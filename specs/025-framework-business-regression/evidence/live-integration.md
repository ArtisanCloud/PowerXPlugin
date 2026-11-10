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

## 续作前置检查（2026-10-19）

用户要求继续完成剩余任务后，在当前 `025-framework-business-regression` 分支重新检查：

```bash
python3 scripts/testing/regression-live-check.py --output artifacts/regression/live-readiness-2026-10-19.json
```

退出码为 `2`，报告如下；本次未覆盖旧报告：

```json
{
  "schema_version": "v1",
  "status": "blocked",
  "reason_code": "REGRESSION_LIVE_ENV_REQUIRED",
  "missing_fields": [
    "REGRESSION_API_BASE",
    "REGRESSION_PLUGIN_ID",
    "REGRESSION_TENANT_UUID",
    "REGRESSION_LIVE_PROFILE_UUID",
    "REGRESSION_LIVE_RESOURCE_TYPE",
    "REGRESSION_LIVE_RESOURCE_UUID",
    "REGRESSION_LIVE_TAG_UUID",
    "REGRESSION_TOKEN_ENV"
  ]
}
```

另外只读解析了 `skeleton/backend/etc/config.yaml`：regression 已启用，环境为 `development`，仅配置一个 `controlled` profile，没有 `live` profile、专用租户或资源绑定。`skeleton/backend/.env` 也没有 `REGRESSION_*` 配置。上述结论来自磁盘配置与当前检查进程环境，不代表已读取运行中服务的有效配置。

缺少必要配置时，前置检查未发送任何 HTTP 请求。未执行标签绑定、授权／撤权、迁移或业务清理；未启动、停止或重启任何服务，也未修改运行配置。T045 保持未完成，总进度仍为 56/57。

继续执行需要环境负责人提供已有专用环境的配置文件路径，包含上列八项配置；测试 token 通过 `REGRESSION_TOKEN_ENV` 指定的环境变量提供，不写入聊天、文档或报告。就绪后先检查实际目录和固定绑定，再执行真实授权、幂等、业务读回及原标签集合恢复验收。
