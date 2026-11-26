package eventbridge

import "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/provider"

type Config struct {
	Enabled        bool
	RuntimeMode    provider.RuntimeMode
	LocalQueueSize int
}

func (c Config) Normalized() (Config, error) {
	cfg := c
	if cfg.LocalQueueSize <= 0 {
		cfg.LocalQueueSize = 1024
	}

	if err := cfg.RuntimeMode.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
