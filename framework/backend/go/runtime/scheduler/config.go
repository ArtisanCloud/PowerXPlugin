package scheduler

import "strings"

type Config struct {
	Enabled         bool
	Mode            string
	FallbackToLocal bool
	OwnerType       string
	OwnerID         string
	DefaultTopic    string
}

func (c Config) Normalized() (Config, error) {
	cfg := c
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "" {
		if cfg.Enabled {
			mode = ModeHost
		} else {
			mode = ModeLocal
		}
	}
	if !cfg.Enabled {
		mode = ModeLocal
	}
	switch mode {
	case ModeLocal, ModeHost, ModeDual:
		cfg.Mode = mode
	default:
		return Config{}, ErrInvalidMode
	}
	if strings.TrimSpace(cfg.OwnerType) == "" {
		cfg.OwnerType = OwnerTypePlugin
	}
	if strings.TrimSpace(cfg.DefaultTopic) == "" {
		cfg.DefaultTopic = DefaultTriggeredTopic
	}
	return cfg, nil
}
