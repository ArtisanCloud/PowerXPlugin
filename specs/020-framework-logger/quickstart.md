# Quickstart: Framework 统一日志适配

## 1. 目标

验证插件通过 framework 统一日志门面后，能够在宿主模式默认走 `stdout + json`，并支持受控多 sink、故障降级重试与治理检查。

## 2. 前置条件

1. 当前分支为 `020-framework-logger`。  
2. PowerX 宿主可用，且可注入日志策略。  
3. 插件后端可启动，具备管理端访问能力。  
4. 如需验证扩展 sink，准备好可用 `file` 目录或 Loki 接入配置。

## 3. 宿主模式验证（P1）

1. 设置宿主模式运行环境（`POWERX_PROXY=1`）。  
2. 下发或加载默认日志策略（仅 stdout、json）。  
3. 触发至少一条业务日志（建议带 `trace_id`）。  
4. 在宿主日志检索中确认：
   - 日志存在；
   - 字段包含 `plugin_id tenant_uuid component level trace_id`；
   - 可通过同一 `trace_id` 关联链路。

## 4. 多 sink 并行与降级验证（P2）

1. 配置策略开启多个 sink（如 `stdout + file` 或 `stdout + loki`）。  
2. 触发 probe 日志并查看每个 sink 的路由结果。  
3. 人为使一个 sink 不可用（例如无效 Loki 地址）。  
4. 验证：
   - 其余 sink 输出正常；
   - 失败 sink 有告警与重试记录；
   - 主业务请求不受阻断。

## 5. 业务日期口径验证

1. 触发跨时区业务日志样本。  
2. 验证日志同时包含：
   - UTC 时间戳；
   - `biz_date`；
   - `biz_tz`。  
3. 确认统计口径一致，不因运行节点时区变化而漂移。

## 6. 遗留直写日志治理验证（P3）

1. 扫描现有代码中的直写日志调用。  
2. 确认治理状态能区分 `detected/warned/blocked/resolved`。  
3. 配置治理截止版本并验证：
   - 截止前：可告警不阻断；
   - 截止后：违规项进入阻断。

## 7. 回归建议命令

```bash
mkdir -p tmp/gocache tmp/gomodcache
GOCACHE=$PWD/tmp/gocache GOMODCACHE=$PWD/tmp/gomodcache \
  go test ./framework/backend/go/runtime/common/logging -count=1
```

```bash
mkdir -p tmp/gocache tmp/gomodcache
GOCACHE=$PWD/tmp/gocache GOMODCACHE=$PWD/tmp/gomodcache \
  go test ./skeleton/backend/go-gin/internal/config \
          ./skeleton/backend/go-gin/internal/logger \
          ./skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops \
          -count=1
```

```bash
# 治理扫描（默认 warn；mode=block 或触发截止版本时返回非 0）
FRAMEWORK_LOGGER_GUARD_MODE=warn \
  ./scripts/testing/framework-logger-guard.sh ./skeleton/backend/go-gin ./framework/backend/go
```

## 8. 执行与排障说明

1. `POWERX_PROXY=1` 但 `file/loki` 未生效：检查 `authorized_extra_sinks` 是否授权，宿主默认只允许 `stdout`。
2. `PUT /api/v1/admin/runtime/logging/policy` 返回 400：检查 `mode/format/level/sinks/retry` 是否满足校验（host 必须 `stdout+json`）。
3. `probe` outcomes 出现 `dropped`：先检查 sink 注册状态，再检查请求超时导致的重试中断。
4. CI 中治理扫描失败：若 `status=blocked`，根据输出清单整改直写日志，或核对 `plugin_version` 与 `governance_deadline_version`。
5. Go 测试缓存权限错误：固定使用 `GOCACHE=$PWD/tmp/gocache` 与 `GOMODCACHE=$PWD/tmp/gomodcache`。

## 9. 契约联调最小检查

1. `GET /api/v1/admin/runtime/logging/policy` 成功返回 `code=0`，`data` 直接为策略对象。  
2. `PUT /api/v1/admin/runtime/logging/policy` 成功返回 `code=0`，HTTP 200，`data` 为最终生效策略。  
3. `POST /api/v1/admin/runtime/logging/probe` 成功返回 `code=0`，`data.outcomes[*].status` 枚举为 `success|failed|retrying|dropped`。
## 10. 验收映射（对应 Success Criteria）

1. **SC-001**: 新增代码无直写日志；截止版本后违规数为 0。  
2. **SC-002**: 抽样日志可用 `trace_id` 串联查询。  
3. **SC-003**: 单 sink 故障不影响主业务成功率，且有告警/重试。  
4. **SC-004**: 跨插件字段一致性检查全部通过。  
5. **SC-005**: 记录排障时长并与基线比较，验证下降目标。
