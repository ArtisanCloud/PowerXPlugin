package capabilities

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadRequiredHostCapabilities(t *testing.T) {
	for _, tc := range []struct {
		yaml    string
		want    int
		invalid bool
	}{
		{"capabilities:\n  required: [cap.a, cap.b]\n", 2, false},
		{"capabilities:\n  required: []\n", 0, false},
		{"capabilities:\n  required: [cap.a, cap.a]\n", 0, true},
		{"capabilities:\n  required: [12]\n", 0, true},
	} {
		dir := t.TempDir()
		manifest := filepath.Join(dir, "plugin.yaml")
		if err := os.WriteFile(manifest, []byte("catalogs:\n  capabilities: caps.yaml\n"), 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "caps.yaml"), []byte(tc.yaml), 0600); err != nil {
			t.Fatal(err)
		}
		ids, err := LoadRequiredHostCapabilities(manifest)
		if (err != nil) != tc.invalid || len(ids) != tc.want {
			t.Fatalf("ids=%v err=%v", ids, err)
		}
	}
}
