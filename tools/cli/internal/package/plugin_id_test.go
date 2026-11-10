package pkg

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
)

func TestPackageEnvironmentIdentityFailsBeforeWriting(t *testing.T) {
	keys := []string{"POWERX_PLUGIN_ID", "POWERX_SECURITY_JWT_AUDIENCE", "NUXT_APP_BASE_URL", "NUXT_PUBLIC_PLUGIN_ID", "NUXT_PUBLIC_POWERX_PLUGIN_ID"}
	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			for _, name := range keys {
				t.Setenv(name, "")
			}
			t.Setenv(key, "com.example.other")
			directory := t.TempDir()
			if err := os.WriteFile(filepath.Join(directory, "plugin.yaml"), []byte("id: com.example.demo\nversion: 1.0.0\n"), 0600); err != nil {
				t.Fatal(err)
			}
			output := filepath.Join(directory, "must-not-exist")
			_, err := NewBuilder().Build(context.Background(), &Options{EntryPath: directory, OutputDir: output, SkipBackend: true, SkipFrontend: true})
			var identity *fwmanifest.IdentityError
			if !errors.As(err, &identity) || identity.Code != "PLUGIN_ID_MISMATCH" {
				t.Fatal(err)
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatal("INVALID_ID_WROTE_PACKAGE")
			}
		})
	}
}
