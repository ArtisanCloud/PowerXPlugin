# Framework 日志统一接入指南（PowerXPlugin）

适用范围：所有基于 PowerXPlugin framework/skeleton 的插件。  
目标：插件统一使用同一套日志机制，支持宿主模式与独立模式一致治理，并对接 PowerX 多日志源采集。

跨项目落地请配合使用：

- `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/guides/develop/framework-logger-alignment-instruction.md`

## 1. 背景与本次调整目的

过去插件侧存在三类分散问题：

- 业务代码直接调用 `logrus`/文件输出，无法统一策略管理。
- 宿主模式（`POWERX_PROXY=1`）下日志输出与 PowerX 采集链路不稳定。
- 日志探针（policy/probe）接口格式不一致，PowerX 监控中心联调成本高。

本次调整的目标是：

- 插件只调用 framework/skeleton 提供的统一日志能力。
- 宿主模式默认 `stdout + json`，由 PowerX 统一采集到 Loki/其他 sink。
- 提供标准运维接口：`GET/PUT policy` + `POST probe`，供 PowerX 监控中心统一控制。

## 2. 本次已完成改动（已落地）

### 2.1 skeleton 全量对齐 logger 机制

- 已将 skeleton 中历史日志调用点对齐到统一 logger 兼容层。
- `framework logger guard` 校验结果：`status=resolved`，`violations=0`。

关键文件：

- `skeleton/backend/go-gin/internal/logger/logger.go`
- `scripts/testing/framework-logger-guard.sh`

### 2.2 framework 版本与兼容层

- skeleton backend 依赖升级到：
  - `github.com/ArtisanCloud/PowerXPlugin/framework/backend/go v0.0.7`
- 补齐 logger 兼容接口（`Entry/Level/New/NewEntry/StandardLogger`），确保旧调用点可平滑运行。

关键文件：

- `skeleton/backend/go-gin/go.mod`
- `skeleton/backend/go-gin/internal/logger/logger.go`

### 2.3 审计日志默认输出改为 stdout

- 默认 audit channel 改为 `stdout`，避免宿主模式下插件私有落盘。
- `AuditWriter` 支持 `stdout/stderr/file`，由配置选择。

关键文件：

- `skeleton/backend/go-gin/internal/config/config.go`
- `skeleton/backend/go-gin/internal/observability/security/audit_writer.go`

### 2.4 runtime logging 接口契约对齐（PowerX 联调）

已对齐插件接口（注意：由插件提供，PowerX 通过插件转发路径访问）：

- `GET /api/v1/admin/runtime/logging/policy`
- `PUT /api/v1/admin/runtime/logging/policy`
- `POST /api/v1/admin/runtime/logging/probe`

主要对齐点：

- 成功响应统一：`{"code":0,"message":"ok","data":...}`
- `PUT /policy` 返回 200 与最终生效策略。
- policy 按 `tenant_uuid` 隔离，不允许跨租户修改。
- probe 使用租户严格校验后的 tenant 上下文。

关键文件：

- `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_policy_handler.go`
- `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_probe_handler.go`
- `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/routes.go`
- `skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/rbac.go`

## 3. 统一日志机制：运行策略

### 3.1 宿主模式（`POWERX_PROXY=1`）

- framework 规则：默认强制 `mode=host`。
- 输出规则：默认 `sinks=[stdout]` + `format=json`。
- 目的：把插件日志统一交给 PowerX 底座采集链路。

### 3.2 standalone/dev 模式

- 默认可用 `stdout`。
- 可按策略启用 `file` 或 `loki`。
- 建议本地调试短期使用 file，长期仍建议标准化 stdout json。

### 3.3 低基数字段原则

建议固定低基数字段：

- `plugin_id`
- `tenant_uuid`
- `component`
- `level`

高基数业务标识（如 `message_id/order_id`）仅放日志 JSON 字段，不做 labels。

## 4. 插件开发使用方式（标准）

### 4.1 业务代码日志调用（framework facade）

```go
import runtimelogging "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/common/logging"

func handle(ctx context.Context) {
    runtimelogging.FromContext(ctx).With(runtimelogging.Fields{
        runtimelogging.FieldComponent: "example.order",
        "channel": "wecom",
    }).Emit("info", "order event emitted", runtimelogging.Fields{
        "event": "order.created",
        "order_id": "o_123",
    })
}
```

### 4.2 skeleton 兼容层调用（遗留代码过渡）

```go
import pxlog "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/logger"

func legacy() {
    pxlog.WithField("component", "legacy.module").Info("legacy compatible log")
}
```

迁移原则：新代码优先 framework facade，遗留代码可先走 skeleton logger 兼容层。

## 5. PowerX 对接说明

PowerX 不需要调用插件内部 logger 实现，只需要调用插件运维接口。

调用路径（宿主转发）通常为：

- `/_p/<plugin_id>/api/v1/admin/runtime/logging/policy`
- `/_p/<plugin_id>/api/v1/admin/runtime/logging/probe`

链路职责：

- PowerX：接口编排、权限与多日志源汇聚。
- 插件：根据 mode/policy 决定 sink 并输出结构化日志。

## 6. 其他插件迁移清单（可直接执行）

1. 升级依赖到 `framework/backend/go >= v0.0.7`。
2. 禁止业务代码直接 `logrus`/`zap`/文件落盘。
3. 新日志点统一使用 framework facade 或 skeleton logger 兼容层。
4. 接入 runtime logging 三接口（policy/probe）。
5. 确认宿主模式默认 `stdout + json`。
6. 通过 logger guard 校验并清零违规。

## 7. 验收与回归命令

在仓库根目录执行：

```bash
# 1) 日志治理检查（必须为 resolved）
FRAMEWORK_LOGGER_GUARD_MODE=warn ./scripts/testing/framework-logger-guard.sh ./skeleton/backend/go-gin ./framework/backend/go

# 2) 后端全量测试
GOCACHE=$PWD/tmp/gocache GOMODCACHE=$PWD/tmp/gomodcache go test ./skeleton/backend/go-gin/... -count=1

# 3) 打包安装产物
make dist DIST_DIR=dist/0.1.1
```

产物目录：`skeleton/dist/0.1.1`

## 8. 联调验收（PowerX 监控中心）

安装插件到宿主后，至少验证：

1. `GET policy` 不 404，返回 `code/message/data`。
2. `PUT policy` 成功返回最终策略。
3. `POST probe` 返回 `outcomes`。
4. 使用 `trace_id` 可在 PowerX 监控日志检索到 probe 日志。
