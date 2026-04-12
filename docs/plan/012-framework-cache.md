# PowerXPlugin Cache Bridge 接入指引

本指引描述插件在宿主模式下如何使用 PowerX 底座统一缓存能力，
并在底座不可用时自动降级到本地实现。

## 1. 目标
- 插件使用统一 Cache 接口，不直接依赖 Redis 细节。
- 宿主模式下由 PowerX 注入配置并管理连接。
- 支持 `local | corex | dual` 三种模式与自动降级。

## 2. 配置位置
- **宿主模式**：PowerX 注入 `host-values.yaml` / `config.yaml` 的 `cache` 块。
- **Standalone 模式**：插件自带 `skeleton/backend/etc/config.yaml`。
- **不在 plugin.yaml 配置**缓存地址。

## 3. Cache Bridge
### 模式
- `local`：使用本地内存缓存
- `corex`：调用 PowerX 底座缓存驱动（默认 Redis）
- `dual`：双写/双读，便于灰度与回滚

### 配置示例
```yaml
cache:
  driver: redis
  redis_url: "redis://:password@127.0.0.1:6379/3"
  prefix: "powerx:{tenant_uuid}"
  default_ttl: 10m
```

### 使用示例（伪代码）
```go
cache.Set(ctx, "kb:chunks:123", data, 10*time.Minute)
val, ok := cache.Get(ctx, "kb:chunks:123")
```

## 4. 运行时注入与降级
- 宿主不可用时，Bridge 自动降级为 `local`（可配置）。
- 推荐日志：记录降级原因与恢复时间。

## 5. 认证与租户
- 宿主模式下由 PowerX 统一注入租户上下文。
- 业务 key 建议自动带 `tenant_uuid` 前缀。
