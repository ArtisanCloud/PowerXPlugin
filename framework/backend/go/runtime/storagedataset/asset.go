package storagedataset

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"regexp"
	"slices"

	"gopkg.in/yaml.v3"
)

var datasetKey = regexp.MustCompile(`^[a-z][a-z0-9_.-]{0,63}$`)
var relationName = regexp.MustCompile(`^[a-z_][a-z0-9_]{0,62}$`)
var clusterName = regexp.MustCompile(`^[a-zA-Z0-9_.:-]{1,128}$`)

type Retention struct {
	Min       int `json:"minimum_days"`
	Max       int `json:"maximum_days"`
	Suggested int `json:"suggested_days"`
}

type Declaration struct {
	Key                 string            `json:"dataset_key"`
	Labels              map[string]string `json:"display_name_i18n"`
	Version             int64             `json:"contract_version"`
	SchemaVersion       int64             `json:"schema_contract_version"`
	BindingRef          string            `json:"source_binding_ref"`
	TenantScope         string            `json:"tenant_scope"`
	AdapterKey          string            `json:"adapter_key"`
	AdapterVersion      int64             `json:"adapter_version"`
	Actions             []string          `json:"allowed_actions"`
	Retention           *Retention        `json:"retention_bounds,omitempty"`
	ArchiveBeforeDelete bool              `json:"archive_before_delete,omitempty"`
}

type Asset struct {
	declaration Declaration
	digest      string
	tables      []string
}

func (a Asset) Key() string        { return a.declaration.Key }
func (a Asset) BindingRef() string { return a.declaration.BindingRef }

// LoadAssets validates declarations in the immutable installed plugin package.
// Callers supply only the trusted package root.
func LoadAssets(root string) ([]Asset, error) {
	fs, err := os.OpenRoot(root)
	if err != nil {
		return nil, ErrSource
	}
	defer fs.Close()
	raw, err := readAsset(fs, "plugin.yaml", 1<<20)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Datasets map[string]struct {
			Digest string   `yaml:"sha256"`
			Tables []string `yaml:"observation_tables"`
		} `yaml:"data_lifecycle"`
	}
	if yaml.Unmarshal(raw, &manifest) != nil {
		return nil, ErrContract
	}
	keys := make([]string, 0, len(manifest.Datasets))
	for key := range manifest.Datasets {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	assets := make([]Asset, 0, len(keys))
	for _, key := range keys {
		item := manifest.Datasets[key]
		if !datasetKey.MatchString(key) || len(item.Tables) < 1 || len(item.Tables) > 32 {
			return nil, ErrContract
		}
		tables := slices.Clone(item.Tables)
		slices.Sort(tables)
		for i, table := range tables {
			if !relationName.MatchString(table) || i > 0 && tables[i-1] == table {
				return nil, ErrContract
			}
		}
		content, err := readAsset(fs, "contracts/storage-datasets/"+key+".json", 16384)
		if err != nil {
			return nil, err
		}
		var declaration Declaration
		if item.Digest != digest(content) || decodeContract(content, HostVersion, "Declaration", &declaration) != nil || declaration.Key != key {
			return nil, ErrContract
		}
		if declaration.BindingRef != "database" {
			return nil, ErrScope
		}
		if b := declaration.Retention; b != nil && (b.Min > b.Suggested || b.Suggested > b.Max) {
			return nil, ErrContract
		}
		assets = append(assets, Asset{declaration, item.Digest, tables})
	}
	return assets, nil
}

func readAsset(root *os.Root, path string, limit int64) ([]byte, error) {
	f, err := root.Open(path)
	if err != nil {
		return nil, ErrContract
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil || !stat.Mode().IsRegular() || stat.Size() > limit {
		return nil, ErrContract
	}
	raw, err := io.ReadAll(io.LimitReader(f, limit+1))
	if err != nil || int64(len(raw)) > limit {
		return nil, ErrContract
	}
	return raw, nil
}

func digest(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}
