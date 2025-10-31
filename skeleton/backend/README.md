# Skeleton Backend

该目录提供最小可运行的 PowerX 插件后端示例：

1. 确保已在仓库根目录执行 `go work sync`。
2. 切换到本目录并运行：

   ```bash
   go run ./cmd/plugin
   ```

3. 验证接口：

   ```bash
   curl http://localhost:8078/api/v1/ping
   ```

示例仅包含 `GET /api/v1/ping`，实际项目可继续在 `internal/` 下扩展业务逻辑。
