# Quickstart: Async Runtime Scheduler 模式切换验收

## 1. 前置条件

1. 当前分支：`017-async-runtime-scheduler-switch`
2. 已启动插件后端，具备 admin 调试凭证（`$USER_TOKEN`）
3. 可访问 runtime 调试接口与 WS 订阅入口
4. 已准备两种模式配置：
   - standalone local：`POWERX_PROXY=0` + `taskbus_provider=redis`
   - delegated proxy：`POWERX_PROXY=1` + `taskbus_provider=host`

## 2. 配置冲突 fail-fast 验证

制造冲突组合（例如 `POWERX_PROXY=1` + `taskbus_provider=redis`），启动服务。

验收标准：

1. 启动失败（或明确阻断 scheduler）
2. 返回明确冲突错误（可定位）
3. 不得出现静默继续执行

## 3. Standalone Local 验收

### 3.1 订阅 WS

```bash
wscat -c "ws://127.0.0.1:8078/api/ws?authorization=Bearer%20$USER_TOKEN"
```

### 3.2 触发手动链路（对照组）

```bash
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/event-bridge/emit \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"topic":"powerx.runtime.scheduler.triggered.v1","payload":{"source":"manual-check","progress":10}}'
```

### 3.3 触发调度链路（目标组）

使用调度任务触发同一 topic（或等价调试入口模拟 cron 触发）。

验收标准：

1. 两组触发均可收到 `ack + event`
2. 状态流转与业务结果一致（允许耗时差异）

## 4. Delegated Proxy 验收

### 4.1 模式与凭证

```bash
export POWERX_PROXY=1
export PX_GATEWAY_AUTH_SCHEME=apikey
export PX_GATEWAY_API_KEY=<key>
```

### 4.2 联调顺序

1. 创建 topic
2. grant ACL
3. 执行手动触发与调度触发
4. 对比语义一致性

参考：

1. `docs/guides/async_runtime/event_fabric/integration_playbook.md`
2. `docs/guides/async_runtime/websocket/debug_playbook.md`

## 5. 权限失败与重试闭环验证

构造 proxy 权限失败场景（topic 未授权或 key 快照缺失）。

验收标准：

1. 任务进入“有上限重试”
2. 重试超限后自动创建工单
3. 对应任务进入暂停状态
4. 非运维/管理员角色不能恢复
5. 运维/管理员恢复后任务可重新触发

最小闭环命令（示例 `dispatch_id=dispatch-us3-001`）：

```bash
# A. 连续重试直到超限（默认上限 3）
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/dispatches/dispatch-us3-001/retry \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"error_code":"AUTH_FORBIDDEN","error_message":"topic not allowed"}'
```

预期：前两次 `202`，第三次 `409(CONFLICT)`。

```bash
# B. 超限后暂停并创建工单
curl -sS -X POST http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/dispatches/dispatch-us3-001/pause \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"paused_job_id":"job-us3-001"}'
```

预期：返回 `201`，响应中包含 `ticket_id`。

```bash
# C. 非 ops/admin 恢复（应失败）
curl -sS -X POST "http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/tickets/$TICKET_ID/resume" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"operator_role":"viewer","operator_id":"qa-user","reason":"try-resume"}'
```

预期：`403(FORBIDDEN)`。

```bash
# D. ops/admin 恢复（应成功）
curl -sS -X POST "http://127.0.0.1:8078/api/v1/admin/runtime/scheduler/tickets/$TICKET_ID/resume" \
  -H "Authorization: Bearer $USER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"operator_role":"ops","operator_id":"ops-user","reason":"permission fixed"}'
```

预期：`200` 且 `ticket_status=resolved`，随后再次执行 retry 返回 `202`。

## 6. 指标与日志检查

```bash
curl -sS http://127.0.0.1:8078/api/v1/admin/runtime/metrics | rg 'plugin_event_bridge_(emit_total|drop_total|latency_ms)'
```

```bash
rg 'trace_id|topic|status|gateway_auth_scheme|outbound_token_source' <runtime-log-file>
```

## 7. 验收结论模板

1. 冲突配置是否 fail-fast：PASS/FAIL
2. 双模式触发语义一致：PASS/FAIL
3. 权限失败重试闭环完整：PASS/FAIL
4. 恢复权限边界正确：PASS/FAIL
5. 观测字段完整可检索：PASS/FAIL

## 8. 统计窗口

1. 成功率、首次通过率、定位时长采用发布后 14 天窗口
2. 结果需保留可追溯台账（日期、模式、场景、结论、操作者）

## 9. SC-003 台账模板（首次通过率）

| date | mode | total_checks | first_pass_checks | first_pass_rate | reviewer | notes |
|---|---|---:|---:|---:|---|---|
| YYYY-MM-DD | standalone/proxy | 0 | 0 | 0% | QA |  |

统计步骤：

1. 统计窗口固定为发布后连续 14 天。
2. `total_checks` = 该模式在窗口内执行的标准联调总次数。
3. `first_pass_checks` = 首次执行即通过的次数（同一变更批次仅计首次）。
4. `first_pass_rate = first_pass_checks / total_checks * 100%`。
5. 每行需附带证据引用（日志、命令输出、工单号）。

## 10. SC-004 台账模板（定位时长下降）

| period | ticket_count | avg_locate_minutes | baseline_compare | owner | notes |
|---|---:|---:|---:|---|---|
| pre_release_14d | 0 | 0 | baseline | Ops |  |
| post_release_14d | 0 | 0 | target: -50% | Ops |  |

统计步骤：

1. 取发布前 14 天与发布后 14 天两段窗口，口径一致。
2. `ticket_count` = 与模式切换相关故障工单数。
3. `avg_locate_minutes` = 从告警/报障到根因确认的平均分钟数。
4. `baseline_compare` 使用 `(post - pre) / pre` 计算变化比例。
5. 目标达成条件：`post <= pre * 50%`（下降至少 50%）。

## 11. Phase 6 回归命令记录（2026-03-24）

执行命令：

```bash
mkdir -p tmp/gocache tmp/gomodcache && cd skeleton/backend/go-gin && \
GOCACHE=$PWD/../../tmp/gocache GOMODCACHE=$PWD/../../tmp/gomodcache \
go test ./cmd/plugin ./internal/config ./internal/services/admin/runtime_ops \
  ./internal/transport/http/admin/runtime_ops ./tests/integration \
  -run 'Scheduler|TaskBusProvider|ValidateSchedulerRetryMaxAttemptsRange|DefaultSchedulerConfigValidation' \
  -count=1
```

结果摘要：

1. `cmd/plugin`：PASS
2. `internal/config`：PASS
3. `internal/services/admin/runtime_ops`：PASS
4. `internal/transport/http/admin/runtime_ops`：PASS
5. `tests/integration`：PASS
