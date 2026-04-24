# Framework Logger 跨项目对齐执行说明（绝对路径版）

适用对象：需要把现有插件对齐到 PowerXPlugin 统一日志机制的团队。  
执行方式：按本文顺序执行，所有路径均使用绝对路径。

## 1. 统一规范入口（先读）

- 统一接入指南：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/docs/guides/develop/framework-logger-unification.md`
- 本特性 spec：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/020-framework-logger/spec.md`
- quickstart：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/020-framework-logger/quickstart.md`
- API 合同：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/specs/020-framework-logger/contracts/framework-logger.openapi.yaml`

## 2. 对齐目标（必须同时满足）

1. 插件业务代码不再直写 `logrus/zap/file`。
2. 插件统一通过 framework/skeleton logger 门面输出。
3. 宿主模式（`POWERX_PROXY=1`）默认 `stdout + json`。
4. runtime logging 管理接口与 probe 接口契约一致（`code/message/data`）。
5. logger guard 扫描结果为 `status=resolved`。

## 3. 必改文件（参考本仓库基线）

以下文件是本次标准实现的对齐锚点，其他项目应按同类位置完成同等改造：

- framework runtime logging：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/policy.go`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/router.go`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go/runtime/common/logging/facade.go`
- skeleton logger 兼容层：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/logger/logger.go`
- runtime logging admin 接口：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_policy_handler.go`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/logging_probe_handler.go`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/runtime_ops/routes.go`
- 租户隔离辅助：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/transport/http/admin/common/tenant.go`
- 默认配置（宿主优先 stdout）：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/config/config.go`
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin/internal/observability/security/audit_writer.go`
- 治理脚本：
  - `/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/scripts/testing/framework-logger-guard.sh`

## 4. 其他项目对齐执行步骤

1. 升级依赖到 framework go `v0.0.7+`。
2. 迁移日志调用到 framework/skeleton logger。
3. 接入 runtime logging 三接口：
   - `GET /api/v1/admin/runtime/logging/policy`
   - `PUT /api/v1/admin/runtime/logging/policy`
   - `POST /api/v1/admin/runtime/logging/probe`
4. policy/probe 成功响应统一为：
   - `{"code":0,"message":"ok","data":...}`
5. policy 按 tenant_uuid 隔离，禁止跨租户改策略。
6. 宿主模式强制 `host + stdout + json`。
7. 跑治理扫描与回归测试，直至全部通过。

## 5. 标准回归命令（绝对路径）

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
FRAMEWORK_LOGGER_GUARD_MODE=warn \
  /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/scripts/testing/framework-logger-guard.sh \
  /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/skeleton/backend/go-gin \
  /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/framework/backend/go
```

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
mkdir -p /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tmp/gocache \
         /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tmp/gomodcache
GOCACHE=/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tmp/gocache \
GOMODCACHE=/private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin/tmp/gomodcache \
go test ./skeleton/backend/go-gin/... -count=1
```

```bash
cd /private/var/www/html/ArtisanCloud/X/PowerX/Core/Plugins/PowerXPlugin
make dist DIST_DIR=dist/0.1.1
```

## 6. 对接 PowerX 的验收口径

1. PowerX 经插件转发路径可访问 policy/probe 接口（不 404）。
2. PUT policy 返回 200 + 最终生效策略。
3. POST probe 返回 outcomes（status 枚举：`success|failed|retrying|dropped`）。
4. 在 PowerX 监控侧可按 `trace_id` 查到 probe 日志。

