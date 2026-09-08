package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateCatalogConflicts(t *testing.T) {
	t.Run("events catalog conflicts with top-level events", func(t *testing.T) {
		plugin := map[string]interface{}{
			"catalogs": map[string]interface{}{
				"events": "./plugin.d/events.yaml",
			},
			"events": map[string]interface{}{
				"topics": []interface{}{},
			},
		}
		err := validateCatalogConflicts(plugin)
		if err == nil {
			t.Fatal("expected conflict error, got nil")
		}
		want := `catalog conflict on field "events" (catalog=events)`
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err.Error())
		}
		if !strings.Contains(err.Error(), "remove top-level events and keep plugin.d/events.yaml only") {
			t.Fatalf("expected events remediation hint, got %q", err.Error())
		}
	})

	t.Run("rbac catalog conflicts with top-level permission declarations", func(t *testing.T) {
		plugin := map[string]interface{}{
			"catalogs": map[string]interface{}{
				"rbac": "./plugin.d/rbac.yaml",
			},
			"permissions": []interface{}{},
		}
		err := validateCatalogConflicts(plugin)
		if err == nil {
			t.Fatal("expected conflict error, got nil")
		}
		want := `catalog conflict on field "permissions" (catalog=rbac)`
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("expected error to contain %q, got %q", want, err.Error())
		}
	})

	t.Run("no conflict when top-level field is absent", func(t *testing.T) {
		plugin := map[string]interface{}{
			"catalogs": map[string]interface{}{
				"events": "./plugin.d/events.yaml",
			},
		}
		if err := validateCatalogConflicts(plugin); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})
}

func TestCatalogConflictDetectedBeforeMergeFromFiles(t *testing.T) {
	tmpDir := t.TempDir()
	pluginDDir := filepath.Join(tmpDir, "plugin.d")
	if err := os.MkdirAll(pluginDDir, 0o755); err != nil {
		t.Fatalf("mkdir plugin.d: %v", err)
	}

	eventsCatalogPath := filepath.Join(pluginDDir, "events.yaml")
	eventsCatalog := "events:\n  topics:\n    - key: com.powerx.demo.topic\n"
	if err := os.WriteFile(eventsCatalogPath, []byte(eventsCatalog), 0o644); err != nil {
		t.Fatalf("write events catalog: %v", err)
	}

	pluginPath := filepath.Join(tmpDir, "plugin.yaml")
	pluginContent := strings.Join([]string{
		"id: com.powerx.plugins.demo",
		"name: demo",
		"version: 0.1.0",
		"catalogs:",
		"  events: ./plugin.d/events.yaml",
		"events:",
		"  topics:",
		"    - key: com.powerx.demo.legacy",
		"",
	}, "\n")
	if err := os.WriteFile(pluginPath, []byte(pluginContent), 0o644); err != nil {
		t.Fatalf("write plugin yaml: %v", err)
	}

	pluginMap, err := loadYAMLFile(pluginPath)
	if err != nil {
		t.Fatalf("load plugin yaml: %v", err)
	}

	err = validateCatalogConflicts(pluginMap)
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}
	if !strings.Contains(err.Error(), `catalog conflict on field "events" (catalog=events)`) {
		t.Fatalf("expected events conflict, got %q", err.Error())
	}
	if !strings.Contains(err.Error(), "remove top-level events and keep plugin.d/events.yaml only") {
		t.Fatalf("expected remediation hint, got %q", err.Error())
	}
}

func TestValidateRequiredHostCapabilities(t *testing.T) {
	required := []string{"com.corex.capabilities.grant_status.read", "com.corex.metadata.dictionary.read"}

	plugin := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"required": stringSliceToInterfaces(required),
		},
	}
	if err := validateRequiredHostCapabilities(plugin); err != nil {
		t.Fatalf("explicit read-only requirements rejected: %v", err)
	}

	missing := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"required": stringSliceToInterfaces(required[1:]),
		},
	}
	err := validateRequiredHostCapabilities(missing)
	if err == nil {
		t.Fatal("missing grant-status capability accepted")
	}
	if !strings.Contains(err.Error(), "com.corex.capabilities.grant_status.read") {
		t.Fatalf("expected missing capability in error, got %q", err.Error())
	}

	malformed := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"required": []interface{}{123},
		},
	}
	err = validateRequiredHostCapabilities(malformed)
	if err == nil {
		t.Fatal("expected malformed required capability entry to fail")
	}
	if !strings.Contains(err.Error(), "capabilities.required[0]") {
		t.Fatalf("expected invalid entry location in error, got %q", err.Error())
	}
}

func TestRequiredCapabilitiesAreExplicitAndStrict(t *testing.T) {
	for _, tc := range []struct {
		ids     []interface{}
		invalid bool
	}{
		{[]interface{}{}, false},
		{[]interface{}{"com.corex.capabilities.grant_status.read"}, false},
		{[]interface{}{"com.corex.capabilities.grant_status.read", "com.example.plugin.read"}, false},
		{[]interface{}{"com.corex.capabilities.grant_status.read", "com.corex.capabilities.grant_status.read"}, true},
		{[]interface{}{" com.corex.capabilities.grant_status.read"}, true},
	} {
		err := validateRequiredHostCapabilities(map[string]interface{}{"capabilities": map[string]interface{}{"required": tc.ids}})
		if (err != nil) != tc.invalid {
			t.Fatalf("ids=%v err=%v", tc.ids, err)
		}
	}
}

func TestShouldLoadManifest(t *testing.T) {
	cases := []struct {
		name             string
		manifestPath     string
		capabilitiesOnly bool
		pluginOnly       bool
		want             bool
	}{
		{name: "full validation", manifestPath: "manifest.yaml", want: true},
		{name: "capabilities only", manifestPath: "manifest.yaml", capabilitiesOnly: true, want: false},
		{name: "plugin only", manifestPath: "manifest.yaml", pluginOnly: true, want: false},
		{name: "no manifest", want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldLoadManifest(tc.manifestPath, tc.capabilitiesOnly, tc.pluginOnly); got != tc.want {
				t.Fatalf("shouldLoadManifest() = %v, want %v", got, tc.want)
			}
		})
	}
}

func stringSliceToInterfaces(values []string) []interface{} {
	result := make([]interface{}, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func TestValidateEventTopicsRequiresPluginPublishACL(t *testing.T) {
	plugin := map[string]interface{}{
		"id": "com.powerx.plugins.demo",
		"events": map[string]interface{}{
			"topics": []interface{}{
				map[string]interface{}{
					"key":     "powerx.runtime.scheduler.triggered.v1",
					"actions": []interface{}{"publish", "subscribe"},
				},
			},
		},
	}

	t.Run("accepts plugin publish acl", func(t *testing.T) {
		eventFabric := map[string]interface{}{
			"topics": []interface{}{
				map[string]interface{}{
					"key": "powerx.runtime.scheduler.triggered.v1",
					"acl": []interface{}{
						map[string]interface{}{
							"principal_type": "member",
							"principal_id":   "member:system",
							"actions":        []interface{}{"publish", "subscribe"},
						},
						map[string]interface{}{
							"principal_type": "plugin",
							"principal_id":   "plugin:com.powerx.plugins.demo",
							"actions":        []interface{}{"publish"},
						},
					},
				},
			},
		}

		if err := validateEventTopics(plugin, eventFabric); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	t.Run("rejects member or role only acl", func(t *testing.T) {
		eventFabric := map[string]interface{}{
			"topics": []interface{}{
				map[string]interface{}{
					"key": "powerx.runtime.scheduler.triggered.v1",
					"acl": []interface{}{
						map[string]interface{}{
							"principal_type": "member",
							"principal_id":   "member:system",
							"actions":        []interface{}{"publish", "subscribe"},
						},
						map[string]interface{}{
							"principal_type": "role",
							"principal_id":   "role:role_admin",
							"actions":        []interface{}{"publish"},
						},
					},
				},
			},
		}

		err := validateEventTopics(plugin, eventFabric)
		if err == nil {
			t.Fatal("expected missing plugin publish ACL error, got nil")
		}
		if !strings.Contains(err.Error(), "缺少插件服务态 publish ACL (plugin:com.powerx.plugins.demo)") {
			t.Fatalf("expected plugin principal ACL error, got %q", err.Error())
		}
	})

	t.Run("rejects plugin acl without publish", func(t *testing.T) {
		eventFabric := map[string]interface{}{
			"topics": []interface{}{
				map[string]interface{}{
					"key": "powerx.runtime.scheduler.triggered.v1",
					"acl": []interface{}{
						map[string]interface{}{
							"principal_type": "plugin",
							"principal_id":   "plugin:com.powerx.plugins.demo",
							"actions":        []interface{}{"subscribe"},
						},
					},
				},
			},
		}

		err := validateEventTopics(plugin, eventFabric)
		if err == nil {
			t.Fatal("expected missing plugin publish ACL error, got nil")
		}
		if !strings.Contains(err.Error(), "powerx.runtime.scheduler.triggered.v1") {
			t.Fatalf("expected topic in error, got %q", err.Error())
		}
	})
}
