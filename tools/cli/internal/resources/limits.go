package resources

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ResourceType represents the type of resource
type ResourceType string

const (
	CPU     ResourceType = "cpu"
	Memory  ResourceType = "memory"
	Disk    ResourceType = "disk"
	Network ResourceType = "network"
	Files   ResourceType = "files"
)

// Limit represents a resource limit
type Limit struct {
	Type      ResourceType `json:"type"`
	Value     int64        `json:"value"`
	Unit      string       `json:"unit"`
	Threshold float64      `json:"threshold,omitempty"`
}

// ResourceMonitor monitors resource usage
type ResourceMonitor struct {
	mu        sync.RWMutex
	limits    map[ResourceType]Limit
	usage     map[ResourceType]*atomic.Int64
	maxUsage  map[ResourceType]int64
	callbacks map[ResourceType][]func(usage int64, limit Limit)
	enabled   bool
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor() *ResourceMonitor {
	return &ResourceMonitor{
		limits:    make(map[ResourceType]Limit),
		usage:     make(map[ResourceType]*atomic.Int64),
		maxUsage:  make(map[ResourceType]int64),
		callbacks: make(map[ResourceType][]func(usage int64, limit Limit)),
		enabled:   true,
	}
}

// SetLimit sets a resource limit
func (rm *ResourceMonitor) SetLimit(limit Limit) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.limits[limit.Type] = limit
	rm.usage[limit.Type] = &atomic.Int64{}
	rm.usage[limit.Type].Store(0)
	rm.maxUsage[limit.Type] = 0
}

// GetLimit gets a resource limit
func (rm *ResourceMonitor) GetLimit(resourceType ResourceType) (Limit, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	limit, ok := rm.limits[resourceType]
	return limit, ok
}

// AddUsage adds to the usage of a resource
func (rm *ResourceMonitor) AddUsage(resourceType ResourceType, delta int64) {
	rm.mu.RLock()
	limit, hasLimit := rm.limits[resourceType]
	rm.mu.RUnlock()

	if !hasLimit {
		return
	}

	rm.ensureUsageCounter(resourceType)
	rm.usage[resourceType].Add(delta)

	// Check threshold
	current := rm.usage[resourceType].Load()
	if limit.Threshold > 0 && limit.Value > 0 {
		percentage := float64(current) / float64(limit.Value) * 100
		if percentage >= limit.Threshold {
			rm.triggerCallbacks(resourceType, current, limit)
		}
	}

	// Update max usage
	rm.mu.Lock()
	if current > rm.maxUsage[resourceType] {
		rm.maxUsage[resourceType] = current
	}
	rm.mu.Unlock()
}

// SetUsage sets the usage of a resource
func (rm *ResourceMonitor) SetUsage(resourceType ResourceType, value int64) {
	rm.mu.RLock()
	limit, hasLimit := rm.limits[resourceType]
	rm.mu.RUnlock()

	if !hasLimit {
		return
	}

	rm.ensureUsageCounter(resourceType)
	rm.usage[resourceType].Store(value)

	// Check threshold
	if limit.Threshold > 0 && limit.Value > 0 {
		percentage := float64(value) / float64(limit.Value) * 100
		if percentage >= limit.Threshold {
			rm.triggerCallbacks(resourceType, value, limit)
		}
	}

	// Update max usage
	rm.mu.Lock()
	if value > rm.maxUsage[resourceType] {
		rm.maxUsage[resourceType] = value
	}
	rm.mu.Unlock()
}

// GetUsage gets the current usage of a resource
func (rm *ResourceMonitor) GetUsage(resourceType ResourceType) int64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	usage, ok := rm.usage[resourceType]
	if !ok {
		return 0
	}

	return usage.Load()
}

// GetMaxUsage gets the maximum usage of a resource
func (rm *ResourceMonitor) GetMaxUsage(resourceType ResourceType) int64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	return rm.maxUsage[resourceType]
}

// GetUsagePercentage gets the usage percentage of a resource
func (rm *ResourceMonitor) GetUsagePercentage(resourceType ResourceType) float64 {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	limit, limitOk := rm.limits[resourceType]
	usage, usageOk := rm.usage[resourceType]

	if !limitOk || !usageOk {
		return 0
	}

	return float64(usage.Load()) / float64(limit.Value) * 100
}

// AddCallback adds a callback for when a threshold is reached
func (rm *ResourceMonitor) AddCallback(resourceType ResourceType, callback func(usage int64, limit Limit)) {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.callbacks[resourceType] = append(rm.callbacks[resourceType], callback)
}

func (rm *ResourceMonitor) triggerCallbacks(resourceType ResourceType, usage int64, limit Limit) {
	rm.mu.RLock()
	callbacks := rm.callbacks[resourceType]
	rm.mu.RUnlock()

	for _, callback := range callbacks {
		callback(usage, limit)
	}
}

// GetStats returns all resource statistics
func (rm *ResourceMonitor) GetStats() map[string]interface{} {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	stats := make(map[string]interface{})

	for resourceType, limit := range rm.limits {
		usage := int64(0)
		if u, ok := rm.usage[resourceType]; ok {
			usage = u.Load()
		}

		percentage := 0.0
		if limit.Value > 0 {
			percentage = float64(usage) / float64(limit.Value) * 100
		}
		stats[string(resourceType)] = map[string]interface{}{
			"limit":      limit.Value,
			"unit":       limit.Unit,
			"usage":      usage,
			"maxUsage":   rm.maxUsage[resourceType],
			"percentage": percentage,
			"threshold":  limit.Threshold,
		}
	}

	return stats
}

func (rm *ResourceMonitor) ensureUsageCounter(resourceType ResourceType) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if rm.usage[resourceType] == nil {
		rm.usage[resourceType] = &atomic.Int64{}
	}
}

// MemoryLimiter limits memory usage
type MemoryLimiter struct {
	mu             sync.Mutex
	limit          int64
	allocated      int64
	gcThreshold    int64
	onLimitReached func(allocated int64)
}

// NewMemoryLimiter creates a new memory limiter
func NewMemoryLimiter(limitBytes int64, gcThreshold int64) *MemoryLimiter {
	if limitBytes <= 0 {
		limitBytes = 500 * 1024 * 1024 // 500MB default
	}
	if gcThreshold <= 0 {
		gcThreshold = limitBytes * 80 / 100 // 80% of limit
	}

	return &MemoryLimiter{
		limit:       limitBytes,
		gcThreshold: gcThreshold,
	}
}

// SetOnLimitReached sets a callback for when the limit is reached
func (ml *MemoryLimiter) SetOnLimitReached(callback func(allocated int64)) {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	ml.onLimitReached = callback
}

// Allocate allocates memory
func (ml *MemoryLimiter) Allocate(size int64) error {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	newAllocated := ml.allocated + size

	// Check if we're over the limit
	if newAllocated > ml.limit {
		err := &ResourceError{
			Type:    Memory,
			Value:   newAllocated,
			Limit:   ml.limit,
			Message: "memory limit exceeded",
		}
		if ml.onLimitReached != nil {
			ml.onLimitReached(newAllocated)
		}
		return err
	}

	ml.allocated = newAllocated

	// Check if we should trigger GC
	if ml.allocated > ml.gcThreshold {
		runtime.GC()
	}

	return nil
}

// Deallocate deallocates memory
func (ml *MemoryLimiter) Deallocate(size int64) {
	ml.mu.Lock()
	defer ml.mu.Unlock()

	if size > ml.allocated {
		ml.allocated = 0
	} else {
		ml.allocated -= size
	}
}

// GetUsage returns the current memory usage
func (ml *MemoryLimiter) GetUsage() int64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.allocated
}

// GetLimit returns the memory limit
func (ml *MemoryLimiter) GetLimit() int64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return ml.limit
}

// GetUsagePercentage returns the usage percentage
func (ml *MemoryLimiter) GetUsagePercentage() float64 {
	ml.mu.Lock()
	defer ml.mu.Unlock()
	return float64(ml.allocated) / float64(ml.limit) * 100
}

// ResourceError represents a resource error
type ResourceError struct {
	Type      ResourceType `json:"type"`
	Value     int64        `json:"value"`
	Limit     int64        `json:"limit"`
	Message   string       `json:"message"`
	Timestamp time.Time    `json:"timestamp"`
}

// Error implements the error interface
func (e *ResourceError) Error() string {
	return e.Message
}

// FileDescriptorLimiter limits the number of open file descriptors
type FileDescriptorLimiter struct {
	mu             sync.Mutex
	limit          int
	openCount      int
	onLimitReached func(openCount int)
}

// NewFileDescriptorLimiter creates a new file descriptor limiter
func NewFileDescriptorLimiter(limit int) *FileDescriptorLimiter {
	if limit <= 0 {
		limit = 1024 // Default limit
	}

	return &FileDescriptorLimiter{
		limit: limit,
	}
}

// SetOnLimitReached sets a callback for when the limit is reached
func (fdl *FileDescriptorLimiter) SetOnLimitReached(callback func(openCount int)) {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()
	fdl.onLimitReached = callback
}

// Acquire acquires a file descriptor
func (fdl *FileDescriptorLimiter) Acquire() error {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()

	if fdl.openCount >= fdl.limit {
		return &ResourceError{
			Type:    Files,
			Value:   int64(fdl.openCount),
			Limit:   int64(fdl.limit),
			Message: "file descriptor limit exceeded",
		}
	}

	fdl.openCount++
	return nil
}

// Release releases a file descriptor
func (fdl *FileDescriptorLimiter) Release() {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()

	if fdl.openCount > 0 {
		fdl.openCount--
	}
}

// GetUsage returns the current number of open file descriptors
func (fdl *FileDescriptorLimiter) GetUsage() int {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()
	return fdl.openCount
}

// GetLimit returns the file descriptor limit
func (fdl *FileDescriptorLimiter) GetLimit() int {
	fdl.mu.Lock()
	defer fdl.mu.Unlock()
	return fdl.limit
}

// RateLimiter limits the rate of operations
type RateLimiter struct {
	mu             sync.Mutex
	limit          int64
	used           int64
	resetTime      time.Time
	window         time.Duration
	onLimitReached func(used int64)
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int64, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 1000 // Default: 1000 operations per window
	}
	if window <= 0 {
		window = 1 * time.Second // Default: 1 second window
	}

	return &RateLimiter{
		limit:     limit,
		resetTime: time.Now().Add(window),
		window:    window,
	}
}

// SetOnLimitReached sets a callback for when the limit is reached
func (rl *RateLimiter) SetOnLimitReached(callback func(used int64)) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.onLimitReached = callback
}

// Allow checks if an operation is allowed
func (rl *RateLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset if window has passed
	if now.After(rl.resetTime) {
		rl.used = 0
		rl.resetTime = now.Add(rl.window)
	}

	// Check if we're under the limit
	if rl.used < rl.limit {
		rl.used++
		return true
	}

	// Trigger callback if limit is reached
	if rl.onLimitReached != nil {
		rl.onLimitReached(rl.used)
	}

	return false
}

// GetUsage returns the current usage in the window
func (rl *RateLimiter) GetUsage() int64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.used
}

// GetLimit returns the rate limit
func (rl *RateLimiter) GetLimit() int64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.limit
}

// GetRemaining returns the remaining operations in the current window
func (rl *RateLimiter) GetRemaining() int64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.limit - rl.used
}

// GetResetTime returns the time when the window resets
func (rl *RateLimiter) GetResetTime() time.Time {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.resetTime
}

// CPUProfiler profiles CPU usage
type CPUProfiler struct {
	mu          sync.Mutex
	enabled     bool
	samples     []CPUSample
	maxSamples  int
	onHighUsage func(usage float64)
}

// CPUSample represents a CPU usage sample
type CPUSample struct {
	Timestamp time.Time `json:"timestamp"`
	Usage     float64   `json:"usage"`
	Delta     float64   `json:"delta"`
}

// NewCPUProfiler creates a new CPU profiler
func NewCPUProfiler(maxSamples int) *CPUProfiler {
	if maxSamples <= 0 {
		maxSamples = 100
	}

	return &CPUProfiler{
		enabled:    true,
		samples:    make([]CPUSample, 0, maxSamples),
		maxSamples: maxSamples,
	}
}

// SetOnHighUsage sets a callback for when CPU usage is high
func (cp *CPUProfiler) SetOnHighUsage(callback func(usage float64)) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.onHighUsage = callback
}

// Sample takes a CPU usage sample
func (cp *CPUProfiler) Sample() {
	if !cp.enabled {
		return
	}

	// Get current CPU usage (this is platform-specific)
	// For simplicity, we'll use a mock implementation
	usage := getCPUUsage()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	var delta float64
	if len(cp.samples) > 0 {
		delta = usage - cp.samples[len(cp.samples)-1].Usage
	}

	sample := CPUSample{
		Timestamp: time.Now(),
		Usage:     usage,
		Delta:     delta,
	}

	cp.samples = append(cp.samples, sample)

	// Keep only the last maxSamples
	if len(cp.samples) > cp.maxSamples {
		cp.samples = cp.samples[1:]
	}

	// Check if usage is high
	if cp.onHighUsage != nil && usage > 80 { // 80% threshold
		cp.onHighUsage(usage)
	}
}

// GetAverageUsage returns the average CPU usage over all samples
func (cp *CPUProfiler) GetAverageUsage() float64 {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if len(cp.samples) == 0 {
		return 0
	}

	var sum float64
	for _, sample := range cp.samples {
		sum += sample.Usage
	}

	return sum / float64(len(cp.samples))
}

// GetLatestUsage returns the latest CPU usage
func (cp *CPUProfiler) GetLatestUsage() float64 {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if len(cp.samples) == 0 {
		return 0
	}

	return cp.samples[len(cp.samples)-1].Usage
}

// Enable enables the profiler
func (cp *CPUProfiler) Enable() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.enabled = true
}

// Disable disables the profiler
func (cp *CPUProfiler) Disable() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.enabled = false
}

// getCPUUsage is a mock implementation of getting CPU usage
// In a real implementation, this would read from /proc or use platform-specific APIs
func getCPUUsage() float64 {
	// This is a placeholder implementation
	// In production, you would use:
	// - Unix: read from /proc/stat
	// - Windows: GetSystemTimes API
	// - macOS: host_processor_info
	return 0
}

// ResourceTracker tracks resource usage over time
type ResourceTracker struct {
	mu               sync.RWMutex
	metrics          map[ResourceType][]MetricPoint
	maxPoints        int
	samplingInterval time.Duration
	ticker           *time.Ticker
	monitor          *ResourceMonitor
}

// MetricPoint represents a metric data point
type MetricPoint struct {
	Timestamp time.Time         `json:"timestamp"`
	Type      ResourceType      `json:"type"`
	Value     int64             `json:"value"`
	Unit      string            `json:"unit"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// NewResourceTracker creates a new resource tracker
func NewResourceTracker(monitor *ResourceMonitor, samplingInterval time.Duration) *ResourceTracker {
	if samplingInterval <= 0 {
		samplingInterval = 5 * time.Second
	}

	return &ResourceTracker{
		monitor:          monitor,
		metrics:          make(map[ResourceType][]MetricPoint),
		maxPoints:        1000,
		samplingInterval: samplingInterval,
	}
}

// Start starts tracking resources
func (rt *ResourceTracker) Start() {
	rt.ticker = time.NewTicker(rt.samplingInterval)

	go func() {
		for range rt.ticker.C {
			rt.sample()
		}
	}()
}

// Stop stops tracking resources
func (rt *ResourceTracker) Stop() {
	if rt.ticker != nil {
		rt.ticker.Stop()
	}
}

// sample collects current resource usage
func (rt *ResourceTracker) sample() {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	stats := rt.monitor.GetStats()

	for resourceType, data := range stats {
		dataMap, _ := data.(map[string]interface{})
		if dataMap == nil {
			continue
		}
		value := toInt64(dataMap["usage"])
		unit, _ := dataMap["unit"].(string)
		metric := MetricPoint{
			Timestamp: time.Now(),
			Type:      ResourceType(resourceType),
			Value:     value,
			Unit:      unit,
		}

		rt.metrics[ResourceType(resourceType)] = append(rt.metrics[ResourceType(resourceType)], metric)

		// Keep only the last maxPoints
		if len(rt.metrics[ResourceType(resourceType)]) > rt.maxPoints {
			rt.metrics[ResourceType(resourceType)] = rt.metrics[ResourceType(resourceType)][1:]
		}
	}
}

// GetMetrics returns metrics for a resource type
func (rt *ResourceTracker) GetMetrics(resourceType ResourceType) []MetricPoint {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	metrics := make([]MetricPoint, len(rt.metrics[resourceType]))
	copy(metrics, rt.metrics[resourceType])

	return metrics
}

// GetAllMetrics returns all metrics
func (rt *ResourceTracker) GetAllMetrics() map[ResourceType][]MetricPoint {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	all := make(map[ResourceType][]MetricPoint)
	for k, v := range rt.metrics {
		metrics := make([]MetricPoint, len(v))
		copy(metrics, v)
		all[k] = metrics
	}

	return all
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int:
		return int64(val)
	case int32:
		return int64(val)
	case int64:
		return val
	case float64:
		return int64(val)
	case float32:
		return int64(val)
	default:
		return 0
	}
}
