package main

import (
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestValidateTaskBusProviderConflict(t *testing.T) {
	tests := []struct {
		name           string
		proxy          string
		provider       string
		expectProvider string
	}{
		{name: "standalone keeps redis", proxy: "0", provider: "redis", expectProvider: "redis"},
		{name: "delegated keeps host", proxy: "1", provider: "host", expectProvider: "host"},
		{name: "standalone keeps host override", proxy: "0", provider: "host", expectProvider: "host"},
		{name: "delegated overrides redis to host", proxy: "1", provider: "redis", expectProvider: "host"},
		{name: "empty proxy uses config value", proxy: "", provider: "redis", expectProvider: "redis"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				EventBridge: &config.EventBridgeConfig{TaskBusProvider: tt.provider},
			}
			err := validateTaskBusProviderConflict(cfg, tt.proxy)
			if err != nil {
				t.Fatalf("期望校验成功，实际失败: %v", err)
			}
			if got := cfg.EventBridge.TaskBusProvider; got != tt.expectProvider {
				t.Fatalf("provider mismatch: got=%s want=%s", got, tt.expectProvider)
			}
		})
	}
}
