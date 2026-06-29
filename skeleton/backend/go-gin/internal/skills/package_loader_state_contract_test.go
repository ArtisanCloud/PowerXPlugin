package skills

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadTemplatePackageKeepsStateContract(t *testing.T) {
	pkg, err := LoadPackageAt(filepath.Join("..", "..", "..", "..", "skills", "template"))
	require.NoError(t, err)

	require.Equal(t, "com.powerx.plugins.base.template.prepare", pkg.Manifest.Executor.PrepareCapability)
	require.Equal(t, "com.powerx.plugins.base.template.create", pkg.Manifest.Executor.ActionMap["create"])

	contract := pkg.Manifest.StateContract
	require.NotEmpty(t, contract)
	require.Equal(t, "1.0", contract["schema_version"])
	stateKeys, ok := contract["state_keys"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, stateKeys, "template.create")
	require.Contains(t, stateKeys, "template.update")
}
