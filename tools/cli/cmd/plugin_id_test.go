package cmd

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
)

func TestInitRejectsIdentityBeforeWriting(t *testing.T) {
	for _, id := range []string{"com.powerx.plugin.demo", "com..example", "com.example.ai_craft"} {
		destination := filepath.Join(t.TempDir(), "untouched")
		err := runInit([]string{id, "--directory", destination, "--git-init=false", "--init-config=false"})
		var identity *fwmanifest.IdentityError
		if !errors.As(err, &identity) {
			t.Fatal(err)
		}
		if _, err := os.Stat(destination); !os.IsNotExist(err) {
			t.Fatal("INVALID_ID_WROTE_FILES")
		}
	}
}

func TestPluginIDCommand(t *testing.T) {
	for _, id := range []string{"com.powerx.plugins.demo", "com.example.demo"} {
		if err := Execute([]string{"plugin-id", "check", "--id", id}); err != nil {
			t.Fatal(err)
		}
	}
	if Execute([]string{"plugin-id", "check", "--id", "com.powerx.plugin.demo"}) == nil {
		t.Fatal("LEGACY_ID_ACCEPTED")
	}
	file := filepath.Join(t.TempDir(), "plugin.yaml")
	if err := os.WriteFile(file, []byte("id: 'com.example.demo' # quoted YAML\nversion: 1.0.0\nruntime:\n  env:\n    POWERX_PLUGIN_ID: com.example.other\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if Execute([]string{"plugin-id", "check", "--manifest", file}) == nil {
		t.Fatal("MISMATCH_ACCEPTED")
	}
}
