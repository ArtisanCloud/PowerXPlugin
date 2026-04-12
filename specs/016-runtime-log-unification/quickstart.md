# Quickstart: Runtime 日志统一对齐验收

## 1. 前置条件

1. 当前分支：`016-runtime-log-unification`
2. backend 可启动并触发 Task/WS 关键链路事件
3. 已开启结构化日志输出

## 2. 运行测试

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
go test ./framework/backend/go/runtime/... ./skeleton/backend/go-gin/internal/...
```

## 3. 关键字段检查

将 `<runtime-log-file>` 替换为实际日志文件路径：

```bash
rg 'trace_id|task_id|tenant_uuid|tenant_key|subscriber_id|topic|status' <runtime-log-file>
```

## 4. 缺失上下文检查

```bash
rg 'status=skipped|reason=missing_context|task_id=unknown|subscriber_id=unknown' <runtime-log-file>
```

## 5. 状态枚举检查

```bash
rg 'status=(queued|processing|succeeded|failed|skipped)' <runtime-log-file>
```

## 6. 样本规模验收

按以下 6 类链路分别统计，且每类不少于 20 条事件：

1. `task.enqueue`
2. `task.consume`
3. `task.ack`
4. `task.fail`
5. `ws.publish`
6. `ws.dispatch`

样本规则：
1. 关键链路采用全量样本，不做降采样
2. 每类样本需同时覆盖成功与失败路径
3. 任一类样本不足 20 条时，本轮验收视为未完成

## 7. 文档对齐验收

确认以下文档口径一致：

1. `specs/016-runtime-log-unification/spec.md`
2. `docs/guides/async_runtime/observability/README.md`
3. `docs/plan/develop/log/runtime-log-field-matrix.md`

## 8. Host / Standalone 对比验证（US2）

按同一事件类型分别在 Host 与 Standalone 触发后，对比日志字段与状态语义：

1. 在 Standalone 模式触发 `ws.publish` 与 `ws.dispatch` 各至少 20 条；
2. 在 Host 模式触发同类事件各至少 20 条；
3. 对两组日志分别检索：

```bash
rg 'topic=(ws.publish|ws.dispatch)|status=(queued|processing|succeeded|failed|skipped)|tenant_uuid=|tenant_key=' <runtime-log-file>
```

4. 抽样核对以下口径必须一致：
   - `tenant_uuid` 为主字段，`tenant_key` 与其镜像一致
   - `status` 仅出现 `queued|processing|succeeded|failed|skipped`
   - 关键扩展字段保留：`gateway_auth_scheme`、`outbound_token_source`、`plugin_id`、`component`

## 9. SC-003 统计口径（首次通过率）

1. 统计窗口：发布后连续 14 天
2. 样本来源：QA 首次执行验收记录
3. 计算方式：`首次通过次数 / 首次验收总次数`

## 10. SC-004 统计口径（返工次数下降）

1. 对比窗口：发布前 14 天 vs 发布后 14 天
2. 样本来源：runtime 日志口径相关排障工单
3. 计算方式：`(发布前返工次数 - 发布后返工次数) / 发布前返工次数`

## 11. Phase 6 执行记录（T031）

执行时间：2026-03-22

1. framework 回归：
   - 命令：`go test ./framework/backend/go/runtime/common/logging ./framework/backend/go/bootstrap ./framework/backend/go/runtime/taskbus ./framework/backend/go/runtime/wsbus -count=1`
   - 结果：PASS（详见 `tmp/phase6-framework-tests.log`）
2. skeleton 回归：
   - 命令：`go test ./skeleton/backend/go-gin/internal/logger ./skeleton/backend/go-gin/internal/shared/app ./skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops -count=1`
   - 结果：PASS（详见 `tmp/phase6-skeleton-tests.log`）

## 12. 日志检索回归与样本统计（T032）

本次演练日志文件：`tmp/runtime-log-regression-20260322.log`（121 条）

样本统计：

1. `task.enqueue`: 20
2. `task.consume`: 20
3. `task.ack`: 20
4. `task.fail`: 20
5. `ws.publish`: 20
6. `ws.dispatch`: 20
7. `status` 枚举匹配条数：121
8. `reason=missing_context`：1
9. 必需字段匹配条数：121

结论：

1. 六类关键链路均达到每类 `>=20` 样本要求
2. 缺失上下文规则可检索
3. 最小字段契约检索通过

## 13. SC-003 样本台账模板（T033）

统计口径：

1. 统计窗口：发布后连续 14 天
2. 指标定义：`首次通过次数 / 首次验收总次数`

台账模板：

| date | env | mode | total_checks | first_pass_checks | first_pass_rate | reviewer | remark |
|---|---|---|---:|---:|---:|---|---|
| YYYY-MM-DD | staging/prod | host/standalone | 0 | 0 | 0% | QA |  |

## 14. 7 天回滚窗口演练记录（T036）

演练时间：2026-03-22  
演练记录文件：`tmp/phase6-rollback-drill-20260322.txt`

触发条件检查结果：

1. 非标准 `status` 条数：0
2. 关键字段空值条数：0
3. 是否触发回滚：否（`rollback_trigger=0`）

演练结论：

1. 当前版本无需触发回滚
2. 回滚窗口执行路径可复用（按 `research.md` 回滚说明）

## 15. 性能基线采集步骤（T037）

采集命令：

```bash
cd framework/backend/go
/usr/bin/time -p env GOCACHE=$PWD/.gocache GOMODCACHE=$PWD/.gomodcache go test ./runtime/taskbus ./runtime/wsbus -count=1
```

采集要求：

1. 同一机器、同一负载窗口执行 3 次取中位数
2. 记录 `taskbus/wsbus` 包耗时与 `real/user/sys`
3. 与改造前基线对比，确认未超 `<5%` 退化阈值
