package performance

import (
	"sync"
	"time"
)

// Throttler provides rate limiting functionality
type Throttler struct {
	mu        sync.Mutex
	rate      time.Duration
	lastTime  time.Time
	allowance float64
	maxAllow  float64
}

// NewThrottler creates a new throttler
// rate is the time between allowed operations (e.g., 250ms means 4 operations per second)
func NewThrottler(rate time.Duration) *Throttler {
	return &Throttler{
		rate:      rate,
		maxAllow:  1.0,
		allowance: 1.0,
	}
}

// Allow checks if an operation is allowed
func (t *Throttler) Allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(t.lastTime)

	// Add allowance based on elapsed time
	t.allowance += elapsed.Seconds() / t.rate.Seconds()
	t.lastTime = now

	// Cap allowance
	if t.allowance > t.maxAllow {
		t.allowance = t.maxAllow
	}

	// Check if we can proceed
	if t.allowance >= 1.0 {
		t.allowance -= 1.0
		return true
	}

	return false
}

// Reset resets the throttler
func (t *Throttler) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.lastTime = time.Now()
	t.allowance = t.maxAllow
}

// RateLimiter provides token bucket rate limiting
type RateLimiter struct {
	mu          sync.Mutex
	tokens      float64
	maxTokens   float64
	tokensPerSec float64
	lastTime    time.Time
}

// NewRateLimiter creates a new rate limiter
// maxTokens is the maximum number of tokens
// tokensPerSec is the rate at which tokens are added
func NewRateLimiter(maxTokens, tokensPerSec float64) *RateLimiter {
	if maxTokens <= 0 {
		maxTokens = 10
	}
	if tokensPerSec <= 0 {
		tokensPerSec = 1
	}

	return &RateLimiter{
		tokens:      maxTokens,
		maxTokens:   maxTokens,
		tokensPerSec: tokensPerSec,
		lastTime:    time.Now(),
	}
}

// Allow checks if an operation is allowed and consumes a token
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastTime)

	// Add tokens based on elapsed time
	r.tokens += elapsed.Seconds() * r.tokensPerSec
	r.lastTime = now

	// Cap tokens
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}

	// Check if we can consume a token
	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return true
	}

	return false
}

// Wait waits until a token is available
func (r *RateLimiter) Wait() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return
	}

	// Calculate wait time
	wait := (1.0 - r.tokens) / r.tokensPerSec
	time.Sleep(time.Duration(wait * float64(time.Second)))
}

// Reset resets the rate limiter
func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = r.maxTokens
	r.lastTime = time.Now()
}

// ConcurrencyLimiter limits the number of concurrent operations
type ConcurrencyLimiter struct {
	sem   chan struct{}
	mu    sync.Mutex
	count int
	max   int
}

// NewConcurrencyLimiter creates a new concurrency limiter
func NewConcurrencyLimiter(max int) *ConcurrencyLimiter {
	if max <= 0 {
		max = 10
	}

	return &ConcurrencyLimiter{
		sem: make(chan struct{}, max),
		max: max,
	}
}

// Acquire acquires a slot
func (c *ConcurrencyLimiter) Acquire() {
	c.mu.Lock()
	c.count++
	c.mu.Unlock()

	c.sem <- struct{}{}
}

// Release releases a slot
func (c *ConcurrencyLimiter) Release() {
	<-c.sem

	c.mu.Lock()
	c.count--
	c.mu.Unlock()
}

// AcquireWithTimeout acquires a slot with timeout
func (c *ConcurrencyLimiter) AcquireWithTimeout(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c.sem <- struct{}{}:
		c.mu.Lock()
		c.count++
		c.mu.Unlock()
		return true
	case <-timer.C:
		return false
	}
}

// Current returns the current number of acquired slots
func (c *ConcurrencyLimiter) Current() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.count
}

// Max returns the maximum number of slots
func (c *ConcurrencyLimiter) Max() int {
	return c.max
}
