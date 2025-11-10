package resources

import (
	"testing"
	"time"
)

func TestResourceMonitor_SetLimit(t *testing.T) {
	rm := NewResourceMonitor()

	// Set a limit
	rm.SetLimit(Limit{
		Type:  Memory,
		Value: 1024 * 1024 * 1024, // 1GB
		Unit:  "bytes",
	})

	// Get the limit back
	limit, ok := rm.GetLimit(Memory)
	if !ok {
		t.Error("Expected to get limit for Memory")
	}

	if limit.Value != 1024*1024*1024 {
		t.Errorf("Expected value 1GB, got %d", limit.Value)
	}
}

func TestResourceMonitor_AddUsage(t *testing.T) {
	rm := NewResourceMonitor()

	// Set a limit
	rm.SetLimit(Limit{
		Type:      Memory,
		Value:     100,
		Unit:      "bytes",
		Threshold: 80, // 80%
	})

	// Track threshold crossing
	crossed := false
	rm.AddCallback(Memory, func(usage int64, limit Limit) {
		crossed = true
	})

	// Add usage below threshold
	rm.AddUsage(Memory, 70)
	if rm.GetUsage(Memory) != 70 {
		t.Errorf("Expected usage 70, got %d", rm.GetUsage(Memory))
	}

	// Cross threshold
	rm.AddUsage(Memory, 15) // Total 85
	if !crossed {
		t.Error("Expected threshold callback to be triggered")
	}

	// Check percentage
	percentage := rm.GetUsagePercentage(Memory)
	if percentage != 85 {
		t.Errorf("Expected 85%%, got %f", percentage)
	}
}

func TestResourceMonitor_GetStats(t *testing.T) {
	rm := NewResourceMonitor()

	// Set multiple limits
	rm.SetLimit(Limit{Type: Memory, Value: 100, Unit: "bytes"})
	rm.SetLimit(Limit{Type: CPU, Value: 100, Unit: "percent"})

	// Add some usage
	rm.AddUsage(Memory, 50)
	rm.AddUsage(CPU, 75)

	// Get stats
	stats := rm.GetStats()

	if len(stats) != 2 {
		t.Errorf("Expected 2 stats, got %d", len(stats))
	}

	memStats, ok := stats["memory"].(map[string]interface{})
	if !ok {
		t.Fatalf("Expected memory stats map, got %T", stats["memory"])
	}
	if memStats["usage"].(int64) != 50 {
		t.Errorf("Expected memory usage 50, got %v", memStats["usage"])
	}

	if memStats["percentage"].(float64) != 50 {
		t.Errorf("Expected memory percentage 50, got %v", memStats["percentage"])
	}
}

func TestMemoryLimiter_Allocate(t *testing.T) {
	ml := NewMemoryLimiter(1000, 800) // 1KB limit, GC at 800 bytes

	// Test successful allocation
	err := ml.Allocate(500)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if ml.GetUsage() != 500 {
		t.Errorf("Expected usage 500, got %d", ml.GetUsage())
	}

	// Test limit exceeded
	err = ml.Allocate(600) // Total 1100 > 1000
	if err == nil {
		t.Error("Expected error for exceeding limit")
	}

	resourceErr, ok := err.(*ResourceError)
	if !ok {
		t.Error("Expected ResourceError type")
	}

	if resourceErr.Type != Memory {
		t.Errorf("Expected type Memory, got %v", resourceErr.Type)
	}

	if resourceErr.Value != 1100 {
		t.Errorf("Expected value 1100, got %d", resourceErr.Value)
	}
}

func TestMemoryLimiter_Deallocate(t *testing.T) {
	ml := NewMemoryLimiter(1000, 800)

	ml.Allocate(500)
	ml.Allocate(300)

	// Deallocate
	ml.Deallocate(200)
	if ml.GetUsage() != 600 {
		t.Errorf("Expected usage 600, got %d", ml.GetUsage())
	}

	// Deallocate more than allocated
	ml.Deallocate(1000)
	if ml.GetUsage() != 0 {
		t.Errorf("Expected usage 0, got %d", ml.GetUsage())
	}
}

func TestMemoryLimiter_GetUsagePercentage(t *testing.T) {
	ml := NewMemoryLimiter(1000, 800)

	ml.Allocate(250)
	percentage := ml.GetUsagePercentage()
	if percentage != 25 {
		t.Errorf("Expected 25%%, got %f", percentage)
	}
}

func TestFileDescriptorLimiter_Acquire(t *testing.T) {
	fdl := NewFileDescriptorLimiter(3)

	// Acquire within limit
	err := fdl.Acquire()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if fdl.GetUsage() != 1 {
		t.Errorf("Expected usage 1, got %d", fdl.GetUsage())
	}

	// Acquire up to limit
	fdl.Acquire()
	fdl.Acquire()

	if fdl.GetUsage() != 3 {
		t.Errorf("Expected usage 3, got %d", fdl.GetUsage())
	}

	// Exceed limit
	err = fdl.Acquire()
	if err == nil {
		t.Error("Expected error for exceeding limit")
	}

	resourceErr, ok := err.(*ResourceError)
	if !ok {
		t.Error("Expected ResourceError type")
	}

	if resourceErr.Type != Files {
		t.Errorf("Expected type Files, got %v", resourceErr.Type)
	}
}

func TestFileDescriptorLimiter_Release(t *testing.T) {
	fdl := NewFileDescriptorLimiter(3)

	fdl.Acquire()
	fdl.Acquire()

	fdl.Release()
	if fdl.GetUsage() != 1 {
		t.Errorf("Expected usage 1, got %d", fdl.GetUsage())
	}

	// Release more than acquired
	fdl.Release()
	fdl.Release()
	if fdl.GetUsage() != 0 {
		t.Errorf("Expected usage 0, got %d", fdl.GetUsage())
	}
}

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	// First 3 should be allowed
	if !rl.Allow() {
		t.Error("Expected first operation to be allowed")
	}
	if !rl.Allow() {
		t.Error("Expected second operation to be allowed")
	}
	if !rl.Allow() {
		t.Error("Expected third operation to be allowed")
	}

	// 4th should be blocked
	if rl.Allow() {
		t.Error("Expected fourth operation to be blocked")
	}

	// Check usage
	if rl.GetUsage() != 3 {
		t.Errorf("Expected usage 3, got %d", rl.GetUsage())
	}

	// Check remaining
	if rl.GetRemaining() != 0 {
		t.Errorf("Expected 0 remaining, got %d", rl.GetRemaining())
	}
}

func TestRateLimiter_WindowReset(t *testing.T) {
	rl := NewRateLimiter(3, 100*time.Millisecond)

	// Use up the limit
	rl.Allow()
	rl.Allow()
	rl.Allow()

	// Should be blocked
	if rl.Allow() {
		t.Error("Expected operation to be blocked before window reset")
	}

	// Wait for window to reset
	time.Sleep(150 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow() {
		t.Error("Expected operation to be allowed after window reset")
	}
}

func TestRateLimiter_GetResetTime(t *testing.T) {
	rl := NewRateLimiter(3, time.Second)

	resetTime := rl.GetResetTime()
	if resetTime.IsZero() {
		t.Error("Expected non-zero reset time")
	}

	// Reset time should be in the future
	if time.Now().After(resetTime) {
		t.Error("Expected reset time to be in the future")
	}
}

func TestCPUProfiler_Sample(t *testing.T) {
	cp := NewCPUProfiler(10)

	// Track high usage
	cp.SetOnHighUsage(func(usage float64) {
		// no-op; placeholder for future assertions
	})

	// Sample multiple times
	for i := 0; i < 5; i++ {
		cp.Sample()
	}

	// Get latest usage
	latest := cp.GetLatestUsage()
	if latest < 0 {
		t.Error("Expected non-negative CPU usage")
	}

	// Get average usage
	avg := cp.GetAverageUsage()
	if avg < 0 {
		t.Error("Expected non-negative average CPU usage")
	}
}

func TestCPUProfiler_EnableDisable(t *testing.T) {
	cp := NewCPUProfiler(10)

	cp.Disable()
	cp.Sample()

	// Should not sample when disabled
	if len(cp.samples) != 0 {
		t.Error("Expected no samples when disabled")
	}

	cp.Enable()
	cp.Sample()

	// Should sample when enabled
	if len(cp.samples) == 0 {
		t.Error("Expected samples when enabled")
	}
}

func TestResourceTracker_StartStop(t *testing.T) {
	rm := NewResourceMonitor()
	rm.SetLimit(Limit{Type: Memory, Value: 100, Unit: "bytes"})

	rt := NewResourceTracker(rm, 100*time.Millisecond)

	// Start tracking
	rt.Start()

	// Wait for a sample
	time.Sleep(150 * time.Millisecond)

	// Get metrics
	metrics := rt.GetMetrics(Memory)
	if len(metrics) == 0 {
		t.Error("Expected at least one metric sample")
	}

	// Stop tracking
	rt.Stop()

	// Should not get new samples after stopping
	count := len(metrics)
	time.Sleep(150 * time.Millisecond)

	newMetrics := rt.GetMetrics(Memory)
	if len(newMetrics) > count {
		t.Error("Expected no new metrics after stopping")
	}
}

func TestResourceTracker_GetAllMetrics(t *testing.T) {
	rm := NewResourceMonitor()
	rm.SetLimit(Limit{Type: Memory, Value: 100, Unit: "bytes"})
	rm.SetLimit(Limit{Type: CPU, Value: 100, Unit: "percent"})

	rt := NewResourceTracker(rm, time.Hour) // Very long interval
	rt.Start()

	// Add some usage
	rm.AddUsage(Memory, 50)
	rm.AddUsage(CPU, 75)

	// Force a sample
	rt.sample()

	// Get all metrics
	all := rt.GetAllMetrics()

	if len(all) != 2 {
		t.Errorf("Expected 2 metric types, got %d", len(all))
	}

	if len(all[Memory]) == 0 {
		t.Error("Expected at least one memory metric")
	}

	if len(all[CPU]) == 0 {
		t.Error("Expected at least one CPU metric")
	}

	rt.Stop()
}
