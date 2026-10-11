package wsbus

import (
	"context"
	"errors"
	"strings"
)

// OpenLocalHub selects an explicit local transport. A failed Redis subscription
// never changes the selected transport. The caller must invoke cleanup.
func OpenLocalHub(ctx context.Context, provider string, cfg RedisHubConfig) (LocalHub, func() error, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case "", "memory":
		return NewMemoryHub(), func() error { return nil }, nil
	case "redis":
		hub, err := NewRedisHub(cfg)
		if err != nil {
			return nil, nil, err
		}
		if ctx == nil {
			ctx = context.Background()
		}
		runCtx, cancel := context.WithCancel(ctx)
		if err := hub.Start(runCtx); err != nil {
			cancel()
			_ = hub.Close()
			return nil, nil, err
		}
		return hub, func() error { cancel(); return hub.Close() }, nil
	default:
		return nil, nil, errors.New("WSBUS_PROVIDER_INVALID")
	}
}
