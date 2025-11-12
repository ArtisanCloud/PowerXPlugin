# Go CLI 热加载调试指南（dev --watch）

> 适用对象：插件开发者、联调/审核支持工程师  
> 场景定位：在「发布前」阶段，通过 `px-plugin dev --watch` 将本地产物热加载到 Dev API，验证权限、菜单、回滚与 Telemetry，确保进入 Publish Hub 流水线前具备可观测性。

---

## 1. 能力概览

| 能力 | 说明 |
|------|------|
| 热加载 | 递归监听插件目录，250ms 去抖后增量打包并推送到 Dev API。 |
| 会话管理 | `list-sessions / resume / stop / logs` 让多个调试会话可并行、可恢复。 |
| mTLS | 完整复用 `px auth configure` 生成的证书，支持 ServerName/SNI、自签与轮换。 |
| 资源守卫 | 限制 CPU/内存/文件监控数量，防止 watch 任务拖垮宿主机。 |
| 审计与 Telemetry | `~/.px-plugin/audit/*.log` 与 `plugin.dev.reload.*` 指标，供 Publish Hub 审核追溯。 |
| 故障自愈 | 自动退避 + 回滚机制，保证 reload 失败后可以恢复到上一次成功版本。 |

Recommended versions:

- Go 1.24+（构建 CLI 二进制）
- Node.js 18+/npm 9+（用于插件本身的依赖安装）
- OpenSSL 3.0+（验证证书）
- PowerX Dev API 已启用 `PX_PLUGIN_DEV_MODE` 和 mTLS

---

## 2. 快速开始

1. **构建 Go CLI 二进制**
   ```bash
   cd /path/to/PowerXPlugin/tools/cli
   go build -o px-plugin ./cmd/px-plugin
   # 可选：安装到 GOPATH/bin
   go install ./cmd/px-plugin
   ```
   验证版本：
   ```bash
   px-plugin --version
   ```

2. **准备 mTLS 证书（PowerX 平台 CLI）**
   > 若尚未安装 `px` CLI，可在 PowerX 主仓运行：  
   > ```bash
   > cd /path/to/PowerX/backend
> go install ./cmd/px                                  # 安装到 $(go env GOPATH)/bin
> # 指定版本号（可选）
> go install -ldflags "-X main.version=v0.1.0" ./cmd/px
   > export PATH="$(go env GOPATH)/bin:$PATH"             # 或设置到 shell profile
   > px --help                                            # 验证命令可用
   > ```

   ```bash
   # 在 PowerX 平台主仓（或已安装 px CLI 的目录）执行
   cd /path/to/PowerX
   ./bin/px auth configure

   # 验证证书已生成
   ls ~/.powerx/cli   # client.crt / client.key / ca.crt
   ```
   > `px` CLI 隶属于 PowerX 平台仓库，用于与 Dev/Marketplace 环境交互；本仓库构建的是 `px-plugin` CLI。两者需要配套使用：先用 `px auth configure` 写入凭证，再在 PowerXPlugin 仓库调用 `px-plugin dev --watch`。

3. **启动热加载（在通过 `px-plugin init` 生成的插件工程目录）**
   ```bash
   cd /path/to/your-plugin   # 例如 plugins/com.powerx.example，由 px-plugin init 创建
   px-plugin dev --watch \
     --entry . \
     --tenant demo-tenant \
     --dev-api https://dev-api.powerx.local
   ```

4. **观察输出**
   - CLI 将展示 `sessionId`、`reloadToken`、热加载延迟（P95）以及 SSE 日志端点。
   - `CTRL+C` 会安全退出并调用 `DELETE /internal/dev/plugins/register/{sessionId}` 释放会话。

5. **查看会话与日志**
   ```bash
   px-plugin dev --list-sessions
   px-plugin dev --logs <session-id>
   px-plugin dev --stop <session-id>
   ```

---

## 3. 命令与参数速查

### 3.1 `px-plugin dev --watch`

| Flag | 说明 |
|------|------|
| `--entry <path>` | 插件目录（必填），建议指向 dist 或仓库根目录。 |
| `--tenant <id>` | 目标调试租户，默认 `default`。 |
| `--dev-api <url>` | Dev API 地址，缺省为 `http://127.0.0.1:8077`。 |
| `--ignore <pattern>` | 排除文件/目录，支持多次传入。 |
| `--mtls-cert/key/ca` | 覆写证书路径，默认读取 `~/.powerx/cli/*`。 |
| `--mtls-server-name` | 自定义 TLS SNI，用于多租或测试环境。 |
| `--mtls-skip-verify` | 仅限本地调试，跳过服务端证书校验。 |
| `--logs-level` / `--logs-file` | 控制 CLI 日志级别与输出位置。 |
| `--max-procs` | 限制 Go runtime 的最大线程数，默认取机器 CPU 核心或 `PX_MAX_PROCS`。 |
| `--max-memory-mb` | 资源警戒线（默认 100MB），超过后降级为串行 reload。 |
| `--max-cpu-percent` | CPU 阈值（默认 10%），可通过 `PX_RESOURCE_CPU_THRESHOLD` 配置。 |
| `--max-watch-files` | 默认 10000，防止监听树过大。 |

示例：
```bash
px-plugin dev --watch \
  --entry ./dist \
  --tenant qa-tenant \
  --dev-api https://dev-api.powerx.dev \
  --ignore "node_modules/**" \
  --logs-level debug
```

### 3.2 会话相关命令

| 命令 | 用途 |
|------|------|
| `px-plugin dev --list-sessions` | 查看本机所有活跃/历史会话，含 reload 统计。 |
| `px-plugin dev --resume <session-id>` | 重新附着到已存在的会话，维持同一 `reloadToken`。 |
| `px-plugin dev --stop <session-id>` | 主动终止远端会话。 |
| `px-plugin dev --logs <session-id>` | 订阅 SSE 日志，可指定 `--logs-level` 和输出文件。 |

输出示例：
```
ID:        session-17309790
Plugin:    com.powerx.example v1.4.0
Tenant:    demo-tenant
Status:    active
Reloads:   18 (avg 230ms, success 100%)
```

### 3.3 多插件项目如何隔离？

- **独立会话 ID**：每次执行 `px-plugin dev --watch` 都会向 Dev API 注册一个新的 `sessionId`，并把 `pluginId`、`tenant`、`entryPath`、`manifestHash` 等信息写入 `~/.px-plugin/sessions/{sessionId}.json`。不同插件或不同目录自然对应不同的 session 文件。
- **按 entry 监听**：CLI 只会监听 `--entry` 指定目录（默认当前目录），`fsnotify` watcher 与资源阈值都局限在该目录树内，不会互相干扰。
- **后端隔离**：Dev API 以 `pluginId + tenant + sessionId` 为维度维护热加载上下文，同一租户可以同时接入多个插件，每个会话的 reload/rollback 日志、SSE 流都独立。
- **命令行管理**：`px-plugin dev --list-sessions` 会列出本机所有会话，你可以通过 `resume/stop/logs` 指定某个 `sessionId` 操作，避免串线。

因此，只需在不同的插件仓库（或不同 entry 目录）分别运行 `px-plugin dev --watch`，就能在同一 Dev API 上实现多项目并行调试，彼此之间既不会共享文件监听，也不会互相覆盖 session。

---

## 4. 配置来源

CLI 读取配置的优先级：命令行 > 环境变量 (`PX_*`) > `~/.px-plugin/config.json` > 内建默认值。

### 4.1 配置文件示例

`~/.px-plugin/config.json`
```json
{
  "global": {
    "logLevel": "info",
    "cacheDir": "~/.px-plugin/cache"
  },
  "devApi": {
    "baseUrl": "https://dev-api.powerx.dev",
    "timeout": 30,
    "retries": 3,
    "enableMtls": true,
    "certPath": "~/.powerx/cli/client.crt",
    "keyPath": "~/.powerx/cli/client.key",
    "caCertPath": "~/.powerx/cli/ca.crt"
  },
  "watch": {
    "debounceMs": 250,
    "maxFiles": 10000,
    "ignore": ["**/.git/**", "**/node_modules/**"]
  },
  "performance": {
    "maxConcurrency": 12,
    "hashCacheSize": 5000
  }
}
```

### 4.2 常用环境变量

```bash
export PX_DEV_API_BASEURL=https://dev-api.powerx.dev
export PX_TENANT=demo-tenant
export PX_MTLS_CERT_PATH=~/.powerx/cli/client.crt
export PX_MTLS_KEY_PATH=~/.powerx/cli/client.key
export PX_MTLS_CA_PATH=~/.powerx/cli/ca.crt
export PX_RESOURCE_MEMORY_MB=150
export PX_RESOURCE_CPU_THRESHOLD=15
export PX_MAX_WATCH_FILES=20000
```

---

## 5. 安全与证书

1. **获取证书**：执行 `px auth configure`，默认保存到 `~/.powerx/cli/`。  
2. **验证有效期**：
   ```bash
   openssl x509 -in ~/.powerx/cli/client.crt -noout -dates
   ```
3. **配置 CLI**：通过配置文件或环境变量指向证书位置。  
4. **证书轮换**：在 `~/.px-plugin/config.json` 中开启
   ```json
   "security": {
     "enableMtls": true,
     "autoRotate": true,
     "rotationCheck": 5
   }
   ```
   CLI 会每 5 分钟检测证书有效期，小于 30 天会提示重新执行 `px auth configure`。

---

## 6. 性能指标与优化

| 指标 | Go CLI | TypeScript CLI | 优势 |
|------|--------|----------------|------|
| 启动耗时 | ~150ms | ~400ms | -62.5% |
| 内存占用 | ~45MB | ~80MB | -43.8% |
| Reload P95 | ~1.2s | ~2.1s | -42.9% |
| CPU 使用 | ~8% | ~15% | -46.7% |

优化建议：

1. **增大并发**：`performance.maxConcurrency` 调整为接近 CPU 线程数。  
2. **开启增量构建**：在插件 `build` 流程中保留缓存，CLI 即可只重建变更文件。  
3. **调节 debounce**：小改动可把 `watch.debounceMs` 调低至 100-150ms。  
4. **哈希缓存**：`hashCacheSize` 设置为 5000+，减轻重复文件扫描成本。  
5. **观测指标**：Grafana `dev-hotload` dashboard 重点关注 `dev.hotload.go_cli_reload_duration_ms` 与 `dev.hotload.go_cli_memory_bytes`。

---

<a id="health-checks"></a>
## Health Checks（诊断）

`px-plugin doctor` 会生成 `.doctor/report.json`，并在终端展示各项检查结果。常用选项：

| Flag | 含义 |
|------|------|
| `--entry <path>` | 指定插件目录，默认为当前路径。 |
| `--dev-api <url>` | 覆盖 Dev API 基址。 |
| `--check-env` / `--check-devapi` / `--check-mtls` / `--check-watch` | 只执行特定检查。 |
| `--output <file>` | 将 JSON 报告写入指定文件。 |
| `--mtls-*` | 复用 dev 命令的证书参数。 |

示例：
```bash
px-plugin doctor --entry . --check-devapi
px-plugin doctor --check-mtls --mtls-cert ~/.powerx/cli/client.crt
```

报告示例：
```json
{
  "generatedAt": "2025-11-02T09:21:33Z",
  "entryPath": "/repo/plugins/sample-plugin",
  "devApiBase": "https://dev-api.powerx.dev",
  "results": [
    {
      "name": "Dev API",
      "status": "pass",
      "details": "Register/Delete handshake succeeded",
      "durationMs": 842
    },
    {
      "name": "mTLS",
      "status": "warn",
      "details": "mTLS not configured; using plain HTTP"
    }
  ]
}
```

审计/质检在 Publish Hub 工单中可直接附上该报告，证明调试环境满足准入条件。

---

<a id="error-recovery--rollback"></a>
## Error Recovery & Rollback（错误恢复与回滚）

Go CLI 内置指数退避与回滚机制：

1. **指数退避**：API 调用失败会按 `1s → 2s → 4s → 8s → 30s` 的节奏重试，防止抖动。
2. **自动回滚**：只要有一次成功的 `reload`，后续失败会自动发送 `strategy=rollback` 请求，把 Dev API 恢复到上一版本。
3. **审计记录**：`~/.px-plugin/audit/*.log` 中可看到 `reload.fail` → `rollback` → `reload.success` 的完整链路，包含 `rollbackReason` 与 `rollbackAt`。

若连续失败，可通过：
```bash
px-plugin dev --logs <session-id> --logs-level debug
tail -f ~/.px-plugin/audit/*.log
```
定位问题，再结合 `px-plugin doctor` 修复。

---

## 7. 常见问题（FAQ）

| 报错 | 可能原因 | 解决方案 |
|------|----------|----------|
| `tls handshake failed` | 证书过期 / SNI 不匹配 | 重新执行 `px auth configure`，或加 `--mtls-server-name dev.powerx.local`。 |
| `watch limit exceeded (10000)` | 监听目录过大 | 使用 `--max-watch-files` 或在配置文件调整 `watch.maxFiles`；排除 `node_modules`。 |
| `reload failed: context deadline exceeded` | Dev API 拥塞或网络不通 | 检查 `PX_DEV_API_BASEURL`、VPN/代理；必要时提高 `devApi.timeout`。 |
| `rollback missing bundle` | 从未成功 reload | 先确保构建成功，再触发一次文件变更。 |
| `insufficient permissions: tenant not allowed` | 租户无调试权限 | 与运维确认租户白名单，或在 Dev API 中开启 demo tenant。 |

调试建议：

- 开启调试日志：`export PX_DEBUG=true && export PX_GLOBAL_LOGLEVEL=debug`
- 保存 CLI 输出：`px-plugin dev --watch ... --logs-file /tmp/dev-watch.log`
- 结合 `docs/guides/cli/go-cli-troubleshooting.md` 获取更细的排查脚本。

---

## 8. 与 TypeScript CLI 的对比

| 特性 | Go CLI | TypeScript CLI | 备注 |
|------|-------|----------------|------|
| 运行时 | 单一二进制 | 需 Node.js | Go 版易于分发。 |
| 启动速度 | ✅ | ⚠️ | Go CLI 更快。 |
| 内存占用 | ✅ 45MB 左右 | ⚠️ 80MB+ | 适合资源受限环境。 |
| mTLS | ✅ | ✅ | 功能对齐。 |
| SSE 日志 | ✅ | ✅ | 两者格式一致。 |
| Session 持久化 | ✅ | ✅ | 审计结构一致。 |
| 插件扩展 | ⚠️（Go 插件） | ✅（JS Hook） | 若需自定义构建链，TS CLI 灵活度更高。 |
| 调试体验 | ⚠️（Go 工具链） | ✅（前端生态丰富） | 推荐两个 CLI 同时保留。 |

---

## 9. API 参考

### 9.1 会话模型
```json
{
  "id": "session-123",
  "pluginId": "com.powerx.demo",
  "version": "1.4.0",
  "entryPath": "/plugins/demo",
  "tenant": "demo-tenant",
  "status": "active",
  "metrics": {
    "reloadCount": 15,
    "avgReloadTime": 250,
    "successRate": 0.95
  }
}
```

### 9.2 Dev API 端点

| Method | Endpoint | 说明 |
|--------|----------|------|
| `POST` | `/internal/dev/plugins/register` | 注册调试会话，返回 `sessionId` 与 `reloadToken`。 |
| `POST` | `/internal/dev/plugins/reload` | 推送新的 bundle。 |
| `DELETE` | `/internal/dev/plugins/register/{sessionId}` | 终止会话。 |
| `GET` | `/internal/dev/hosts/sessions/{sessionId}/logs` | SSE 日志流。 |
| `POST` | `/internal/dev/plugins/rollback` | CLI 内部使用，触发回滚。 |

---

## 10. 最佳实践

1. **进入 Publish Hub 前**，至少完成一次 `px-plugin dev --watch` + `px-plugin doctor`，并保留输出文件。  
2. **提交工单** 时附上：CLI 日志、`.doctor/report.json`、`~/.px-plugin/audit/latest.log`。  
3. **定期清理缓存**：`rm -rf ~/.px-plugin/cache` 防止历史 bundle 占用磁盘。  
4. **组合调试命令**：若需要宿主模拟/沙箱验证，可与 `px-plugin host`、`px-plugin sandbox` 搭配使用。  
5. **监控资源**：`top` / `htop` 或 `PX_RESOURCE_*` 环境变量确保 CLI 不会影响 CI/CD 节点。  
6. **与 TS CLI 对齐**：涉及 Dev API 契约变更时，先更新 Go/TS 双端并在 `tmp/go-cli-dev-watch-bench` 跑基准测试。

---

## 11. 支持渠道

- 常见问题：`docs/guides/cli/go-cli-troubleshooting.md`
- 指标/Bench：`scripts/perf/go-cli-dev-watch-bench.sh`
- 工单：PowerX 内部 Issue Tracker，分类选择 “Publish Hub / Dev Watch”
- 紧急支持：Slack `#powerx-dev-hotload`

如需补充案例或翻译改进，请在仓库提交 PR 或 Issue。
