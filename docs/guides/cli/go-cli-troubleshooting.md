# Go CLI dev --watch Troubleshooting & FAQ

## 1. 环境自检速查

1. **基础命令**  
   ```bash
   go build -o ./bin/px-plugin ./tools/cli/cmd/px-plugin
   ./bin/px-plugin --help
   ./bin/px-plugin dev --help
   ```
   若 `--help` 输出异常，优先确认 `GO111MODULE=on`、`go env GOPATH`、`go work sync` 状态。

2. **Doctor 报告**  
   ```bash
   ./bin/px-plugin doctor --entry <plugin-dir>
   cat <plugin-dir>/.doctor/report.json
   ```
   - `Toolchain` 失败：升级 Go ≥1.24、Node ≥18。
   - `mTLS` 警告：检查 `PX_MTLS_CERT_PATH/KEY_PATH/CA_PATH` 或 `~/.px-plugin/certs/`。
   - `Dev API` 失败：确认 `make devapi` 已启动，或使用 `--dev-api` 指向实际环境。

3. **SSE 日志**  
   ```bash
   ./bin/px-plugin dev --logs <session-id> \
     --dev-api http://127.0.0.1:8077/api/v1 \
     --logs-level debug
   ```
   若连接失败，请确认 `sessionId` 来自 `.px-plugin/sessions/*.json`，且 Dev API `GET /internal/dev/plugins/{session}` 可访问。

## 2. 常见错误对照

| 症状 | 排查步骤 | 参考文档 |
|------|----------|----------|
| `reload failed: tls handshake timeout` | 检查 `px auth configure` 产出的证书是否过期；可在 `docs/guides/publish/go-cli-dev-watch.md#health-checks` 找到证书更新流程 | `docs/guides/publish/go-cli-dev-watch.md` |
| `session register failed: 401` | `PX_DEV_API_BASE` 指向生产但未配置 `PX_MTLS_*`；在 `~/.px-plugin/config.json` 补上 mTLS | `docs/guides/quickstart.md#dev-api-热更新与-doctor-诊断` |
| `fsnotify watcher failed` | macOS 需允许终端“完全磁盘访问”，Windows 建议排除杀毒对 `px-plugin` 的阻断；跨平台脚本可用于验证差异 | `scripts/test/cross-platform-test.sh` |
| `watch limit exceeded (10000)` | 使用 `--max-watch-files` 或 `PX_MAX_WATCH_FILES` 放宽限制；也可在 `~/.px-plugin/config.json` 的 `watch.maxFiles` 中配置 | `docs/guides/publish/go-cli-dev-watch.md` |
| `rollback missing bundle` | 表示首次 reload 尚未成功；可运行 `px-plugin dev --watch` 再触发一次构建，使回滚缓存就绪 | `docs/guides/publish/go-cli-dev-watch.md#error-recovery--rollback` |

## 3. 跨平台自检脚本

`bash scripts/test/cross-platform-test.sh` 将针对 `linux/amd64`、`darwin/amd64`、`darwin/arm64`、`windows/amd64` 执行交叉构建与基本命令检测：

```bash
cd tools/cli
bash ../scripts/test/cross-platform-test.sh
```

输出会写入 `tmp/cross-platform-test/cross-platform-report.md`，并依据宿主平台自动跳过不可执行的二进制（标记为 `BUILD_ONLY`）。建议在 CI 中分别于 Linux/Windows 运行，以覆盖路径分隔符、证书路径等差异。

## 4. FAQ

1. **如何切换 Dev API**  
   - 临时指定：`px-plugin dev --watch --dev-api http://10.0.0.8:8077/api/v1`  
   - 全局配置：在 `~/.px-plugin/config.json` 写入 `{"devApi":{"baseUrl":"https://dev.powerx.example"}}`

2. **能否禁用 Telemetry？**  
   - 支持：`px-plugin dev --watch --no-telemetry` 或设置 `PX_PLUGIN_NO_TELEMETRY=true`。  
   - 仍会记录审计日志（本地 `.px-plugin/audit`），符合法规要求。

3. **如何在离线机器调试？**  
   - 运行 `px-plugin dev --watch --dev-api http://localhost:8077/api/v1 --mtls-skip-verify` 与自签证书。  
   - 若需完全离线，可使用 mock Dev API（`tools/cli/internal/devapi/mock_api.go`）并在测试中注入。

4. **跨平台注意事项**  
   - Windows：路径建议使用双引号与正斜杠 (`--entry "C:/workspace/plugin"`)，必要时在 PowerShell 中将 `PX_MTLS_*` 转为绝对路径。  
   - Linux：若使用 musl/Alpine，需设置 `CGO_ENABLED=0` 构建 `px-plugin`。

## 5. 相关参考

- `docs/guides/quickstart.md#dev-api-热更新与-doctor-诊断`
- `docs/guides/publish/go-cli-dev-watch.md`
- `docs/development/t095-cross-platform-summary.md`
- `scripts/test/cross-platform-test.sh`
