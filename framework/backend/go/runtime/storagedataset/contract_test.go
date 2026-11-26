package storagedataset

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAssetsChecksManifestDigestAndPaths(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "contracts", "storage-datasets")
	require.NoError(t, os.MkdirAll(dir, 0700))
	declaration := []byte(`{"dataset_key":"case_report","display_name_i18n":{"en":"Case report","zh-CN":"案例报告"},"contract_version":1,"schema_contract_version":1,"tenant_scope":"tenant","source_binding_ref":"database","adapter_key":"fixture","adapter_version":1,"allowed_actions":["observe","preview","delete","archive"],"retention_bounds":{"minimum_days":7,"suggested_days":30,"maximum_days":90},"archive_before_delete":true}`)
	assetFile := filepath.Join(dir, "case_report.json")
	index := func(key, sum, members string) {
		raw := fmt.Sprintf("id: com.example.fixture\ndata_lifecycle:\n  %s:\n    sha256: '%s'\n    observation_tables: %s\n", key, sum, members)
		require.NoError(t, os.WriteFile(filepath.Join(root, "plugin.yaml"), []byte(raw), 0600))
	}
	require.NoError(t, os.WriteFile(assetFile, declaration, 0600))
	index("case_report", digest(declaration), "[reports, reference_items]")
	assets, err := LoadAssets(root)
	require.NoError(t, err)
	require.Len(t, assets, 1)
	require.Equal(t, "case_report", assets[0].Key())
	require.Equal(t, []string{"reference_items", "reports"}, assets[0].tables)
	asset := assets[0]
	scope := Scope{SourceBindingUUID: "binding", DefinitionDigest: asset.digest, SchemaVersion: 1, AdapterKey: "fixture", AdapterVersion: 1}
	ds := Dataset{Asset: asset, Binding: Binding{UUID: "binding"}}
	require.True(t, ds.match(scope, "archive"))
	require.False(t, ds.match(scope, "delete"))

	index("case_report", strings.Repeat("0", 64), "[reports]")
	_, err = LoadAssets(root)
	require.ErrorIs(t, err, ErrContract)
	index("case_report", digest(declaration), "[reports, reports]")
	_, err = LoadAssets(root)
	require.ErrorIs(t, err, ErrContract)
	index("../case_report", digest(declaration), "[reports]")
	_, err = LoadAssets(root)
	require.ErrorIs(t, err, ErrContract)

	index("case_report", digest(declaration), "[reports]")
	external := filepath.Join(t.TempDir(), "outside.json")
	require.NoError(t, os.WriteFile(external, declaration, 0600))
	require.NoError(t, os.Remove(assetFile))
	require.NoError(t, os.Symlink(external, assetFile))
	_, err = LoadAssets(root)
	require.ErrorIs(t, err, ErrContract)
}

func TestRejectAmbiguousAndOversizedWireValues(t *testing.T) {
	const validID = "a71beed0-cdb0-4d7c-9b31-7931e9eb4466"
	valid := `{"contract_version":"powerx.storage-dataset-host/v1","request_uuid":"` + validID + `"}`
	require.NoError(t, decodeContract([]byte(valid), HostVersion, "PreviewTaskNotice", nil))
	for _, bad := range []string{
		strings.TrimSuffix(valid, "}") + `,"request_uuid":"` + validID + `"}`,
		strings.TrimSuffix(valid, "}") + `,"source_binding_uuid":"` + validID + `"}`,
		valid + valid,
	} {
		require.ErrorIs(t, decodeContract([]byte(bad), HostVersion, "PreviewTaskNotice", nil), ErrContract)
	}
	require.NoError(t, validateArchive([]byte("{\"x\":1}\n"), 1))
	require.ErrorIs(t, validateArchive([]byte("{\"x\":1}"), 1), ErrContract)
	require.ErrorIs(t, validateArchive([]byte("[1]\n"), 1), ErrContract)
	require.ErrorIs(t, validateArchive([]byte("{\"x\":1}\n"), 2), ErrContract)
	require.ErrorIs(t, validateArchive(make([]byte, MaxArchiveBytes+1), 1), ErrContract)
}
