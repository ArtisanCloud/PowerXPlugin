# Quickstart: 业务场景回归工作台

**Status**: 本地实现与分层验收已落地；真实 Core 授权验收待专用环境（用户确认本次先完成本地）。实际结果见 [acceptance.md](evidence/acceptance.md)。
**Branch**: `025-framework-business-regression`
**Requirements**: [spec.md](./spec.md)
**Implementation**: [tasks.md](./tasks.md)

## 1. 前置条件与配置

- 在隔离开发或专用测试实例部署 Go/Nuxt 插件，准备测试租户与具备 regression:read/invoke/manage 权限的管理端身份。
- 后端沿用 backend/etc/config.yaml、CONFIG_PATH、server.api_prefix、database.schema；不得为工作台在根目录另建运行配置。
- 已实现 regression.enabled（默认 false）、regression.environment（development/test）、regression.profiles。Profile 声明稳定 UUID、key、title_key、kind、fixture_namespace；依赖 factory 在插件代码注册，不从浏览器接受任意 endpoint。
- 启用时缺少 Store、必需表、当前身份或 profile factory 必须明确失败；不得自动切到内存 Store、真实业务租户或另一种 profile。
- 通过现有数据库工具及 POWERX_RUN_MIGRATE 显式执行迁移；验证三类表、UUID 主键、索引及 PostgreSQL RLS。未授权运行迁移时不得由工作台自动补表。
- 前端菜单通过 `NUXT_PUBLIC_REGRESSION_ENABLED=true` 开启；后端执行开关独立校验。
- live profile 还须配置 `tenant_uuid`、`metadata_resource_type`、`metadata_resource_uuid`，绑定已有专用资源；场景会恢复原标签集合。
- live profile 的 Core/外部测试服务、STS 和 capability grants 由操作者准备，工作台不自动授予权限。
- 使用已注册场景与 profile 的稳定 UUID；页面选择显示名称，CLI 参数使用 UUID。UUID 必须真实，不能填造一个不存在的资源。

## 2. 插件开发者接入步骤

1. 使用 Framework Registry 注册套件及场景；Definition 声明本地化 key、输入 schema、非空步骤、非空且唯一的 required_assertion_keys、验证边界和执行/清理时限。
2. 实现 Preflight、Prepare、Execute、Cleanup，通过 Recorder 记录业务操作与结构化断言，通过 FixtureRegistry 登记创建资源。
3. controlled 使用正式业务 Service/HTTP/存储，仅替换声明的外部边界；live 注入现有 Core/外部服务客户端，不改变应用全局配置。
4. 按标准模型映射注入 Store、租户事务、迁移、RBAC 和配置；将公共工作台组件放入插件薄页面，注册 locale 与菜单。
5. 在原生 feature test 调用同一场景验证业务行为；不能把正式鉴权入口换成伪造身份后仍宣称验证了入口链路。
6. 将目录、场景与依赖说明、验证步骤和异常恢复动作加入功能指南。

结果展示通过 `describeCase(definition).results` 显式声明字段、枚举与 locale，Framework 不包含插件业务字段。登记资源时保存名称、角色和同场景来源 UUID，清理后通过 `Cleaned(ctx, resourceUUID, evidence)` 保存正式读回证据。现有 Tailwind 4 插件在主 CSS 中导入 `@artisan-cloud/plugin-framework-admin/layer/assets/css/sources.css`；新模板已包含。详见 [SDK 展示契约](contracts/scenario-sdk.md#工作台业务展示契约)。

## 3. 页面验收：目录与受控回归

入口、按钮、字段和失败处理见[回归用例管理页面指南](../../docs/guides/features/025-framework-business-regression/usecase-workbench.md)。

1. 登录测试身份，打开插件管理端“业务回归”；确认菜单、按钮分别受查看/执行/管理权限控制。
2. 默认进入“回归用例”，核对当前环境和可运行数量（controlled 为 8 条 Template 用例，完整目录见[受控指南](../../docs/guides/features/025-framework-business-regression/usecase-controlled.md#7-操作步骤)）。在“模板创建与修改”行点击“查看详情”，了解创建、修改、重新读取三个步骤、通过条件和数据影响；版本、时限及依赖在“技术详情”查看。数据准备和清理由场景生命周期负责，不伪造为已登记的业务步骤。
3. 点击“运行”，核对预填业务表单后“开始执行”。接受后默认进入当次结果概览，从目标用例“查看业务校验”核对预期与实际；切到“测试数据与清理”检查名称、来源和逐项清理证据，“执行与诊断”查看步骤与技术快照。桌面纵向分区、窄屏可滚动导航须真实验收，切换时保留当前用例，SSE 更新不重置任务。
4. 核对数据库或正式读回：模板创建和修改真实发生，最后测试资源已清理，运行 verdict=passed。
5. 同一场景连续三次执行，每次使用独立数据与批次，均达到相同业务预期，无额外遗留资源。
6. 通过表头全选当前列表或逐条勾选，出现批量入口；在准备弹窗核对每条业务目的和清理影响，按需展开预填输入，输入错误自动展开对应表单；点击开始后验证 8 条真实用例按序运行并分别报告；活动执行期间的新提交得到忙碌提示。运行准备中的“返回”不创建批次，执行记录可回看原快照。

## 4. 页面验收：失败、恢复与清理

- 制造业务断言失败：界面显示具体预期与实际值，不能用任务 completed 或接口 202 代替结论。
- 刷新、关闭后重开：快照恢复，后台继续执行，已保存步骤与新事件一致。
- 在正常会话下断开更新连接：显示连接异常；点击重连后按游标补读，事件没有遗漏或重复展示，不增加轮询。
- 取消可等待场景：先显示停止中，协作式退出后清理，最终为 cancelled 且非通过。
- 制造部分 Prepare、超时、panic 和清理失败：已登记资源仍进入清理，未完成证据和剩余资源可定位；显式重新清理后保留原 verdict。
- 中断自有测试实例并恢复：原执行为 interrupted，不自动重放；仅在实施验收授权范围内操作该实例，不停止或重启用户服务。
- 切换两个租户、身份或宿主会话 revision：关闭旧连接，旧记录和游标不能跨租户访问。
- 以无效 UUID、未知场景、不匹配 profile 或 schema 错误输入提交：明确拒绝，不建立成功运行记录。
- 以空必要断言列表或重复 key 注册场景：明确拒绝；正常执行漏报必要 key、只记录可选断言、重复报告时不能通过，详情列出 missing_assertion_keys 或重复原因。
- 将必要断言降为可选、标为 blocked/skipped 或移除其证据：不能获得通过结果；已阻塞/取消的未执行断言保留原原因，不伪造业务值。
- 删除或修改事件 meta 必填项、tenant_uuid/来源/版本：保存与推送明确拒绝；重连后 source_plugin、occurred_at、payload_version、request_id、trace_id 保持原持久化值。

## 5. 页面验收：真实集成

1. 在具备真实授权的专用测试环境选择 live profile 和标签绑定生命周期场景。
2. 使用 live profile 在后端固定绑定的专用资源，输入其 resource_type 和既有测试标签 UUID；场景先捕获原标签集合，经正式 Core 客户端绑定并读回。
3. 重复绑定后检查业务结果与绑定数量符合幂等预期，清理恢复原标签集合。场景不创建、停用或删除既有标签。
4. 保留真实身份、授权结果、请求/Trace、业务读回和清理证据；不能以 transport mock 或 UI 提示替代。
5. 在不配置或撤销必要授权的环境执行，结果为 blocked，理由可见，无 controlled fallback。

## 6. CLI 与自动化验收

以下为当前实现的命令。REGRESSION_API_BASE、REGRESSION_CASE_UUID、REGRESSION_PROFILE_UUID 和测试凭据由操作者配置；不要把凭据写入文档或版本控制。

```bash
px-plugin regression list --api-base "$REGRESSION_API_BASE" --token-env REGRESSION_TEST_TOKEN --json
px-plugin regression run --api-base "$REGRESSION_API_BASE" --token-env REGRESSION_TEST_TOKEN --profile "$REGRESSION_PROFILE_UUID" --case "$REGRESSION_CASE_UUID" --plugin-id "$REGRESSION_PLUGIN_ID" --inputs ./tmp/regression-inputs.json --request-uuid "$REGRESSION_REQUEST_UUID" --wait --report ./tmp/regression-run.json
px-plugin regression report --api-base "$REGRESSION_API_BASE" --token-env REGRESSION_TEST_TOKEN --run "$REGRESSION_RUN_UUID" --format junit --output ./tmp/regression-run.xml
```

- 核对页面与 CLI 报告的业务字段、判定与结构一致；不同运行的 UUID 和时间不要求相同。
- 验证重复传输请求不会重复执行；主动重跑失败场景生成关联的新运行。
- --wait 使用 SSE，连接失败明确报告可恢复 run_uuid，不假装通过或取消后台执行。
- `run --wait` 退出 0 仅用于 passed 且清理成功/无需清理；参数/鉴权/HTTP/文件错误退出 1；已结束但未通过退出 2；SSE 中断退出 3。未加 `--wait` 的退出 0 只表示提交被接受。
- 多选使用重复的 `--case` 参数，每次传一个目录返回的 UUID；逗号拼接不是该契约，必须明确拒绝。
- CI 默认在临时隔离实例运行 controlled；live 仅在显式配置真实环境时运行，阻塞不能记为通过。

## 7. 开发检查与证据分层

实施后的检查包括：

- Framework 新模块合同与关键执行机制、GORM 租户事务和原子事件更新。
- Skeleton 业务场景、鉴权/租户隔离、迁移幂等和回滚、PostgreSQL RLS。
- Framework Client 类型/会话/SSE；Nuxt Vitest、build 与真实浏览器工作台操作。
- 现有 sync:templates 检查、px-plugin init 生成项目，验证注册、构建与场景执行闭环。
- 当前 Skeleton 的 npm lint 入口仍为占位；实施验收不能单独以其退出码作为 lint 通过证据。

验证记录放在本 feature 的 evidence 目录，由实施任务创建真实记录，不提前生成成功报告。记录至少区分源码/合同、数据库/迁移、受控场景、浏览器、真实 Core 授权与业务读回。

如启动临时后端或前端，记录命令、端口、父/子 PID；结束、失败或中断后仅关闭自有进程并确认端口释放。真实 grant/revoke、用户服务重启、发布或其他仓库修改需另行明确授权。

## 8. 当前可执行的本地验收

```bash
bash scripts/testing/regression.sh
python3 scripts/testing/regression-postgres.py
python3 scripts/testing/regression-browser.py
```

前端先安装依赖并构建，浏览器脚本使用 `.output`，所以前端变更后须重建。也可设置 `REGRESSION_POSTGRES=1 REGRESSION_BROWSER=1` 执行总脚本。测试脚本创建并关闭自己的临时服务，报告保存在 `artifacts/regression/`；该目录不进入脚手架或发布包。完整操作指南见 [guide.md](../../docs/guides/features/025-framework-business-regression/guide.md)。

## 9. 可重复的生成插件页面验收

```bash
python3 scripts/testing/regression-generated.py
```

该脚本在临时目录生成 alpha／beta，打包当前 Framework 的 allowlist，分别安装依赖、构建各自 Nuxt、连接各自生成后端验证页面与 CLI，最终清理临时工程。使用隔离 npm 11.6.2 处理本机 npm 11.3 的 peer resolver 错误；不修改全局 npm，不使用 `--legacy-peer-deps`，不共享两个工程的 node_modules。仍属于未发布本地 Framework 验收。

也可设置 `REGRESSION_GENERATED_BROWSER=1` 执行总脚本。CI 的 business-regression job 现已启用 Skeleton 的 Chromium 业务闭环；本次没有触发远端 CI。只读 live 前置检查见 [live-integration.md](evidence/live-integration.md)。
