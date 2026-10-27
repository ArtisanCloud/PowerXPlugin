package config

import (
	"os"
	"path/filepath"
	"strings"
)

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

// defaultAICatalogDirectory 在未配置 ai_catalog 时回退查找 provider catalog。
// 安装态由宿主注入 POWERX_PLUGIN_ROOT，dist 包布局为 backend/bin/plugin + backend/etc/ai/providers.d。
func defaultAICatalogDirectory() string {
	candidates := make([]string, 0, 2)
	if root := strings.TrimSpace(os.Getenv("POWERX_PLUGIN_ROOT")); root != "" {
		candidates = append(candidates, filepath.Join(root, "backend", "etc", "ai", "providers.d"))
	}
	if exe, err := os.Executable(); err == nil {
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		candidates = append(candidates, filepath.Join(filepath.Dir(filepath.Dir(exe)), "etc", "ai", "providers.d"))
	}
	for _, dir := range candidates {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
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
