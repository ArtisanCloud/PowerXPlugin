package templates

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
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
	cmd := exec.Command("git", "ls-files", "--cached", "--others", "--exclude-standard", "-z", "--", ".")
	cmd.Dir = root
	listed, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	inputs := strings.Split(string(listed), "\x00")
	sort.Strings(inputs)
	for _, name := range inputs {
		if name == "" {
			continue
		}
		path := filepath.Join(root, filepath.FromSlash(name))
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		files[name] = raw
		hash.Write([]byte(name))
		hash.Write([]byte{0})
		hash.Write(raw)
	}
	version := "v0.0.27-dev.g" + hex.EncodeToString(hash.Sum(nil))[:16]
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

func TestFrameworkSnapshotIncludesGitInputs(t *testing.T) {
	proxy, version := frameworkSnapshot(t)
	archive, err := zip.OpenReader(filepath.Join(proxy, "github.com/!artisan!cloud/!power!x!plugin/framework/backend/go/@v", version+".zip"))
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	found := false
	for _, file := range archive.File {
		if strings.HasSuffix(file.Name, "/coverage.out") {
			t.Fatal("FRAMEWORK_COVERAGE_ARTIFACT_NOT_ALLOWED")
		}
		if strings.HasSuffix(file.Name, "/iam/contracts/.gitkeep") {
			found = true
		}
	}
	if !found {
		t.Fatal("FRAMEWORK_GIT_INPUT_MISSING")
	}
}
