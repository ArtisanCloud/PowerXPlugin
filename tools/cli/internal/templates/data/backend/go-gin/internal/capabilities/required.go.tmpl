package capabilities

import (
	"errors"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadRequiredHostCapabilities resolves the same catalog references as the
// capability manager. It never derives grants from all constructed clients.
func LoadRequiredHostCapabilities(path string) ([]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var root map[string]any
	if err := yaml.Unmarshal(raw, &root); err != nil {
		return nil, err
	}
	if err := mergeCatalogReferences(path, root); err != nil {
		return nil, err
	}
	section, ok := root["capabilities"].(map[string]any)
	if !ok {
		return nil, errors.New("FRAMEWORK_CAPABILITY_REQUIREMENTS_INVALID")
	}
	if section["required"] == nil {
		return nil, nil
	}
	values, ok := section["required"].([]any)
	if !ok {
		return nil, errors.New("FRAMEWORK_CAPABILITY_REQUIREMENTS_INVALID")
	}
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		id, ok := value.(string)
		if !ok || id == "" || strings.TrimSpace(id) != id || seen[id] {
			return nil, errors.New("FRAMEWORK_CAPABILITY_REQUIREMENTS_INVALID")
		}
		seen[id] = true
		result = append(result, id)
	}
	return result, nil
}
