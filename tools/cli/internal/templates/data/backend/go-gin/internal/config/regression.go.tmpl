package config

import (
	Regression "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/regression"
	"strings"
)

type RegressionProfileConfig struct {
	TenantUUID           string `yaml:"tenant_uuid" json:"tenant_uuid"`
	MetadataResourceType string `yaml:"metadata_resource_type" json:"metadata_resource_type"`
	MetadataResourceUUID string `yaml:"metadata_resource_uuid" json:"metadata_resource_uuid"`
	UUID                 string `yaml:"uuid" json:"uuid"`
	Key                  string `yaml:"key" json:"key"`
	Kind                 string `yaml:"kind" json:"kind"`
	TitleKey             string `yaml:"title_key" json:"title_key"`
	FixtureNamespace     string `yaml:"fixture_namespace" json:"fixture_namespace"`
}
type RegressionConfig struct {
	Enabled     bool                      `yaml:"enabled" json:"enabled"`
	Environment string                    `yaml:"environment" json:"environment"`
	Profiles    []RegressionProfileConfig `yaml:"profiles" json:"profiles"`
}

func (c RegressionConfig) Validate() error {
	if !c.Enabled {
		return nil
	}
	if c.Environment != "development" && c.Environment != "test" {
		return Regression.NewError(Regression.InvalidRequest)
	}
	if len(c.Profiles) == 0 {
		return Regression.NewError(Regression.InvalidRequest)
	}
	seen := map[string]bool{}
	for _, p := range c.Profiles {
		if !Regression.ValidUUID(p.UUID) || seen[p.UUID] || strings.TrimSpace(p.Key) == "" || strings.TrimSpace(p.TitleKey) == "" || strings.TrimSpace(p.FixtureNamespace) == "" || (p.Kind != "controlled" && p.Kind != "live") {
			return Regression.NewError(Regression.InvalidRequest)
		}
		seen[p.UUID] = true
		if p.Kind == "live" && (!Regression.ValidUUID(p.TenantUUID) || !Regression.ValidUUID(p.MetadataResourceUUID) || strings.TrimSpace(p.MetadataResourceType) == "") {
			return Regression.NewError(Regression.InvalidRequest)
		}
	}
	return nil
}
