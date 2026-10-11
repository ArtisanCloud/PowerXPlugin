package config

// LocalAIConfig describes plugin-owned model execution. It is never populated
// from gateway credentials or selected dynamically by a business request.
type LocalAIConfig struct {
	Models []LocalAIModel `yaml:"models" json:"models"`
	Agents []LocalAgent   `yaml:"agents" json:"agents"`
}

// AICatalogConfig stores the versioned PowerX provider catalog separately from
// the subset of providers that this plugin can execute in local mode.
type AICatalogConfig struct {
	Directory string `yaml:"directory" json:"directory"`
}

type LocalAgent struct {
	UUID     string `yaml:"uuid" json:"uuid"`
	Name     string `yaml:"name" json:"name"`
	ModelKey string `yaml:"model_key" json:"model_key"`
}

type LocalAIModel struct {
	Key            string   `yaml:"key" json:"key"`
	Provider       string   `yaml:"provider" json:"provider"`
	Model          string   `yaml:"model" json:"model"`
	Endpoint       string   `yaml:"endpoint" json:"endpoint"`
	Modalities     []string `yaml:"modalities" json:"modalities"`
	TimeoutSeconds int      `yaml:"timeout_seconds" json:"timeout_seconds"`
}
