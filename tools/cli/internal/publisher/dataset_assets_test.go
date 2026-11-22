//go:build darwin || linux

package publisher

import (
	"archive/zip"
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/powerx-plugin/cli/internal/datasetasset"
	"github.com/stretchr/testify/require"
)

func TestBuildRejectsMissingOrChangedDatasetDeclaration(t *testing.T) {
	const declaration = "{\n  \"dataset_key\": \"reports\"\n}\n"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(declaration)))
	for _, tc := range []struct {
		name   string
		body   string
		reason string
	}{
		{name: "missing", reason: "DATASET_ASSET_INVALID"},
		{name: "digest_mismatch", body: "{}\n", reason: "DATASET_ASSET_DIGEST_MISMATCH"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, e, p := buildFixture(t)
			writeBuildDatasetIndex(t, root, sum)
			if tc.body != "" {
				writeBuildDatasetDeclaration(t, root, tc.body)
			}

			b := runFixture(t, e, p)
			require.Equal(t, "failed", b.Stage)
			require.NotNil(t, b.Error)
			require.Equal(t, ArtifactInvalid, b.Error.Code)
			require.Equal(t, tc.reason, b.Error.Params["reason"])
			require.Nil(t, b.Artifact)
			work := filepath.Join(e.work.path, b.BuildID)
			require.NoFileExists(t, filepath.Join(work, "artifact.zip"))
			stages, err := filepath.Glob(filepath.Join(work, "staging-*"))
			require.NoError(t, err)
			require.Empty(t, stages)
		})
	}
}

func TestBuildFreezesOnlyDeclaredDatasetAssets(t *testing.T) {
	root, e, p := buildFixture(t)
	const declaration = "{\n  \"dataset_key\": \"reports\"\n}\n"
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte(declaration)))
	writeBuildDatasetIndex(t, root, sum)
	writeBuildDatasetDeclaration(t, root, declaration)

	first := runFixture(t, e, p)
	requireReady(t, e, first)
	require.Equal(t, map[string]string{"reports.json": declaration}, buildDatasetDeclarations(t, e, first))

	// Keep the old declaration on disk while removing it from the next manifest.
	writeBuildDatasetIndex(t, root, "")
	second := runFixture(t, e, p)
	requireReady(t, e, second)
	require.NotEqual(t, first.BuildID, second.BuildID)
	require.Empty(t, buildDatasetDeclarations(t, e, second))
	require.Equal(t, map[string]string{"reports.json": declaration}, buildDatasetDeclarations(t, e, first))
}

func writeBuildDatasetIndex(t *testing.T, root, sum string) {
	t.Helper()
	// The fixture builds plugin.yaml from scratch, so pass the same index into the dist manifest.
	script := filepath.Join(root, "scripts/dist.mjs")
	raw, err := os.ReadFile(script)
	require.NoError(t, err)
	const marker = "\n// test dataset index\n"
	original, _, _ := strings.Cut(string(raw), marker)
	if sum != "" {
		index := fmt.Sprintf("data_lifecycle:\n  reports:\n    sha256: %q\n    observation_tables: [reports]\n", sum)
		original += marker + fmt.Sprintf("fs.appendFileSync(path.join(dist,'plugin.yaml'), %q);\n", index)
	}
	require.NoError(t, os.WriteFile(script, []byte(original), 0600))
}

func writeBuildDatasetDeclaration(t *testing.T, root, declaration string) {
	t.Helper()
	dir := filepath.Join(root, filepath.FromSlash(datasetasset.Directory))
	require.NoError(t, os.MkdirAll(dir, 0700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "reports.json"), []byte(declaration), 0600))
}

func buildDatasetDeclarations(t *testing.T, e *Engine, b Build) map[string]string {
	t.Helper()
	artifact, err := e.VerifyArtifact(b.BuildID)
	require.NoError(t, err)
	archive, err := zip.OpenReader(filepath.Join(e.state.path, artifact.Path))
	require.NoError(t, err)
	defer archive.Close()
	files := make(map[string]string)
	prefix := b.Version + "/" + datasetasset.Directory + "/"
	for _, file := range archive.File {
		if !strings.HasPrefix(file.Name, prefix) || file.FileInfo().IsDir() {
			continue
		}
		data, err := fs.ReadFile(archive, file.Name)
		require.NoError(t, err)
		files[strings.TrimPrefix(file.Name, prefix)] = string(data)
	}
	return files
}
