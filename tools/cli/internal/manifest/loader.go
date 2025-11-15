package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// PluginManifest represents the minimal fields needed from plugin.yaml.
type PluginManifest struct {
	ID      string `yaml:"id"`
	Version string `yaml:"version"`
	Backend struct {
		Entry string `yaml:"entry"`
	} `yaml:"backend"`
}

// Load reads plugin.yaml from the provided directory (or absolute path).
func Load(path string) (*PluginManifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("manifest stat: %w", err)
	}

	file := path
	if info.IsDir() {
		file = filepath.Join(path, "plugin.yaml")
	}

	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("decode manifest: %w", err)
	}

	if manifest.ID == "" || manifest.Version == "" {
		return nil, fmt.Errorf("manifest missing id/version")
	}

	return &manifest, nil
}
