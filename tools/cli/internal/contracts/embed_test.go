package contracts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilesExportsPinnedUpgradeErrorFixture(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)
	exported := map[string][]byte{}
	for _, file := range files {
		require.NotContains(t, exported, file.Name)
		exported[file.Name] = file.Data
	}
	for _, name := range []string{"ai-settings-llm-catalog.v1.md", "fixtures/ai-settings-catalog.v1.json"} {
		canonical, err := os.ReadFile(filepath.Join("../../../../docs/contracts", name))
		require.NoError(t, err)
		require.Equal(t, canonical, exported["docs/contracts/"+name])
	}
	var source struct {
		SHA256 map[string]string `json:"sha256"`
	}
	require.NoError(t, json.Unmarshal(exported["docs/contracts/ai-media-v2-source.json"], &source))
	require.Len(t, source.SHA256, 4)
	require.Contains(t, source.SHA256, "fixtures/capability-version-upgrade-errors.v1.json")
	for name, expected := range source.SHA256 {
		raw, ok := exported["docs/contracts/"+name]
		require.True(t, ok, name)
		sum := sha256.Sum256(raw)
		require.Equal(t, expected, hex.EncodeToString(sum[:]), name)
		canonical, err := os.ReadFile(filepath.Join("../../../../docs/contracts", name))
		require.NoError(t, err)
		require.Equal(t, canonical, raw)
	}
}
