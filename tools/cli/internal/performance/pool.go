package performance

import (
	"net/http"
	"sync"
	"time"
)

// HTTPClientPool provides a pool of HTTP clients for connection reuse
type HTTPClientPool struct {
	mu       sync.Mutex
	clients  []*http.Client
	idx      int
	closed   bool
}

// NewHTTPClientPool creates a new HTTP client pool
func NewHTTPClientPool(size int) *HTTPClientPool {
	if size <= 0 {
		size = 10
	}

	clients := make([]*http.Client, size)
	for i := 0; i < size; i++ {
		clients[i] = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
			Timeout: 30 * time.Second,
		}
	}

	return &HTTPClientPool{
		clients: clients,
		idx:     0,
	}
}

// Get returns a client from the pool (round-robin)
func (p *HTTPClientPool) Get() *http.Client {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return http.DefaultClient
	}

	client := p.clients[p.idx]
	p.idx = (p.idx + 1) % len(p.clients)
	return client
}

// Close closes all clients in the pool
func (p *HTTPClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.closed {
		return
	}

	p.closed = true
	// Note: We don't actually close the clients as they may be in use
	// In a production system, you'd use a more sophisticated pool
}

// Preloader preloads data for faster access
type Preloader struct {
	data   map[string][]byte
	mu     sync.RWMutex
	loaded int64
}

// NewPreloader creates a new preloader
func NewPreloader() *Preloader {
	return &Preloader{
		data: make(map[string][]byte),
	}
}

// Add adds data to the preloader
func (p *Preloader) Add(key string, data []byte) {
	p.mu.Lock()
	p.data[key] = data
	p.loaded++
	p.mu.Unlock()
}

// Get retrieves preloaded data
func (p *Preloader) Get(key string) ([]byte, bool) {
	p.mu.RLock()
	data, ok := p.data[key]
	p.mu.RUnlock()
	return data, ok
}

// Stats returns preloader statistics
func (p *Preloader) Stats() (count, loaded int64) {
	p.mu.RLock()
	count = int64(len(p.data))
	loaded = p.loaded
	p.mu.RUnlock()
	return
}

// GC performs garbage collection on the preloader
func (p *Preloader) GC() {
	// Simple GC: keep only the most recently used items
	// In a real implementation, you'd track access times
	p.mu.Lock()
	// For now, just clear old data if we have too much
	if len(p.data) > 1000 {
		// Keep only the last 500 items
		newData := make(map[string][]byte)
		count := 0
		for k, v := range p.data {
			newData[k] = v
			count++
			if count >= 500 {
				break
			}
		}
		p.data = newData
	}
	p.mu.Unlock()
}
