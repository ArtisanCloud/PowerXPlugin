package config

import (
	"net/url"
	"strings"
	"time"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
)

type KnowledgeConfig struct {
	Mode             string                     `yaml:"mode" json:"mode"`
	DelegateEndpoint string                     `yaml:"delegate_endpoint" json:"delegate_endpoint"`
	DelegateTimeout  string                     `yaml:"delegate_timeout" json:"delegate_timeout"`
	RequireTenant    bool                       `yaml:"require_tenant" json:"require_tenant"`
	BreakGlassLocal  bool                       `yaml:"break_glass_local" json:"break_glass_local"`
	BreakGlassReason string                     `yaml:"break_glass_reason" json:"break_glass_reason"`
	VectorStore      KnowledgeVectorStoreConfig `yaml:"vector_store" json:"vector_store"`
}

type KnowledgeVectorStoreConfig struct {
	Driver   string                       `yaml:"driver" json:"driver"`
	PGVector KnowledgePGVectorStoreConfig `yaml:"pgvector" json:"pgvector"`
}

type KnowledgePGVectorStoreConfig struct {
	Schema           string `yaml:"schema" json:"schema"`
	Table            string `yaml:"table" json:"table"`
	Dimensions       int    `yaml:"dimensions" json:"dimensions"`
	EnableMigrations bool   `yaml:"enable_migrations" json:"enable_migrations"`
	Lists            int    `yaml:"ivfflat_lists" json:"ivfflat_lists"`
}

func (c *Config) NormalizeKnowledgeConfig() error {
	if c.Knowledge == nil {
		c.Knowledge = &KnowledgeConfig{Mode: c.defaultKnowledgeMode(), DelegateTimeout: "3s", RequireTenant: true}
	}
	mode := strings.ToLower(strings.TrimSpace(c.Knowledge.Mode))
	if mode == "" {
		mode = c.defaultKnowledgeMode()
	}
	switch mode {
	case fwknowledge.ProviderModeLocal, fwknowledge.ProviderModeMock, fwknowledge.ProviderModeDelegated, fwknowledge.ProviderModeThirdParty:
		c.Knowledge.Mode = mode
	default:
		return NewConfigError("knowledge.mode must be one of: local, mock, delegated, third_party")
	}
	c.Knowledge.DelegateEndpoint = resolveConfigValue(c.Knowledge.DelegateEndpoint)
	c.Knowledge.DelegateTimeout = resolveConfigValue(c.Knowledge.DelegateTimeout)
	if strings.TrimSpace(c.Knowledge.DelegateTimeout) == "" {
		c.Knowledge.DelegateTimeout = "3s"
	}
	if _, err := time.ParseDuration(c.Knowledge.DelegateTimeout); err != nil {
		return NewConfigError("knowledge.delegate_timeout must be a valid duration (e.g. 3s, 500ms)")
	}
	if c.Knowledge.Mode == fwknowledge.ProviderModeDelegated || c.Knowledge.Mode == fwknowledge.ProviderModeThirdParty {
		endpoint := strings.TrimSpace(c.Knowledge.DelegateEndpoint)
		if endpoint != "" {
			parsed, err := url.Parse(endpoint)
			if err != nil || parsed.Scheme == "" || parsed.Host == "" {
				return NewConfigError("knowledge.delegate_endpoint must be an absolute URL")
			}
		}
	}
	if err := fwknowledge.ValidateSourcePolicy(fwknowledge.SourcePolicy{
		Mode:       c.Knowledge.Mode,
		Production: c.IsProduction(),
		BreakGlass: c.Knowledge.BreakGlassLocal,
	}); err != nil {
		return NewConfigError(err.Error())
	}
	if c.IsProduction() && c.Knowledge.BreakGlassLocal && strings.TrimSpace(c.Knowledge.BreakGlassReason) == "" {
		return NewConfigError("knowledge.break_glass_reason is required when break_glass_local is enabled")
	}
	c.Knowledge.VectorStore.Driver = strings.ToLower(strings.TrimSpace(c.Knowledge.VectorStore.Driver))
	if c.Knowledge.VectorStore.Driver == "" {
		c.Knowledge.VectorStore.Driver = "pgvector"
	}
	if c.Knowledge.VectorStore.Driver != "pgvector" {
		return NewConfigError("knowledge.vector_store.driver must be pgvector")
	}
	c.Knowledge.VectorStore.PGVector.Schema = strings.TrimSpace(resolveConfigValue(c.Knowledge.VectorStore.PGVector.Schema))
	if c.Knowledge.VectorStore.PGVector.Schema == "" && c.Database != nil {
		c.Knowledge.VectorStore.PGVector.Schema = strings.TrimSpace(c.Database.Schema)
	}
	if c.Knowledge.VectorStore.PGVector.Schema == "" {
		c.Knowledge.VectorStore.PGVector.Schema = "public"
	}
	c.Knowledge.VectorStore.PGVector.Table = strings.TrimSpace(resolveConfigValue(c.Knowledge.VectorStore.PGVector.Table))
	if c.Knowledge.VectorStore.PGVector.Table == "" {
		c.Knowledge.VectorStore.PGVector.Table = "local_knowledge_vectors_v1_1536"
	}
	if c.Knowledge.VectorStore.PGVector.Dimensions <= 0 {
		c.Knowledge.VectorStore.PGVector.Dimensions = 1536
	}
	if c.Knowledge.VectorStore.PGVector.Lists <= 0 {
		c.Knowledge.VectorStore.PGVector.Lists = 100
	}
	return nil
}

// LocalPGVectorEnabled is deliberately strict: delegated knowledge is Core
// owned and must never provision plugin-local vector infrastructure.
func (c *Config) LocalPGVectorEnabled() bool {
	return c != nil && c.Database != nil && strings.EqualFold(c.Database.Driver, "postgres") && c.Knowledge != nil && c.Knowledge.Mode == fwknowledge.ProviderModeLocal && c.Knowledge.VectorStore.Driver == "pgvector"
}

func (c *Config) defaultKnowledgeMode() string {
	if c != nil && (c.IsProduction() || isPowerXProxyMode()) {
		return fwknowledge.ProviderModeDelegated
	}
	return fwknowledge.ProviderModeLocal
}
