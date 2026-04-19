package providers

import (
	"sort"
	"strings"
	"sync"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
)

// Registry 是线程安全 provider 注册表。
type Registry struct {
	mu        sync.RWMutex
	providers map[string]contracts.Provider
}

// NewRegistry 创建 provider 注册表。
func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]contracts.Provider)}
}

// Register 注册 provider。
func (r *Registry) Register(provider contracts.Provider) error {
	if provider == nil {
		return contracts.NewError(contracts.ErrorCodeInvalidProvider, "provider is nil")
	}
	key := normalizeKey(provider.Key())
	if key == "" {
		return contracts.NewError(contracts.ErrorCodeInvalidProvider, "provider key is empty")
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.providers[key]; exists {
		return contracts.NewError(contracts.ErrorCodeProviderRegistered, "provider already registered")
	}
	r.providers[key] = provider
	return nil
}

// Get 读取 provider。
func (r *Registry) Get(key string) (contracts.Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	provider, ok := r.providers[normalizeKey(key)]
	return provider, ok
}

// MustGet 读取 provider，不存在返回统一错误。
func (r *Registry) MustGet(key string) (contracts.Provider, error) {
	provider, ok := r.Get(key)
	if !ok {
		return nil, contracts.NewError(contracts.ErrorCodeProviderNotFound, "provider not found")
	}
	return provider, nil
}

// List 返回已注册 key 列表（排序后）。
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	keys := make([]string, 0, len(r.providers))
	for key := range r.providers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func normalizeKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}
