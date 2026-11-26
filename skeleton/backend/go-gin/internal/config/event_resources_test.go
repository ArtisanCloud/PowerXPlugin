package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const declaredEventFixture = `version: v1
topics:
  - key: fixture.changed
    namespace: com.example.fixture.events
    name: changed
    acl:
      - principal_type: plugin
        principal_id: com.example.fixture
        actions: [publish]
`

func eventResourceFixture(t *testing.T, manifest string) (*Config, string) {
	t.Helper()
	root := t.TempDir()
	t.Setenv("POWERX_PLUGIN_ROOT", root)
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte("id: com.example.fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "events.yaml"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	return &Config{EventBridge: &EventBridgeConfig{TopicManifest: "events.yaml"}}, root
}

func TestEventResourcesLoadOnlyDeclaredKeys(t *testing.T) {
	cfg, root := eventResourceFixture(t, declaredEventFixture)
	if err := cfg.loadEventBridgeResources(root); err != nil {
		t.Fatal(err)
	}
	if cfg.EventBridge.SourcePlugin != "com.example.fixture" || len(cfg.EventBridge.PublishTopicKeys) != 1 || cfg.EventBridge.PublishTopicKeys[0] != "fixture.changed" || len(cfg.EventBridge.TopicBindings) != 0 {
		t.Fatal("RESOURCE_BINDING_FABRICATED")
	}
}

func TestEventResourcesAcceptCoreDeliveryPolicies(t *testing.T) {
	raw := strings.Replace(declaredEventFixture, "topics:\n", "defaults:\n  payload_format: json\n  ack_timeout_seconds: 30\n  metadata: {fixture: true}\ntopics:\n", 1)
	raw = strings.Replace(raw, "    name: changed", "    name: changed\n    payload_format: json\n    max_retry: 3\n    retention_policy: {ttl: 60}", 1)
	cfg, root := eventResourceFixture(t, raw)
	if err := cfg.loadEventBridgeResources(root); err != nil {
		t.Fatal(err)
	}
}

func TestEventResourcesAcceptCoreV1PluginPrincipalAndOwnedNamespace(t *testing.T) {
	raw := strings.ReplaceAll(declaredEventFixture, "com.example.fixture", "com.powerx.plugins.fixture")
	raw = strings.Replace(raw, "version: v1", "version: 1", 1)
	raw = strings.Replace(raw, "namespace: com.powerx.plugins.fixture.events", "namespace: powerx.fixture.events", 1)
	raw = strings.Replace(raw, "principal_id: com.powerx.plugins.fixture", "principal_id: '{{plugin_id}}'", 1)
	cfg, root := eventResourceFixture(t, raw)
	if err := os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte("id: com.powerx.plugins.fixture\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := cfg.loadEventBridgeResources(root); err != nil {
		t.Fatal(err)
	}
	if len(cfg.EventBridge.PublishTopicKeys) != 1 || len(cfg.EventBridge.TopicBindings) != 0 {
		t.Fatal("CORE_V1_DECLARATION_INVALID")
	}
}

func TestEventResourcesRejectInvalidDeclarations(t *testing.T) {
	for _, raw := range []string{
		strings.ReplaceAll(declaredEventFixture, "namespace: com.example.fixture.events", "namespace: com.example.foreign.events"),
		strings.ReplaceAll(declaredEventFixture, "principal_id: com.example.fixture", "principal_id: com.example.foreign"),
		declaredEventFixture + "---\nversion: v1\n",
		declaredEventFixture + strings.Repeat(" ", 256*1024),
	} {
		cfg, root := eventResourceFixture(t, raw)
		if err := cfg.loadEventBridgeResources(root); !errors.Is(err, ErrEventResourceInvalid) {
			t.Fatal("INVALID_DECLARATION_ACCEPTED", err)
		}
	}
}

func TestEventResourcesRejectSymlinkEscapeAndMixedKeys(t *testing.T) {
	cfg, root := eventResourceFixture(t, declaredEventFixture)
	cfg.EventBridge.PublishTopicKeys = []string{"another.key"}
	if err := cfg.loadEventBridgeResources(root); !errors.Is(err, ErrEventResourceInvalid) {
		t.Fatal("MIXED_KEYS_ACCEPTED", err)
	}
	cfg.EventBridge.PublishTopicKeys = nil
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outside, []byte(declaredEventFixture), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.yaml")); err != nil {
		t.Fatal(err)
	}
	cfg.EventBridge.TopicManifest = "escape.yaml"
	if err := cfg.loadEventBridgeResources(root); !errors.Is(err, ErrEventResourceInvalid) {
		t.Fatal("MANIFEST_ESCAPE_ACCEPTED", err)
	}
}
