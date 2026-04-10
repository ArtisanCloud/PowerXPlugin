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

	t.Run("rbac catalog conflicts with top-level permissions", func(t *testing.T) {
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
