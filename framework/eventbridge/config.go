package eventbridge

import "strings"

type Config struct {
	Enabled         bool
	Mode            string
	FallbackToLocal bool
	LocalQueueSize  int
}

func (c Config) Normalized() (Config, error) {
	cfg := c
	if cfg.LocalQueueSize <= 0 {
		cfg.LocalQueueSize = 1024
	}

	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		if cfg.Enabled {
			mode = "taskbus"
		} else {
			mode = "local"
		}
	}

	switch mode {
	case "local", "taskbus", "dual":
		cfg.Mode = mode
	default:
		return Config{}, ErrInvalidMode
	}

	if !cfg.Enabled {
		cfg.Mode = "local"
	}

	return cfg, nil
}

