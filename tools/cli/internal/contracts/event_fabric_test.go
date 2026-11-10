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

func TestFilesExportsEventFabricContractProtoAndFixture(t *testing.T) {
	files, err := Files()
	require.NoError(t, err)
	exported := map[string][]byte{}
	for _, file := range files {
		exported[file.Name] = file.Data
	}
	var source struct {
		SHA256  map[string]string `json:"sha256"`
		Fixture struct {
			File   string `json:"file"`
			SHA256 string `json:"sha256"`
		} `json:"framework_fixture"`
	}
	require.NoError(t, json.Unmarshal(exported["docs/contracts/event-fabric-service-source.json"], &source))
	require.Len(t, source.SHA256, 4)
	source.SHA256[source.Fixture.File] = source.Fixture.SHA256
	for name, digest := range source.SHA256 {
		raw, ok := exported["docs/contracts/"+name]
		require.True(t, ok, name)
		sum := sha256.Sum256(raw)
		require.Equal(t, digest, hex.EncodeToString(sum[:]), name)
		canonical, err := os.ReadFile(filepath.Join("../../../../docs/contracts", name))
		require.NoError(t, err)
		require.Equal(t, canonical, raw, name)
	}
	require.NotEmpty(t, exported["docs/contracts/event-fabric-framework.md"])
	require.NotEmpty(t, exported["docs/contracts/plugin-event-fabric-install.md"])
}
