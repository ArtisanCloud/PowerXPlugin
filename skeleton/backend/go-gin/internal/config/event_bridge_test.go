package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestEventBridgeRejectsRemovedRoutingFields(t *testing.T) {
	for _, field := range []string{"mode: local", "taskbus_provider: redis", "fallback_to_local: false"} {
		for _, prefix := range []string{"event_bridge:\n", "runtime:\n  event_bridge:\n"} {
			indent := "  "
			if prefix[0] == 'r' {
				indent = "    "
			}
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "config.yaml"), []byte(prefix+indent+field+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			t.Setenv("CONFIG_PATH", directory)
			if _, err := Load(); !errors.Is(err, ErrEventBridgeRoutingRemoved) {
				t.Fatal("REMOVED_ROUTING_ACCEPTED_OR_DEFAULTED", field, err)
			}
		}
	}
}

func TestEventBridgeRejectsRoutingFromYAMLMerge(t *testing.T) {
	var cfg Config
	err := yaml.Unmarshal([]byte("routing: &routing\n  mode: local\nruntime:\n  event_bridge:\n    <<: *routing\n    enabled: true\n"), &cfg)
	if !errors.Is(err, ErrEventBridgeRoutingRemoved) {
		t.Fatal("MERGED_ROUTING_ACCEPTED", err)
	}
}
