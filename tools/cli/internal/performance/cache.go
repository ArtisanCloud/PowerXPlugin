package performance

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"sync"
)

// HashCache provides a simple in-memory cache for file hashes
type HashCache struct {
	cache map[string]string
	mu    sync.RWMutex
	hit   int64
	miss  int64
}

// NewHashCache creates a new hash cache
func NewHashCache() *HashCache {
	return &HashCache{
		cache: make(map[string]string),
	}
}

// Get retrieves a hash from the cache
func (c *HashCache) Get(path string) (string, bool) {
	c.mu.RLock()
	hash, ok := c.cache[path]
	c.mu.RUnlock()

	if ok {
		c.mu.Lock()
		c.hit++
		c.mu.Unlock()
		return hash, true
	}

	c.mu.Lock()
	c.miss++
	c.mu.Unlock()
	return "", false
}

// Set stores a hash in the cache
func (c *HashCache) Set(path, hash string) {
	c.mu.Lock()
	c.cache[path] = hash
	c.mu.Unlock()
}

// Clear clears the cache
func (c *HashCache) Clear() {
	c.mu.Lock()
	c.cache = make(map[string]string)
	c.mu.Unlock()
}

// Delete removes a single entry from the cache
func (c *HashCache) Delete(path string) {
	c.mu.Lock()
	delete(c.cache, path)
	c.mu.Unlock()
}

// Stats returns cache statistics
func (c *HashCache) Stats() (hit, miss, size int64) {
	c.mu.RLock()
	hit = c.hit
	miss = c.miss
	size = int64(len(c.cache))
	c.mu.RUnlock()
	return
}

// FastHasher provides optimized hashing
type FastHasher struct {
	pool sync.Pool
}

// NewFastHasher creates a new fast hasher
func NewFastHasher() *FastHasher {
	return &FastHasher{
		pool: sync.Pool{
			New: func() interface{} {
				return sha256.New()
			},
		},
	}
}

// Hash calculates the hash of data
func (h *FastHasher) Hash(data []byte) string {
	hasher := h.pool.Get().(hash.Hash)
	defer h.pool.Put(hasher)

	hasher.Reset()
	hasher.Write(data)
	hash := hasher.Sum(nil)

	return hex.EncodeToString(hash)
}

// BatchProcessor provides batched processing for better performance
type BatchProcessor struct {
	batchSize int
	mu        sync.Mutex
	batch     []interface{}
	handler   func([]interface{})
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(batchSize int, handler func([]interface{})) *BatchProcessor {
	return &BatchProcessor{
		batchSize: batchSize,
		handler:   handler,
		batch:     make([]interface{}, 0, batchSize),
	}
}

// Add adds an item to the batch
func (b *BatchProcessor) Add(item interface{}) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.batch = append(b.batch, item)

	if len(b.batch) >= b.batchSize {
		b.flush()
	}
}

// Flush flushes the current batch
func (b *BatchProcessor) Flush() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flush()
}

func (b *BatchProcessor) flush() {
	if len(b.batch) > 0 {
		b.handler(b.batch)
		b.batch = make([]interface{}, 0, b.batchSize)
	}
}

// StringPool provides string interning for repeated strings
type StringPool struct {
	pool map[string]string
	mu   sync.RWMutex
}

// NewStringPool creates a new string pool
func NewStringPool() *StringPool {
	return &StringPool{
		pool: make(map[string]string),
	}
}

// Get returns an interned string
func (p *StringPool) Get(s string) string {
	p.mu.RLock()
	if interned, ok := p.pool[s]; ok {
		p.mu.RUnlock()
		return interned
	}
	p.mu.RUnlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	// Double-checked locking
	if interned, ok := p.pool[s]; ok {
		return interned
	}

	p.pool[s] = s
	return s
}

// Stats returns pool statistics
func (p *StringPool) Stats() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.pool)
}
