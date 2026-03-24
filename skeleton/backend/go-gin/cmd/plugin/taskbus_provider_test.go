package main

import (
	"strings"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

func TestValidateTaskBusProviderConflict(t *testing.T) {
	tests := []struct {
		name        string
		proxy       string
		provider    string
		expectError bool
	}{
		{name: "standalone with redis", proxy: "0", provider: "redis", expectError: false},
		{name: "delegated with host", proxy: "1", provider: "host", expectError: false},
		{name: "standalone with host conflict", proxy: "0", provider: "host", expectError: true},
		{name: "delegated with redis conflict", proxy: "1", provider: "redis", expectError: true},
		{name: "empty proxy uses standalone default", proxy: "", provider: "redis", expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				EventBridge: &config.EventBridgeConfig{TaskBusProvider: tt.provider},
			}
			err := validateTaskBusProviderConflict(cfg, tt.proxy)
			if tt.expectError && err == nil {
				t.Fatal("期望校验失败，实际成功")
			}
			if !tt.expectError && err != nil {
				t.Fatalf("期望校验成功，实际失败: %v", err)
			}
			if tt.expectError && !strings.Contains(err.Error(), "runtime mode conflict") {
				t.Fatalf("错误信息不符合预期: %v", err)
			}
		})
	}
}
