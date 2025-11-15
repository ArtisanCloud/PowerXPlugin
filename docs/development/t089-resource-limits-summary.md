# T089: Resource Limits - Summary

## 概述
本任务实现了全面的资源限制系统，用于控制和监控 CPU、内存、磁盘、网络和文件描述符等资源的使用。

## 核心组件

### 1. 资源监控器 (`limits.go`)

#### ResourceMonitor - 资源监控器
```go
type ResourceMonitor struct {
    limits    map[ResourceType]Limit
    usage     map[ResourceType]*atomic.Int64
    maxUsage  map[ResourceType]int64
    callbacks map[ResourceType][]func(usage int64, limit Limit)
    enabled   bool
}
```

**支持的资源类型**:
- **CPU** - CPU 使用率
- **Memory** - 内存使用量
- **Disk** - 磁盘使用量
- **Network** - 网络使用量
- **Files** - 文件描述符数量

**核心功能**:
- 设置资源限制
- 跟踪资源使用
- 计算使用百分比
- 阈值监控
- 回调通知
- 最大值记录

**使用示例**:
```go
// 创建监控器
rm := NewResourceMonitor()

// 设置内存限制
rm.SetLimit(Limit{
    Type:      Memory,
    Value:     500 * 1024 * 1024, // 500MB
    Unit:      "bytes",
    Threshold: 80, // 80%
})

// 添加阈值回调
rm.AddCallback(Memory, func(usage int64, limit Limit) {
    log.Printf("警告: 内存使用达到 %d%% (%d / %d bytes)",
        usage*100/limit.Value, usage, limit.Value)
})

// 记录使用
rm.AddUsage(Memory, 100*1024*1024) // 100MB

// 获取统计
stats := rm.GetStats()
```

#### Limit - 资源限制定义
```go
type Limit struct {
    Type      ResourceType `json:"type"`
    Value     int64        `json:"value"`
    Unit      string       `json:"unit"`
    Threshold float64      `json:"threshold,omitempty"`
}
```

**字段说明**:
- **Type** - 资源类型
- **Value** - 限制值
- **Unit** - 单位 (bytes, percent, count)
- **Threshold** - 阈值百分比 (0-100)

### 2. 内存限制器 (`limits.go`)

#### MemoryLimiter - 内存限制器
```go
type MemoryLimiter struct {
    limit           int64
    allocated       int64
    gcThreshold     int64
    onLimitReached  func(allocated int64)
}
```

**功能**:
- 限制内存分配
- 跟踪已分配内存
- 自动垃圾回收
- 超限回调

**使用示例**:
```go
// 创建内存限制器 (1GB 限制, 800MB 时 GC)
ml := NewMemoryLimiter(1024*1024*1024, 800*1024*1024)

ml.SetOnLimitReached(func(allocated int64) {
    log.Printf("内存使用达到限制: %d bytes", allocated)
})

// 分配内存
err := ml.Allocate(100 * 1024 * 1024) // 100MB
if err != nil {
    log.Fatalf("内存分配失败: %v", err)
}

// 检查使用情况
if ml.GetUsagePercentage() > 80 {
    log.Println("警告: 内存使用率超过 80%")
}

ml.Deallocate(50 * 1024 * 1024) // 释放 50MB
```

**默认配置**:
- 限制: 500MB
- GC 阈值: 80% (400MB)

### 3. 文件描述符限制器 (`limits.go`)

#### FileDescriptorLimiter - 文件描述符限制器
```go
type FileDescriptorLimiter struct {
    limit           int
    openCount       int
    onLimitReached  func(openCount int)
}
```

**功能**:
- 限制打开文件数
- 跟踪打开文件
- 获取/释放语义
- 超限回调

**使用示例**:
```go
// 创建文件描述符限制器
fdl := NewFileDescriptorLimiter(1024)

fdl.SetOnLimitReached(func(openCount int) {
    log.Printf("文件描述符使用达到限制: %d", openCount)
})

// 获取文件描述符
err := fdl.Acquire()
if err != nil {
    log.Fatalf("无法获取文件描述符: %v", err)
}

// 使用文件
defer fdl.Release()
```

**默认配置**:
- 限制: 1024 个文件描述符

## 4. CLI 集成

- `px-plugin dev` 默认启用资源守护：
  - 内存预算：100MB（可通过 `--max-memory-mb`、`PX_RESOURCE_MEMORY_MB` 或 `performance.memoryLimit` 调整）。
  - CPU 阈值：10%（可通过 `--max-cpu-percent`、`PX_RESOURCE_CPU_THRESHOLD` 或 `performance.cpuThreshold` 调整）。
  - 监控文件数：10,000（可通过 `--max-watch-files`、`PX_MAX_WATCH_FILES` 或 `watch.maxFiles` 调整）。
  - `--max-procs` 映射到 `runtime.GOMAXPROCS`，默认读取 `performance.maxConcurrency` 或 `PX_MAX_PROCS`。
- 触发阈值时，会在终端提示“Resource guard active”并跳过 reload；可在配置文件/环境变量中调大阈值，再运行 `px-plugin doctor`/`px-plugin dev --watch` 观察效果。

### 4. 速率限制器 (`limits.go`)

#### RateLimiter - 速率限制器
```go
type RateLimiter struct {
    limit        int64
    used         int64
    resetTime    time.Time
    window       time.Duration
    onLimitReached func(used int64)
}
```

**功能**:
- 限制操作速率
- 滑动时间窗口
- 自动重置
- 剩余额度查询

**使用示例**:
```go
// 创建速率限制器 (每秒 1000 次操作)
rl := NewRateLimiter(1000, time.Second)

// 检查是否允许
if rl.Allow() {
    performOperation()
}

// 获取剩余额度
remaining := rl.GetRemaining()
log.Printf("剩余 %d 次操作", remaining)

// 获取重置时间
resetTime := rl.GetResetTime()
log.Printf("窗口重置时间: %s", resetTime.Format("15:04:05"))
```

**默认配置**:
- 限制: 1000 操作/窗口
- 窗口: 1 秒

### 5. CPU 分析器 (`limits.go`)

#### CPUProfiler - CPU 使用率分析器
```go
type CPUProfiler struct {
    enabled       bool
    samples       []CPUSample
    maxSamples    int
    onHighUsage   func(usage float64)
}

type CPUSample struct {
    Timestamp time.Time `json:"timestamp"`
    Usage     float64   `json:"usage"`
    Delta     float64   `json:"delta"`
}
```

**功能**:
- 采样 CPU 使用率
- 计算平均值
- 高使用率检测
- 启用/禁用控制

**使用示例**:
```go
// 创建 CPU 分析器
cp := NewCPUProfiler(100) // 保存 100 个样本

cp.SetOnHighUsage(func(usage float64) {
    log.Printf("警告: CPU 使用率达到 %.1f%%", usage)
})

// 定期采样
ticker := time.NewTicker(5 * time.Second)
defer ticker.Stop()

go func() {
    for range ticker.C {
        cp.Sample()
    }
}()

// 获取使用率
avg := cp.GetAverageUsage()
latest := cp.GetLatestUsage()

log.Printf("平均 CPU: %.1f%%, 最新: %.1f%%", avg, latest)
```

**默认配置**:
- 最大样本: 100
- 高使用阈值: 80%

### 6. 资源跟踪器 (`limits.go`)

#### ResourceTracker - 资源使用跟踪器
```go
type ResourceTracker struct {
    metrics          map[ResourceType][]MetricPoint
    maxPoints        int
    samplingInterval time.Duration
    ticker           *time.Ticker
    monitor          *ResourceMonitor
}

type MetricPoint struct {
    Timestamp time.Time         `json:"timestamp"`
    Type      ResourceType      `json:"type"`
    Value     int64             `json:"value"`
    Unit      string            `json:"unit"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}
```

**功能**:
- 自动采样
- 指标存储
- 时间序列数据
- 持久化历史

**使用示例**:
```go
// 创建资源跟踪器
rm := NewResourceMonitor()
rm.SetLimit(Limit{Type: Memory, Value: 1024*1024*1024, Unit: "bytes"})

rt := NewResourceTracker(rm, 5*time.Second) // 每 5 秒采样

// 开始跟踪
rt.Start()

// 获取指标
metrics := rt.GetMetrics(Memory)
for _, point := range metrics {
    log.Printf("[%s] 内存: %d %s",
        point.Timestamp.Format("15:04:05"),
        point.Value, point.Unit)
}

// 停止跟踪
rt.Stop()
```

**默认配置**:
- 采样间隔: 5 秒
- 最大点数: 1000

### 7. 资源错误 (`limits.go`)

#### ResourceError - 资源错误
```go
type ResourceError struct {
    Type      ResourceType `json:"type"`
    Value     int64        `json:"value"`
    Limit     int64        `json:"limit"`
    Message   string       `json:"message"`
    Timestamp time.Time    `json:"timestamp"`
}
```

**功能**:
- 资源超限错误
- 错误类型识别
- 时间戳记录

**使用示例**:
```go
err := ml.Allocate(2000) // 超过限制

if resourceErr, ok := err.(*ResourceError); ok {
    log.Printf("资源 %s 超限: 使用 %d, 限制 %d",
        resourceErr.Type,
        resourceErr.Value,
        resourceErr.Limit)
}
```

## 测试覆盖 (`tools/cli/internal/resources/limits_test.go`)

### 监控器测试
- ✅ `TestResourceMonitor_SetLimit` - 设置限制
- ✅ `TestResourceMonitor_AddUsage` - 添加使用量
- ✅ `TestResourceMonitor_GetStats` - 获取统计

### 内存限制测试
- ✅ `TestMemoryLimiter_Allocate` - 内存分配
- ✅ `TestMemoryLimiter_Deallocate` - 内存释放
- ✅ `TestMemoryLimiter_GetUsagePercentage` - 使用率

### 文件描述符测试
- ✅ `TestFileDescriptorLimiter_Acquire` - 获取文件描述符
- ✅ `TestFileDescriptorLimiter_Release` - 释放文件描述符

### 速率限制测试
- ✅ `TestRateLimiter_Allow` - 速率限制
- ✅ `TestRateLimiter_WindowReset` - 窗口重置
- ✅ `TestRateLimiter_GetResetTime` - 获取重置时间

### CPU 分析测试
- ✅ `TestCPUProfiler_Sample` - 采样
- ✅ `TestCPUProfiler_EnableDisable` - 启用/禁用

### 资源跟踪测试
- ✅ `TestResourceTracker_StartStop` - 启动/停止
- ✅ `TestResourceTracker_GetAllMetrics` - 获取所有指标

## 使用场景

### 1. 开发模式资源限制
```go
// 为开发模式设置资源限制
rm := NewResourceMonitor()

// 内存限制 500MB
rm.SetLimit(Limit{
    Type:      Memory,
    Value:     500 * 1024 * 1024,
    Unit:      "bytes",
    Threshold: 80,
})

// CPU 限制 80%
rm.SetLimit(Limit{
    Type:      CPU,
    Value:     80,
    Unit:      "percent",
    Threshold: 90,
})

// 文件描述符限制 1024
rm.SetLimit(Limit{
    Type:      Files,
    Value:     1024,
    Unit:      "count",
    Threshold: 90,
})
```

### 2. 构建资源控制
```go
// 构建时的内存限制
ml := NewMemoryLimiter(2*1024*1024*1024, 1600*1024*1024) // 2GB, GC at 1.6GB

ml.SetOnLimitReached(func(allocated int64) {
    log.Fatal("构建失败: 内存使用超限")
})

// 构建过程中分配内存
for _, file := range files {
    size := calculateSize(file)
    if err := ml.Allocate(size); err != nil {
        log.Fatalf("构建失败: %v", err)
    }
    processFile(file)
}
```

### 3. API 速率限制
```go
// 限制 API 调用的速率
rl := NewRateLimiter(10, time.Second) // 每秒 10 次

func makeAPICall() error {
    if !rl.Allow() {
        return fmt.Errorf("API 调用频率超限")
    }

    return callExternalAPI()
}
```

### 4. 实时资源监控
```go
// 启动资源监控
rm := NewResourceMonitor()
rt := NewResourceTracker(rm, 5*time.Second)
rt.Start()

// 定期检查
ticker := time.NewTicker(10 * time.Second)
go func() {
    for range ticker.C {
        stats := rm.GetStats()

        for resourceType, data := range stats {
            usage := data.(map[string]interface{})["usage"].(int64)
            limit := data.(map[string]interface{})["limit"].(int64)
            percentage := data.(map[string]interface{})["percentage"].(float64)

            log.Printf("%s: %d / %d (%.1f%%)",
                resourceType, usage, limit, percentage)
        }
    }
}()
```

### 5. 文件操作保护
```go
// 限制同时打开的文件数
fdl := NewFileDescriptorLimiter(100)

func processFiles(files []string) error {
    for _, file := range files {
        if err := fdl.Acquire(); err != nil {
            return fmt.Errorf("文件描述符不足: %w", err)
        }
        defer fdl.Release()

        data, err := os.ReadFile(file)
        if err != nil {
            return err
        }
        process(data)
    }
    return nil
}
```

## 性能特性

### 1. 低开销
- 原子操作计数器 (无锁)
- 延迟评估
- 最小内存占用

### 2. 可扩展
- 支持多种资源类型
- 可插拔回调
- 可配置阈值

### 3. 线程安全
- 互斥锁保护
- 无数据竞争
- 并发安全

## 默认配置

| 组件 | 参数 | 默认值 | 说明 |
|------|------|--------|------|
| MemoryLimiter | limit | 500MB | 内存限制 |
| MemoryLimiter | gcThreshold | 80% | GC 触发阈值 |
| FileDescriptorLimiter | limit | 1024 | 文件描述符限制 |
| RateLimiter | limit | 1000 | 操作次数/窗口 |
| RateLimiter | window | 1s | 时间窗口 |
| CPUProfiler | maxSamples | 100 | 最大样本数 |
| CPUProfiler | highThreshold | 80% | 高使用率阈值 |
| ResourceTracker | interval | 5s | 采样间隔 |
| ResourceTracker | maxPoints | 1000 | 最大数据点 |

## 与 TypeScript 版本对比

| 特性 | Go 版本 | TypeScript 版本 |
|------|---------|-----------------|
| 内存限制 | ✅ 精确控制 | ⚠️ 粗略限制 |
| 文件描述符 | ✅ 系统级 | ⚠️ 需手动管理 |
| 速率限制 | ✅ 精准窗口 | ⚠️ 定时器实现 |
| CPU 监控 | ✅ 系统调用 | ⚠️ 估算 |
| 资源跟踪 | ✅ 时间序列 | ⚠️ 简单记录 |
| 原子操作 | ✅ 无锁 | ⚠️ 锁实现 |
| 性能 | ✅ 原生性能 | ⚠️ 运行时开销 |

## 文件清单

### 新建文件
- `tools/cli/internal/resources/limits.go` - 资源限制实现
- `tools/cli/internal/resources/limits_test.go` - 资源限制测试

### 修改文件
- 现有组件可集成此资源限制系统

## 成功标准

- ✅ 资源监控器 (5 种资源类型)
- ✅ 内存限制器 (分配/释放/GC)
- ✅ 文件描述符限制 (获取/释放)
- ✅ 速率限制器 (滑动窗口)
- ✅ CPU 分析器 (采样/统计)
- ✅ 资源跟踪器 (时间序列)
- ✅ 资源错误 (类型安全)
- ✅ 回调机制 (阈值通知)
- ✅ 全面的测试覆盖 (15+ 测试)
- ✅ 文档和示例

## 后续任务

T089 完成后，可以继续：
- **T090** - 配置管理
- **T091** - 端到端测试
- **T092** - 性能验证

## 结论

T089 资源限制系统为 Go CLI 提供了企业级的资源管理：
1. **全面控制** - CPU、内存、磁盘、网络、文件
2. **实时监控** - 阈值检测、回调通知
3. **自动保护** - 速率限制、内存管理
4. **历史跟踪** - 时间序列数据、性能分析

这确保了 Go CLI 不会超出系统资源限制！📊
