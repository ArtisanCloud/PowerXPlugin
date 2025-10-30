# Powerx Starter Plugin Backend

该目录包含 `com.powerx.starter` 插件的后端示例：

1. 在仓库根目录执行 `go work sync`（如适用），并保证 `github.com/powerx-plugin/framework` 可被下载或通过 replace 指向本地。
2. 切换到本目录并运行：

   ```bash
   go run ./cmd/plugin
   ```

3. 验证接口：

   ```bash
   curl http://localhost:8077/api/v1/ping
   ```

更多路由与业务逻辑可在 `internal/` 目录扩展。
