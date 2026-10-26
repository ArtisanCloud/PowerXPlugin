package pkg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	fwmanifest "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/manifest"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func appendNotificationBindingFixture(t *testing.T, root string) {
	t.Helper()
	fixture, err := os.ReadFile("../../../../docs/contracts/fixtures/notification-presentations.v2.yaml")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "plugin.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, fixture...), 0600); err != nil {
		t.Fatal(err)
	}
}
func assertPackagedNotificationBindings(t *testing.T, path string) {
	t.Helper()
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		if header.Name != "plugin.yaml" {
			continue
		}
		raw, err := io.ReadAll(tarReader)
		if err != nil {
			t.Fatal(err)
		}
		var doc map[string]any
		if err := yaml.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		bindings, err := fwmanifest.DecodeNotificationPresentations(doc["notification_presentations"])
		if err != nil || len(bindings) != 4 || bindings[0].BusinessType != "sample_track" || bindings[1].BusinessType != "bulk_order" {
			t.Fatal("PACKAGED_PRESENTATION_BINDINGS_LOST")
		}
		fixture, err := os.ReadFile("../../../../docs/contracts/fixtures/notification-presentations.v2.yaml")
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(raw, fixture) {
			t.Fatal("PACKAGED_MANIFEST_CHANGED_BINDINGS")
		}
		return
	}
	t.Fatal("PACKAGED_PLUGIN_YAML_MISSING")
}
