package templates

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const frameworkModule = "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go"

// Package the producer's source as a real Go module proxy artifact. Consumers
// still run with GOWORK=off and no replace, including before a release exists.
func frameworkSnapshot(t *testing.T) (string, string) {
	t.Helper()
	root, err := filepath.Abs("../../../../framework/backend/go")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string][]byte{}
	hash := sha256.New()
	err = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") && path != root {
				return filepath.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		name, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name = filepath.ToSlash(name)
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[name] = raw
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write(raw)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	version := "v0.0.26-dev.g" + hex.EncodeToString(hash.Sum(nil))[:16]
	proxy := t.TempDir()
	escaped := "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go"
	dir := filepath.Join(proxy, escaped, "@v")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	info, _ := json.Marshal(map[string]string{"Version": version, "Time": "2026-10-11T00:00:00Z"})
	for extension, raw := range map[string][]byte{".mod": files["go.mod"], ".info": info} {
		if err := os.WriteFile(filepath.Join(dir, version+extension), raw, 0644); err != nil {
			t.Fatal(err)
		}
	}
	f, err := os.Create(filepath.Join(dir, version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	z := zip.NewWriter(f)
	// Sort the archive so its checksum is deterministic across calls.
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry, err := z.Create(frameworkModule + "@" + version + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(files[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return proxy, version
}
