package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
)

func TestPluginIDCannotBeBypassedByModes(t *testing.T) {
	for _, mode := range []struct{ capability, plugin, sync, syncOnly bool }{{}, {capability: true}, {plugin: true}, {sync: true}, {sync: true, syncOnly: true}} {
		directory := t.TempDir()
		file := filepath.Join(directory, "plugin.yaml")
		if err := os.WriteFile(file, []byte("id: com.powerx.plugin.legacy\n"), 0600); err != nil {
			t.Fatal(err)
		}
		err := run(file, "", "", "", mode.capability, mode.plugin, mode.sync, mode.syncOnly)
		var identity *fwmanifest.IdentityError
		if !errors.As(err, &identity) || identity.Code != "PLUGIN_ID_LEGACY_PREFIX" {
			t.Fatal(err)
		}
		entries, err := os.ReadDir(directory)
		if err != nil || len(entries) != 1 {
			t.Fatal("INVALID_ID_MUTATED_CATALOG", err)
		}
	}
}
