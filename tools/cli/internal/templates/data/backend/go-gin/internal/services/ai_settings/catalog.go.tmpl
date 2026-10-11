package ai_settings

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Catalog struct{ providers map[string]catalogProvider }
type catalogProvider struct {
	ID         string                     `yaml:"id"`
	Name       string                     `yaml:"name"`
	Auth       catalogAuth                `yaml:"auth"`
	Defaults   map[string]string          `yaml:"defaults"`
	Modalities map[string]catalogModality `yaml:"modalities"`
	Apps       map[string]catalogApp      `yaml:"apps"`
}
type catalogAuth struct {
	Scheme   string            `yaml:"scheme"`
	Fields   []string          `yaml:"fields"`
	Defaults map[string]string `yaml:"defaults"`
}
type catalogModality struct {
	Models []catalogModel `yaml:"models"`
}
type catalogApp struct {
	Name       string                     `yaml:"name"`
	Modalities map[string]catalogModality `yaml:"modalities"`
}
type catalogModel struct {
	ID    string `yaml:"id"`
	Label string `yaml:"label"`
}

func LoadCatalog(dir string) (*Catalog, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("AI catalog directory: %w", err)
	}
	c := &Catalog{providers: map[string]catalogProvider{}}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		var item catalogProvider
		if err := yaml.Unmarshal(raw, &item); err != nil {
			return nil, fmt.Errorf("AI catalog %s: %w", entry.Name(), err)
		}
		if strings.TrimSpace(item.ID) != "" {
			c.providers[item.ID] = item
		}
	}
	if len(c.providers) == 0 {
		return nil, fmt.Errorf("AI catalog has no providers")
	}
	return c, nil
}

func (c *Catalog) Providers(modality string) []map[string]any {
	out := []map[string]any{}
	if c == nil {
		return out
	}
	for _, p := range c.providers {
		if _, ok := p.Modalities[modality]; !ok {
			continue
		}
		apps := []map[string]string{}
		for id, app := range p.Apps {
			if _, ok := app.Modalities[modality]; ok {
				apps = append(apps, map[string]string{"id": id, "name": app.Name})
			}
		}
		out = append(out, map[string]any{"id": p.ID, "name": p.Name, "apps": apps, "auth": map[string]any{"scheme": p.Auth.Scheme, "fields": p.Auth.Fields, "defaults": p.Defaults}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	return out
}
func (c *Catalog) Models(modality, provider, app string) []map[string]string {
	if c == nil {
		return []map[string]string{}
	}
	p, ok := c.providers[provider]
	if !ok {
		return []map[string]string{}
	}
	var models []catalogModel
	if app != "" {
		models = p.Apps[app].Modalities[modality].Models
	} else {
		models = p.Modalities[modality].Models
	}
	out := make([]map[string]string, 0, len(models))
	for _, m := range models {
		out = append(out, map[string]string{"id": m.ID, "label": m.Label})
	}
	return out
}

func (c *Catalog) HasModel(modality, provider, app, modelID string) bool {
	for _, model := range c.Models(modality, provider, app) {
		if model["id"] == modelID {
			return true
		}
	}
	return false
}
