package config

import (
	"net/url"
	"strings"
	"time"

	fwknowledge "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/knowledge"
)

type KnowledgeConfig struct {
	Mode             string `yaml:"mode" json:"mode"`
	DelegateEndpoint string `yaml:"delegate_endpoint" json:"delegate_endpoint"`
	DelegateTimeout  string `yaml:"delegate_timeout" json:"delegate_timeout"`
	RequireTenant    bool   `yaml:"require_tenant" json:"require_tenant"`
	BreakGlassLocal  bool   `yaml:"break_glass_local" json:"break_glass_local"`
	BreakGlassReason string `yaml:"break_glass_reason" json:"break_glass_reason"`
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
	return nil
}

func (c *Config) defaultKnowledgeMode() string {
	if c != nil && (c.IsProduction() || isPowerXProxyMode()) {
		return fwknowledge.ProviderModeDelegated
	}
	return fwknowledge.ProviderModeLocal
}
