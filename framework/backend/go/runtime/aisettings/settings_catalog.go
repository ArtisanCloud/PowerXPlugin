package aisettings

import (
	"context"
	"errors"
	"strings"
)

const SettingsCatalogVersion = 1
const SettingsCatalogKind = "ai_settings"

type SettingsCatalogApp struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type SettingsCatalogProvider struct {
	ID   string               `json:"id"`
	Name string               `json:"name"`
	Apps []SettingsCatalogApp `json:"apps"`
}

// SettingsProviderCatalog is read from Core's AI Settings service, independently
// of its generative model catalog and of any plugin-local provider manifests.
type SettingsProviderCatalog struct {
	ContractVersion int                       `json:"contract_version"`
	CatalogKind     string                    `json:"catalog_kind"`
	Environment     string                    `json:"env"`
	Modality        string                    `json:"modality"`
	Providers       []SettingsCatalogProvider `json:"providers"`
}

type SettingsCatalogModel struct {
	ModelKey string `json:"model_key"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Label    string `json:"label"`
}

type SettingsModelCatalog struct {
	ContractVersion int                    `json:"contract_version"`
	CatalogKind     string                 `json:"catalog_kind"`
	Environment     string                 `json:"env"`
	Modality        string                 `json:"modality"`
	Provider        string                 `json:"provider"`
	App             string                 `json:"app"`
	Items           []SettingsCatalogModel `json:"items"`
}

func validSettingsEnvelope(version int, kind, environment, modality string) bool {
	return version == SettingsCatalogVersion && kind == SettingsCatalogKind && strings.TrimSpace(environment) != "" && modality == "llm"
}

func (out SettingsProviderCatalog) Validate() error {
	if !validSettingsEnvelope(out.ContractVersion, out.CatalogKind, out.Environment, out.Modality) || out.Providers == nil {
		return errors.New("AI_SETTINGS_CATALOG_RESPONSE_INVALID")
	}
	seen := map[string]bool{}
	for _, provider := range out.Providers {
		if strings.TrimSpace(provider.ID) == "" || strings.TrimSpace(provider.Name) == "" || seen[provider.ID] {
			return errors.New("AI_SETTINGS_CATALOG_RESPONSE_INVALID")
		}
		seen[provider.ID] = true
		for _, app := range provider.Apps {
			if strings.TrimSpace(app.ID) == "" || strings.TrimSpace(app.Name) == "" {
				return errors.New("AI_SETTINGS_CATALOG_RESPONSE_INVALID")
			}
		}
	}
	return nil
}

func (out SettingsModelCatalog) Validate(provider, app string) error {
	if !validSettingsEnvelope(out.ContractVersion, out.CatalogKind, out.Environment, out.Modality) || out.Items == nil || out.Provider != provider || out.App != app {
		return errors.New("AI_SETTINGS_CATALOG_RESPONSE_INVALID")
	}
	seen := map[string]bool{}
	for _, model := range out.Items {
		key := provider + "/" + model.Model
		if app != "" {
			key = provider + "/" + app + ":" + model.Model
		}
		if strings.TrimSpace(model.Model) == "" || strings.TrimSpace(model.Label) == "" || model.Provider != provider || model.ModelKey != key || seen[model.ModelKey] {
			return errors.New("AI_SETTINGS_CATALOG_RESPONSE_INVALID")
		}
		seen[model.ModelKey] = true
	}
	return nil
}

// SettingsCatalogProviders uses the existing catalog grant and a fixed typed
// operation. Core owns the tenant environment and the provider configuration.
func (c *Client) SettingsCatalogProviders(ctx context.Context, requestID string) (*SettingsProviderCatalog, error) {
	var out SettingsProviderCatalog
	if err := c.invokeCatalog(ctx, CapabilityAISettingsCatalogProviders, map[string]any{"operation": "list_settings_providers", "modality": "llm"}, requestID, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}

// SettingsCatalogModels preserves the exact model identifiers discovered by
// Core. Callers cannot replace Core's environment, endpoint or credentials.
func (c *Client) SettingsCatalogModels(ctx context.Context, provider, app, requestID string) (*SettingsModelCatalog, error) {
	provider, app = strings.TrimSpace(provider), strings.TrimSpace(app)
	if provider == "" {
		return nil, errors.New("AI_REQUEST_INVALID")
	}
	var out SettingsModelCatalog
	body := map[string]any{"operation": "list_settings_models", "modality": "llm", "provider": provider, "app": app}
	if err := c.invokeCatalog(ctx, CapabilityAISettingsCatalogModels, body, requestID, &out); err != nil {
		return nil, err
	}
	if err := out.Validate(provider, app); err != nil {
		return nil, err
	}
	return &out, nil
}
