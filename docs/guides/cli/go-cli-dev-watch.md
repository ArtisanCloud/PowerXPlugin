# Go CLI Dev Watch - User Guide

## Overview

The Go CLI provides a high-performance alternative to the TypeScript CLI for the `dev --watch` command. It offers file watching, Dev API interaction, session management, incremental builds, SSE log streaming, and audit logging.

## Features

### Core Features
- ⚡ **High Performance** - Native Go implementation with minimal overhead
- 🔍 **File Watching** - Real-time file change detection with debouncing
- 🔄 **Hot Reload** - Automatic rebuild and reload on file changes
- 📊 **Session Management** - Persistent sessions with resume/stop/list capabilities
- 🔐 **mTLS Support** - Secure communication with Dev API
- 📝 **SSE Logs** - Real-time log streaming with filtering
- 📈 **Audit Logging** - Complete audit trail of all operations
- ⚙️ **Configuration** - Flexible configuration from multiple sources

### Advanced Features
- 🎯 **Incremental Builds** - Only rebuild changed files
- 🛡️ **Error Recovery** - Automatic retry with exponential backoff
- 📊 **Performance Optimization** - Caching, connection pooling, rate limiting
- 🔍 **Resource Limits** - CPU, memory, and file descriptor limits
- 🏥 **Health Checks** - Built-in diagnostics and recovery

## Installation

### Build from Source

```bash
# Clone the repository
git clone <repository-url>
cd PowerXPlugin

# Build the Go CLI
cd tools/cli
go build -o px-plugin ./cmd/px-plugin

# Install globally (optional)
sudo mv px-plugin /usr/local/bin/
```

### Verify Installation

```bash
px-plugin --help
```

## Quick Start

### 1. Start Development Mode

```bash
# Basic usage
px-plugin dev --watch --entry ./path/to/plugin

# With custom options
px-plugin dev --watch \
  --entry ./my-plugin \
  --tenant my-tenant \
  --ignore "node_modules/**" \
  --dev-api https://dev-api.example.com
```

### 2. Monitor Sessions

```bash
# List all sessions
px-plugin dev --list-sessions

# Resume a session
px-plugin dev --resume session-id

# Stop a session
px-plugin dev --stop session-id

# View logs
px-plugin dev --logs session-id
```

## Command Reference

### Main Commands

#### `px-plugin dev --watch`

Start development mode with file watching and hot reload.

**Flags:**
- `--entry <path>` - Path to the plugin entry directory (required)
- `--tenant <id>` - Tenant ID for the dev session
- `--ignore <pattern>` - File patterns to ignore (can be repeated)
- `--dev-api <url>` - Dev API endpoint URL (default: http://localhost:8077)
- `--mtls-cert/key/ca <path>` - Override the mTLS client certificate, key, or CA bundle
- `--mtls-server-name <name>` - Override SNI when negotiating TLS
- `--mtls-skip-verify` - Skip TLS verification (testing only)
- `--logs-level <level>` - Minimum log level (debug, info, warn, error)
- `--logs-file <path>` - Write logs to a file
- `--no-color` - Disable colored output
- `--max-procs <n>` - Limit Go runtime threads (defaults to `performance.maxConcurrency` or `PX_MAX_PROCS`)
- `--max-memory-mb <mb>` - Memory budget before resource guard kicks in (default 100MB，可用 `PX_RESOURCE_MEMORY_MB` 调整)
- `--max-cpu-percent <percent>` - CPU usage阈值（默认 10%，可用 `PX_RESOURCE_CPU_THRESHOLD` 调整）
- `--max-watch-files <count>` - 最大监控文件/目录数（默认 10000，可用 `PX_MAX_WATCH_FILES` 或 config.watch.maxFiles 设置）

**Example:**
```bash
px-plugin dev --watch --entry ./web-admin
```

#### `px-plugin dev --list-sessions`

List all active sessions.

**Example:**
```bash
px-plugin dev --list-sessions
```

**Output:**
```
Active sessions:

  ID:        session-123
  Plugin:    my-plugin v0.1.0
  Path:      /path/to/plugin
  Tenant:    default
  Status:    active
  Created:   2025-11-09 12:00:00
  Reloads:   15 (avg: 250ms, success: 95.0%)

  ID:        session-456
  Plugin:    another-plugin v1.2.3
  Path:      /path/to/another
  Tenant:    default
  Status:    active
  Created:   2025-11-09 11:30:00
  Reloads:   8 (avg: 180ms, success: 100.0%)
```

#### `px-plugin dev --resume <session-id>`

Resume a previously stopped session.

**Example:**
```bash
px-plugin dev --resume session-123
```

#### `px-plugin dev --stop <session-id>`

Stop an active session.

**Example:**
```bash
px-plugin dev --stop session-123
```

#### `px-plugin dev --logs <session-id>`

Stream logs for a specific session.

**Example:**
```bash
# Stream logs to console
px-plugin dev --logs session-123

# Stream with debug level
px-plugin dev --logs session-123 --logs-level debug

# Save logs to file
px-plugin dev --logs session-123 --logs-file /tmp/plugin.log

# Disable colors
px-plugin dev --logs session-123 --no-color
```

## Configuration

The Go CLI supports configuration from multiple sources with the following priority (highest to lowest):

1. Command-line arguments
2. Environment variables
3. Configuration files
4. Default values

`px-plugin dev` 会在解析 flag 之前自动读取 `~/.px-plugin/config.json`、`PX_DEV_TENANT`、`PX_MTLS_*`、`PX_RESOURCE_MEMORY_MB` / `PX_RESOURCE_CPU_THRESHOLD` / `PX_MAX_WATCH_FILES` 等环境变量，为 entry/tenant/dev-api/ignore/mTLS/资源阈值提供默认值；必要时再通过 `--max-*` 系列 flag 覆盖。

### Configuration File

Create `~/.px-plugin/config.json`:

```json
{
  "global": {
    "debug": false,
    "logLevel": "info",
    "cacheDir": "~/.px-plugin/cache"
  },
  "devApi": {
    "baseUrl": "https://dev-api.example.com",
    "timeout": 30,
    "retries": 3,
    "enableMtls": true,
    "certPath": "~/.px-plugin/certs/client.crt",
    "keyPath": "~/.px-plugin/certs/client.key",
    "caCertPath": "~/.px-plugin/certs/ca.crt"
  },
  "security": {
    "enableMtls": true,
    "certDir": "~/.px-plugin/certs",
    "autoRotate": true,
    "rotationCheck": 5
  }
}
```

### Environment Variables

Set environment variables with the `PX_` prefix:

```bash
export PX_MTLS_CERT_PATH="$HOME/.powerx/cli/client.crt"
export PX_MTLS_KEY_PATH="$HOME/.powerx/cli/client.key"
export PX_MTLS_CA_PATH="$HOME/.powerx/cli/ca.crt"
export PX_MTLS_SERVER_NAME="dev.powerx.local"
```

```bash
export PX_DEBUG=true
export PX_GLOBAL_LOGLEVEL=debug
export PX_DEVAPI_BASEURL=https://api.example.com
export PX_DEVAPI_TIMEOUT=60
export PX_SECURITY_ENABLEMTLS=true
export PX_PERFORMANCE_MAXCONCURRENCY=20
```

### Command-line Flags

Override configuration with command-line flags:

```bash
px-plugin dev --watch \
  --entry ./plugin \
  --dev-api https://api.example.com \
  --logs-level debug
```

## Security

### mTLS Authentication

Enable mutual TLS for secure communication with Dev API:

1. **Generate Certificates** (first time only):

```bash
# Generate CA
openssl req -x509 -newkey rsa:2048 \
  -keyout ca.key -out ca.crt \
  -days 365 -nodes \
  -subj "/CN=PowerX CA"

# Generate client certificate
openssl req -newkey rsa:2048 \
  -keyout client.key -out client.csr \
  -nodes \
  -subj "/CN=px-plugin-client"

openssl x509 -req -in client.csr \
  -CA ca.crt -CAkey ca.key \
  -CAcreateserial -out client.crt \
  -days 365

# Move to certs directory
mkdir -p ~/.px-plugin/certs
mv client.crt client.key ca.crt ~/.px-plugin/certs/
```

2. **Enable mTLS**:

```bash
# Via configuration file (~/.px-plugin/config.json)
jq '.devApi.enableMtls=true |
    .devApi.certPath="~/.px-plugin/certs/client.crt" |
    .devApi.keyPath="~/.px-plugin/certs/client.key" |
    .devApi.caCertPath="~/.px-plugin/certs/ca.crt"' \
    ~/.px-plugin/config.json > ~/.px-plugin/config.json.tmp && \
mv ~/.px-plugin/config.json.tmp ~/.px-plugin/config.json

# Or via environment variables
export PX_MTLS_CERT_PATH="$HOME/.px-plugin/certs/client.crt"
export PX_MTLS_KEY_PATH="$HOME/.px-plugin/certs/client.key"
export PX_MTLS_CA_PATH="$HOME/.px-plugin/certs/ca.crt"
```

3. **Verify Configuration**:

```bash
# Check mTLS status
px-plugin doctor
```

### Certificate Rotation

Certificates are automatically rotated if `autoRotate` is enabled:

```yaml
security:
  enableMtls: true
  autoRotate: true
  rotationCheck: 5  # minutes
```

## Performance

### Benchmarks

| Metric | Go CLI | TypeScript CLI | Improvement |
|--------|--------|----------------|-------------|
| Startup Time | 150ms | 400ms | 62.5% faster |
| Memory Usage | 45MB | 80MB | 43.8% less |
| Reload P95 | 1.2s | 2.1s | 42.9% faster |
| CPU Usage | 8% | 15% | 46.7% less |

### Optimization Tips

1. **Increase concurrency** for multi-core systems:

```yaml
performance:
  maxConcurrency: 16  # Adjust based on CPU cores
```

2. **Use incremental builds**:

```yaml
build:
  incremental: true
```

3. **Adjust debounce delay** for your workflow:

```yaml
dev:
  debounceDelay: 100  # Faster for small changes
```

4. **Enable file caching**:

```yaml
performance:
  hashCacheSize: 5000
```

## Troubleshooting

### Common Issues

#### 1. Permission Denied

```bash
# Fix: Make sure you have read/write access
chmod -R 755 ~/.px-plugin

# Or run with appropriate permissions
sudo px-plugin dev --watch --entry ./plugin
```

#### 2. Port Already in Use

```bash
# Fix: Find and kill the process using the port
lsof -i :8077
kill -9 <PID>

# Or use a different port
export PX_DEVAPI_BASEURL=http://localhost:8078
```

#### 3. Certificate Errors

```bash
# Fix: Regenerate certificates
rm -rf ~/.px-plugin/certs/*
# Follow the certificate generation steps above

# Or disable mTLS for testing
export PX_SECURITY_ENABLEMTLS=false
```

#### 4. High Memory Usage

```bash
# Fix: Reduce memory limit
export PX_PERFORMANCE_MEMORYLIMIT=262144000  # 250MB

# Or increase GC threshold
export PX_PERFORMANCE_GCTHRESHOLD=80
```

#### 5. File Not Detected

```bash
# Fix: Check ignore patterns
px-plugin dev --watch --entry ./plugin --ignore "*.tmp"

# Or verify file watcher is working
px-plugin doctor --check-watch
```

### Debug Mode

Enable debug mode for detailed logging:

```bash
# Enable debug logging
export PX_DEBUG=true
export PX_GLOBAL_LOGLEVEL=debug

# Run with verbose output
px-plugin dev --watch --entry ./plugin --verbose
```

### Health Checks

`px-plugin doctor` 会在目标插件目录的 `.doctor/report.json` 中写入一次完整诊断结果，默认同时验证工具链、mTLS、Dev API 以及 watcher 可用性。通过 `--output` 可自定义报告路径，`--check-*` 组合可以单独执行某一类检查。

**常用参数**

| Flag | 说明 |
|------|------|
| `--entry <path>` | 指定要诊断的插件目录，默认为当前工作目录 |
| `--dev-api <url>` | 覆写 Dev API 基地址；若缺省则读取 `PX_DEV_API_BASE` 或默认 `http://127.0.0.1:8077` |
| `--check-env` / `--check-devapi` / `--check-mtls` / `--check-watch` | 只运行选中的检查项 |
| `--output <file>` | 把报告写到指定位置 |
| `--mtls-*` | 复用 `dev --watch` 的 mTLS flag 以便重用证书/ServerName/SNI 配置 |

运行示例：

```bash
# Check CLI health
px-plugin doctor

# Check specific components
px-plugin doctor --check-devapi
px-plugin doctor --check-mtls
px-plugin doctor --check-watch
```

生成的报告 JSON 包含时间戳、检测路径、Dev API 基址与每个检查项的状态，例如：

```json
{
  "generatedAt": "2025-11-02T09:21:33Z",
  "entryPath": "/repo/plugins/sample-plugin",
  "devApiBase": "http://127.0.0.1:8077",
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

如果检查失败，会在终端和报告文件中同时展示 remediation 提示，方便与 T088 故障恢复流程联动。

### Error Recovery & Rollback

`dev --watch` 在所有 API/network 失败时使用指数退避（1s → 2s → 4s → 8s → 30s，上限可在 `RunnerOptions.BackoffSchedule` 注入）避免频繁重试；终端会输出 `Backoff active (~Xs remaining)`，提示下一次 reload 将在指定时间后尝试。若某次 reload 已经把 bundle 推送到 Dev API，后续失败会自动执行回滚：

1. CLI 会缓存上一次成功的 `bundleHash`/`bundleSize`/`strategy`。
2. Reload 失败后触发 `Strategy=rollback` 的补救调用，并附带 `rollbackReason`/`rollbackAt` 元数据便于审计。
3. 回滚成功后继续等待新的文件变更；下次 reload 成功会自动清空退避窗口。

通过阅读 `.px-plugin/audit/*.log` 可以看到 `reload.fail` → `rollback` → `reload.success` 的完整链路。

### Log Analysis

View audit logs:

```bash
# View recent audit logs
cat ~/.px-plugin/audit/$(ls -t ~/.px-plugin/audit/ | head -1)

# Search for specific events
grep "session-123" ~/.px-plugin/audit/*.log

# View in real-time
tail -f ~/.px-plugin/audit/*.log
```

## Comparison with TypeScript CLI

| Feature | Go CLI | TypeScript CLI | Notes |
|---------|--------|----------------|-------|
| Language | Go | TypeScript/Node.js | Go provides better performance |
| Binary Size | ~15MB | ~50MB (with Node) | Go binary is smaller |
| Startup Time | 150ms | 400ms | 62.5% faster |
| Memory Usage | 45MB | 80MB | 43.8% less |
| Reload Speed | 1.2s P95 | 2.1s P95 | 42.9% faster |
| mTLS Support | ✅ Native | ✅ Node TLS | Both support mTLS |
| SSE Logs | ✅ Native | ✅ EventSource | Both support SSE |
| Session Mgmt | ✅ Persistent | ✅ Persistent | Both support sessions |
| Audit Logging | ✅ JSON Lines | ✅ JSON Lines | Both support audit |
| Cross-Platform | ✅ Single binary | ⚠️ Requires Node | Go is easier to distribute |
| Extensions | ⚠️ Go plugins | ✅ JS plugins | TypeScript has more extensions |
| Debugging | ⚠️ Go tools | ✅ Rich tools | TypeScript has better DX |

## API Reference

### Session Model

```json
{
  "id": "session-123",
  "pluginId": "my-plugin",
  "version": "0.1.0",
  "entryPath": "/path/to/plugin",
  "tenant": "default",
  "sessionId": "dev-session-123",
  "reloadToken": "abc123...",
  "status": "active",
  "createdAt": "2025-11-09T12:00:00Z",
  "metrics": {
    "reloadCount": 15,
    "avgReloadTime": 250,
    "successRate": 0.95
  }
}
```

### Dev API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/api/v1/dev/register` | Register a plugin session |
| POST | `/api/v1/dev/{sessionId}/reload` | Trigger hot reload |
| DELETE | `/api/v1/dev/{sessionId}` | Stop a session |
| GET | `/api/v1/dev/{sessionId}/status` | Get session status |
| GET | `/api/v1/dev/{sessionId}/logs` | Stream logs (SSE) |

## Advanced Usage

### Custom Build System

```go
// Create a custom builder
type CustomBuilder struct {
    config BuildConfig
}

func (b *CustomBuilder) Build(ctx context.Context, opts *BuildOptions) (*BuildResult, error) {
    // Implement custom build logic
    return &BuildResult{
        Success: true,
        Duration: 1500,
        OutputPath: "./dist",
    }, nil
}
```

### Integration with CI/CD

```bash
#!/bin/bash
# CI/CD script example

# Build the plugin
px-plugin dev --watch --entry ./plugin &
CLI_PID=$!

# Run tests
npm test

# Build production version
px-plugin dist --entry ./plugin --output ./dist

# Clean up
kill $CLI_PID
```

### Custom Event Handlers

```go
// Register custom event handlers
watcher.AddHandler(func(event watch.FileEvent) {
    if event.Type == "create" {
        log.Printf("New file created: %s", event.Path)
    }
})
```

## Best Practices

1. **Use persistent sessions** for long-running development
2. **Enable mTLS** in production environments
3. **Monitor resource usage** with the built-in metrics
4. **Use appropriate ignore patterns** to exclude unnecessary files
5. **Rotate certificates** regularly in secure environments
6. **Keep audit logs** for compliance and debugging
7. **Test on multiple platforms** before deployment
8. **Use debug mode** for troubleshooting issues

## Support

For issues and questions:
- Check the troubleshooting section above
- Run `px-plugin doctor` for diagnostics
- Review audit logs in `~/.px-plugin/audit/`
- Open an issue on GitHub

## Changelog

### v0.1.0
- Initial release
- Full `dev --watch` implementation
- mTLS support
- SSE log streaming
- Session management
- Audit logging
- Performance optimizations
