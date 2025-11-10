package performance

import (
	"sync"
	"testing"
	"time"
)

func TestHashCacheBasicOps(t *testing.T) {
	cache := NewHashCache()

	if _, ok := cache.Get("missing"); ok {
		t.Fatalf("expected miss for unknown key")
	}

	cache.Set("file", "hash")
	if v, ok := cache.Get("file"); !ok || v != "hash" {
		t.Fatalf("expected hash value, got %q", v)
	}

	// Stats should reflect 1 hit / 1 miss / size 1
	hit, miss, size := cache.Stats()
	if hit != 1 || miss != 1 || size != 1 {
		t.Fatalf("unexpected stats hit=%d miss=%d size=%d", hit, miss, size)
	}

	cache.Delete("file")
	if _, ok := cache.Get("file"); ok {
		t.Fatalf("expected key to be deleted")
	}

	cache.Set("a", "1")
	cache.Clear()
	if _, ok := cache.Get("a"); ok {
		t.Fatalf("expected cache to be cleared")
	}
}

func TestFastHasherDeterministic(t *testing.T) {
	hasher := NewFastHasher()
	data := []byte("hello world")

	h1 := hasher.Hash(data)
	h2 := hasher.Hash(data)
	if h1 == "" || h1 != h2 {
		t.Fatalf("hash mismatch %q vs %q", h1, h2)
	}
}

func TestBatchProcessor(t *testing.T) {
	var (
		lock sync.Mutex
		got  []interface{}
	)
	handler := func(items []interface{}) {
		lock.Lock()
		defer lock.Unlock()
		got = append(got, items...)
	}

	b := NewBatchProcessor(3, handler)
	b.Add("a")
	b.Add("b")
	b.Add("c") // trigger flush
	b.Add("d")
	b.Flush()

	lock.Lock()
	defer lock.Unlock()
	if len(got) != 4 {
		t.Fatalf("expected 4 items, got %d", len(got))
	}
}

func TestStringPool(t *testing.T) {
	pool := NewStringPool()
	str1 := pool.Get("demo")
	str2 := pool.Get("demo")

	if &str1 == &str2 {
		t.Fatalf("expected different variable addresses")
	}
	if str1 != str2 {
		t.Fatalf("pooled strings mismatch")
	}
	if pool.Stats() != 1 {
		t.Fatalf("expected pool size 1")
	}
}

func TestTimerAndThrottle(t *testing.T) {
	timer := NewTimer()
	time.Sleep(5 * time.Millisecond)
	if timer.Elapsed() < 5*time.Millisecond {
		t.Fatalf("timer elapsed too small")
	}

	timer.Reset()
	if timer.Elapsed() > 2*time.Millisecond {
		t.Fatalf("timer reset should be near-zero")
	}

	throttler := NewThrottler(10 * time.Millisecond)
	if !throttler.Allow() {
		t.Fatalf("first allow should pass")
	}
	if throttler.Allow() {
		t.Fatalf("second allow should be throttled")
	}
	time.Sleep(12 * time.Millisecond)
	if !throttler.Allow() {
		t.Fatalf("throttler should allow after interval")
	}
}

func TestLatencyTrackerPercentile(t *testing.T) {
	tracker := NewLatencyTracker(5)
	for _, d := range []time.Duration{10, 20, 30, 40, 50} {
		tracker.Add(d * time.Millisecond)
	}
	if got := tracker.Percentile(95); got < 40*time.Millisecond {
		t.Fatalf("expected high percentile >=40ms, got %s", got)
	}
	if got := tracker.Percentile(50); got != 30*time.Millisecond {
		t.Fatalf("expected median 30ms, got %s", got)
	}
}
