package pkg

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuilderPackagesOnlyIndexedDatasetDeclarations(t *testing.T) {
	const name = "contracts/storage-datasets/reports.json"
	const declaration = "{\n  \"dataset_key\": \"reports\"\n}\n"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(declaration)))
	root := datasetPackageFixture(t, "reports", sum)
	writeFile(t, filepath.Join(root, name), declaration)
	writeFile(t, filepath.Join(root, "contracts/storage-datasets/undeclared.json"), "{}")

	result, err := NewBuilder().Build(context.Background(), datasetPackageOptions(root))
	require.NoError(t, err)
	require.Equal(t, map[string]string{name: declaration}, packagedDatasetDeclarations(t, result.PackagePath))

	raw, err := os.ReadFile(result.MetadataPath)
	require.NoError(t, err)
	var metadata Metadata
	require.NoError(t, json.Unmarshal(raw, &metadata))
	require.Contains(t, metadata.Artifacts, Artifact{
		Name: name, Path: "payload/" + name, Size: int64(len(declaration)), Hash: sum,
	})
}

func TestBuilderIsolatesPackagesWithTheSameTimestamp(t *testing.T) {
	const first = "contracts/storage-datasets/reports.json"
	const second = "contracts/storage-datasets/other.json"
	const declaration = "{}\n"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(declaration)))
	root := datasetPackageFixture(t, "reports", sum)
	writeFile(t, filepath.Join(root, first), declaration)
	builder := NewBuilder(WithClock(func() time.Time { return time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC) }))
	options := datasetPackageOptions(root)

	a, err := builder.Build(context.Background(), options)
	require.NoError(t, err)
	writeFile(t, filepath.Join(root, second), declaration)
	writeFile(t, filepath.Join(root, "plugin.yaml"), datasetPackageManifest("other", sum))
	b, err := builder.Build(context.Background(), options)
	require.NoError(t, err)

	require.NotEqual(t, a.BuildDir, b.BuildDir)
	require.Equal(t, map[string]string{first: declaration}, packagedDatasetDeclarations(t, a.PackagePath))
	require.Equal(t, map[string]string{second: declaration}, packagedDatasetDeclarations(t, b.PackagePath))
}

func packagedDatasetDeclarations(t *testing.T, path string) map[string]string {
	t.Helper()
	file, err := os.Open(path)
	require.NoError(t, err)
	defer file.Close()
	reader, err := gzip.NewReader(file)
	require.NoError(t, err)
	defer reader.Close()
	archive := tar.NewReader(reader)
	declarations := make(map[string]string)
	for {
		header, err := archive.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		if header.Typeflag == tar.TypeReg && strings.HasPrefix(header.Name, "contracts/storage-datasets/") {
			raw, err := io.ReadAll(archive)
			require.NoError(t, err)
			declarations[header.Name] = string(raw)
		}
	}
	return declarations
}

func TestBuilderRejectsInvalidIndexedDatasetDeclarations(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		target string
		code   string
	}{
		{name: "missing", key: "reports", code: "DATASET_ASSET_INVALID"},
		{name: "digest mismatch", key: "reports", target: "file", code: "DATASET_ASSET_DIGEST_MISMATCH"},
		{name: "symlink", key: "reports", target: "symlink", code: "DATASET_ASSET_INVALID"},
		{name: "escaping directory", key: "reports", target: "directory symlink", code: "DATASET_ASSET_INVALID"},
		{name: "invalid key", key: "../../reports", code: "DATASET_ASSET_KEY_INVALID"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			root := datasetPackageFixture(t, tc.key, strings.Repeat("0", 64))
			dir := filepath.Join(root, "contracts/storage-datasets")
			switch tc.target {
			case "file":
				writeFile(t, filepath.Join(dir, "reports.json"), "{}")
			case "symlink":
				outside := filepath.Join(t.TempDir(), "reports.json")
				writeFile(t, outside, "{}")
				mkDir(t, dir)
				require.NoError(t, os.Symlink(outside, filepath.Join(dir, "reports.json")))
			case "directory symlink":
				outside := t.TempDir()
				writeFile(t, filepath.Join(outside, "reports.json"), "{}")
				mkDir(t, filepath.Dir(dir))
				require.NoError(t, os.Symlink(outside, dir))
			}
			result, err := NewBuilder().Build(context.Background(), datasetPackageOptions(root))
			require.ErrorContains(t, err, tc.code)
			require.Nil(t, result)
			entries, err := os.ReadDir(filepath.Join(root, ".px-plugin", "build"))
			require.NoError(t, err)
			require.Empty(t, entries)
		})
	}
}

func datasetPackageFixture(t *testing.T, key, sum string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "plugin.yaml"), datasetPackageManifest(key, sum))
	writeFile(t, filepath.Join(root, "docs/contracts/manifest.json"), "{}")
	writeFile(t, filepath.Join(root, "docs/contracts/rbac.json"), "{}")
	return root
}

func datasetPackageManifest(key, sum string) string {
	return fmt.Sprintf("id: com.example.demo\nversion: 0.1.0\ndata_lifecycle:\n  %q:\n    sha256: %q\n    observation_tables: [reports]\n", key, sum)
}

func datasetPackageOptions(root string) *Options {
	return &Options{EntryPath: root, SkipFrontend: true, SkipBackend: true}
}
