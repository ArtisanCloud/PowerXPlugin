package performance

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// Counter provides atomic counter operations
type Counter struct {
	value uint64
}

// Add adds a value to the counter
func (c *Counter) Add(delta uint64) {
	atomic.AddUint64(&c.value, delta)
}

// Value returns the current value
func (c *Counter) Value() uint64 {
	return atomic.LoadUint64(&c.value)
}

// Reset resets the counter
func (c *Counter) Reset() {
	atomic.StoreUint64(&c.value, 0)
}

// Gauge provides a gauge that can go up and down
type Gauge struct {
	value int64
	mu    sync.RWMutex
}

// Add adds a value to the gauge
func (g *Gauge) Add(delta int64) {
	g.mu.Lock()
	g.value += delta
	g.mu.Unlock()
}

// Set sets the gauge to a value
func (g *Gauge) Set(value int64) {
	g.mu.Lock()
	g.value = value
	g.mu.Unlock()
}

// Value returns the current value
func (g *Gauge) Value() int64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// Histogram tracks a distribution of values
type Histogram struct {
	counts map[string]*Counter
	mu     sync.RWMutex
}

// NewHistogram creates a new histogram
func NewHistogram() *Histogram {
	return &Histogram{
		counts: make(map[string]*Counter),
	}
}

// Observe adds an observation
func (h *Histogram) Observe(value string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, ok := h.counts[value]; !ok {
		h.counts[value] = &Counter{}
	}
	h.counts[value].Add(1)
}

// Value returns the count for a value
func (h *Histogram) Value(value string) uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if counter, ok := h.counts[value]; ok {
		return counter.Value()
	}
	return 0
}

// Stats returns all histogram stats
func (h *Histogram) Stats() map[string]uint64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	stats := make(map[string]uint64)
	for k, v := range h.counts {
		stats[k] = v.Value()
	}
	return stats
}

// Timer provides timing measurements
type Timer struct {
	start time.Time
	mu    sync.RWMutex
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{
		start: time.Now(),
	}
}

// Elapsed returns the elapsed time
func (t *Timer) Elapsed() time.Duration {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return time.Since(t.start)
}

// Reset resets the timer
func (t *Timer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.start = time.Now()
}

// MetricsCollector collects various performance metrics
type MetricsCollector struct {
	eventsTotal    *Counter
	eventsSkipped  *Counter
	buildsTotal    *Counter
	buildsSuccess  *Counter
	buildsFailed   *Counter
	reloadsTotal   *Counter
	reloadsSuccess *Counter
	activeSessions *Gauge
	cpuUsage       *Gauge
	memoryUsage    *Gauge
	diskUsage      *Gauge
	histogram      *Histogram
	reloadLatency  *LatencyTracker
	eventLatency   *LatencyTracker
	buildLatency   *LatencyTracker
	mu             sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		eventsTotal:    &Counter{},
		eventsSkipped:  &Counter{},
		buildsTotal:    &Counter{},
		buildsSuccess:  &Counter{},
		buildsFailed:   &Counter{},
		reloadsTotal:   &Counter{},
		reloadsSuccess: &Counter{},
		activeSessions: &Gauge{},
		cpuUsage:       &Gauge{},
		memoryUsage:    &Gauge{},
		diskUsage:      &Gauge{},
		histogram:      NewHistogram(),
		reloadLatency:  NewLatencyTracker(200),
		eventLatency:   NewLatencyTracker(200),
		buildLatency:   NewLatencyTracker(200),
	}
}

// RecordEvent records an event
func (m *MetricsCollector) RecordEvent() {
	m.eventsTotal.Add(1)
}

// RecordEventSkipped records a skipped event
func (m *MetricsCollector) RecordEventSkipped() {
	m.eventsSkipped.Add(1)
}

// RecordBuild records a build attempt
func (m *MetricsCollector) RecordBuild(success bool) {
	m.buildsTotal.Add(1)
	if success {
		m.buildsSuccess.Add(1)
	} else {
		m.buildsFailed.Add(1)
	}
}

// RecordReload records a reload attempt
func (m *MetricsCollector) RecordReload(success bool) {
	m.reloadsTotal.Add(1)
	if success {
		m.reloadsSuccess.Add(1)
	}
}

// RecordBuildLatency records build duration
func (m *MetricsCollector) RecordBuildLatency(d time.Duration) {
	m.buildLatency.Add(d)
}

// RecordReloadLatency records reload duration
func (m *MetricsCollector) RecordReloadLatency(d time.Duration) {
	m.reloadLatency.Add(d)
}

// RecordEventLatency records time from event to reload
func (m *MetricsCollector) RecordEventLatency(d time.Duration) {
	m.eventLatency.Add(d)
}

// UpdateActiveSessions updates the active sessions count
func (m *MetricsCollector) UpdateActiveSessions(delta int64) {
	m.activeSessions.Add(delta)
}

// UpdateCPUUsage updates CPU usage percentage
func (m *MetricsCollector) UpdateCPUUsage(percent int64) {
	m.cpuUsage.Set(percent)
}

// UpdateMemoryUsage updates memory usage in MB
func (m *MetricsCollector) UpdateMemoryUsage(mb int64) {
	m.memoryUsage.Set(mb)
}

// UpdateDiskUsage updates disk usage in MB
func (m *MetricsCollector) UpdateDiskUsage(mb int64) {
	m.diskUsage.Set(mb)
}

// ObserveValue observes a value for histogram
func (m *MetricsCollector) ObserveValue(value string) {
	m.histogram.Observe(value)
}

// GetStats returns all metrics
func (m *MetricsCollector) GetStats() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := make(map[string]interface{})

	// Counters
	stats["events_total"] = m.eventsTotal.Value()
	stats["events_skipped"] = m.eventsSkipped.Value()
	stats["builds_total"] = m.buildsTotal.Value()
	stats["builds_success"] = m.buildsSuccess.Value()
	stats["builds_failed"] = m.buildsFailed.Value()
	stats["reloads_total"] = m.reloadsTotal.Value()
	stats["reloads_success"] = m.reloadsSuccess.Value()

	// Gauges
	stats["active_sessions"] = m.activeSessions.Value()
	stats["cpu_usage_percent"] = m.cpuUsage.Value()
	stats["memory_usage_mb"] = m.memoryUsage.Value()
	stats["disk_usage_mb"] = m.diskUsage.Value()

	// Histogram
	stats["histogram"] = m.histogram.Stats()

	stats["reload_latency_p95_ms"] = durationMillis(m.reloadLatency.Percentile(95))
	stats["reload_latency_p50_ms"] = durationMillis(m.reloadLatency.Percentile(50))
	stats["event_latency_p95_ms"] = durationMillis(m.eventLatency.Percentile(95))
	stats["build_latency_p95_ms"] = durationMillis(m.buildLatency.Percentile(95))

	// Calculated metrics
	if m.buildsTotal.Value() > 0 {
		successRate := float64(m.buildsSuccess.Value()) / float64(m.buildsTotal.Value())
		stats["build_success_rate"] = successRate
	}

	if m.reloadsTotal.Value() > 0 {
		successRate := float64(m.reloadsSuccess.Value()) / float64(m.reloadsTotal.Value())
		stats["reload_success_rate"] = successRate
	}

	return stats
}

// Reset resets all metrics
func (m *MetricsCollector) Reset() {
	m.eventsTotal.Reset()
	m.eventsSkipped.Reset()
	m.buildsTotal.Reset()
	m.buildsSuccess.Reset()
	m.buildsFailed.Reset()
	m.reloadsTotal.Reset()
	m.reloadsSuccess.Reset()
	m.activeSessions.Set(0)
	m.cpuUsage.Set(0)
	m.memoryUsage.Set(0)
	m.diskUsage.Set(0)
}

func durationMillis(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	ms := d.Milliseconds()
	if ms == 0 {
		return 1
	}
	return ms
}

// LatencyTracker tracks recent durations for percentile calculation
type LatencyTracker struct {
	samples []time.Duration
	max     int
	mu      sync.RWMutex
}

// NewLatencyTracker creates tracker storing up to max samples
func NewLatencyTracker(max int) *LatencyTracker {
	if max <= 0 {
		max = 100
	}
	return &LatencyTracker{
		samples: make([]time.Duration, 0, max),
		max:     max,
	}
}

// Add records a latency sample
func (l *LatencyTracker) Add(d time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.samples) == l.max {
		copy(l.samples, l.samples[1:])
		l.samples = l.samples[:l.max-1]
	}
	l.samples = append(l.samples, d)
}

// Percentile returns the pth percentile duration
func (l *LatencyTracker) Percentile(p float64) time.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if len(l.samples) == 0 {
		return 0
	}
	values := append([]time.Duration(nil), l.samples...)
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	index := int((p / 100.0) * float64(len(values)-1))
	if index < 0 {
		index = 0
	}
	if index >= len(values) {
		index = len(values) - 1
	}
	return values[index]
}
