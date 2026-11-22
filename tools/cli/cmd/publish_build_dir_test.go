package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFindLatestBuildDirSkipsAnIncompletePackage(t *testing.T) {
	root := t.TempDir()
	buildRoot := filepath.Join(root, ".px-plugin", "build")
	legacy := filepath.Join(buildRoot, "20261122T000000Z")
	ready := filepath.Join(buildRoot, "20261122T000000Z-000000001-100")
	incomplete := filepath.Join(buildRoot, "20261122T000000Z-000000002-200")
	for _, dir := range []string{legacy, ready, incomplete} {
		require.NoError(t, os.MkdirAll(dir, 0700))
	}
	for _, dir := range []string{legacy, ready} {
		require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata.json"), []byte("{}"), 0600))
	}
	require.NoError(t, os.WriteFile(filepath.Join(incomplete, "metadata.json.tmp"), []byte("{}"), 0600))

	dir, err := findLatestBuildDir(root)
	require.NoError(t, err)
	require.Equal(t, ready, dir)
}
