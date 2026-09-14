package aisettings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

type restPayload struct {
	Method   string         `json:"method"`
	Endpoint string         `json:"endpoint"`
	Query    map[string]any `json:"query,omitempty"`
	Body     any            `json:"body,omitempty"`
}

// CatalogProviders reads the canonical catalog from the PowerX Agent settings
// backend. It is only called when the plugin settings page explicitly selects
// the PowerX catalog source.
func (c *Client) CatalogProviders(ctx context.Context, modality, requestID string) ([]CatalogProvider, error) {
	raw, err := c.invokeRawRESTForCapability(ctx, CapabilityAISettingsCatalogProviders, restPayload{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/admin/agents/providers",
		Query: map[string]any{
			"env":      "dev",
			"modality": strings.TrimSpace(modality),
		},
	}, requestID)
	if err != nil {
		return nil, err
	}
	var direct []CatalogProvider
	if json.Unmarshal(raw, &direct) == nil {
		return direct, nil
	}
	var envelope struct {
		// The Core REST binding returns the provider list under data.providers.
		// The capability invocation envelope preserves that REST payload as-is.
		Providers []CatalogProvider `json:"providers"`
		Data      json.RawMessage   `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	if envelope.Providers != nil {
		return envelope.Providers, nil
	}
	var nested struct {
		Providers []CatalogProvider `json:"providers"`
	}
	if err := json.Unmarshal(envelope.Data, &nested); err == nil && nested.Providers != nil {
		return nested.Providers, nil
	}
	var items []CatalogProvider
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, err
	}
	if items == nil {
		items = []CatalogProvider{}
	}
	return items, nil
}

// CatalogModels reads the canonical model IDs from the PowerX Agent settings
// backend. Model IDs are returned without UI label substitution.
func (c *Client) CatalogModels(ctx context.Context, modality, provider, app, requestID string) ([]string, error) {
	query := map[string]any{
		"env":      "dev",
		"modality": strings.TrimSpace(modality),
		"provider": strings.TrimSpace(provider),
	}
	if strings.TrimSpace(app) != "" {
		query["app"] = strings.TrimSpace(app)
	}
	raw, err := c.invokeRawRESTForCapability(ctx, CapabilityAISettingsCatalogModels, restPayload{
		Method:   http.MethodGet,
		Endpoint: "/api/v1/admin/agents/models",
		Query:    query,
	}, requestID)
	if err != nil {
		return nil, err
	}
	var wrapped struct {
		Models []string `json:"models"`
		Data   struct {
			Models []string `json:"models"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &wrapped); err != nil {
		return nil, err
	}
	if wrapped.Models != nil {
		return wrapped.Models, nil
	}
	if wrapped.Data.Models != nil {
		return wrapped.Data.Models, nil
	}
	return []string{}, nil
}

func NewClient(cfg Config) (*Client, error) {
	if cfg.Invoker == nil {
		return nil, errors.New("ai settings: gateway invoker is required")
	}
	return &Client{invoker: cfg.Invoker, tenantUUID: strings.TrimSpace(cfg.TenantUUID)}, nil
}

func (c *Client) Summary(ctx context.Context, requestID string) (Summary, error) {
	var out Summary
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/ai-settings/summary", nil, requestID, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = Summary{}
	}
	return out, nil
}

func (c *Client) ProviderProfiles(ctx context.Context, requestID string) ([]ProviderProfile, error) {
	var envelope struct {
		Items []ProviderProfile `json:"items"`
	}
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/ai-settings/provider-profiles", nil, requestID, &envelope); err != nil {
		return nil, err
	}
	if envelope.Items == nil {
		envelope.Items = []ProviderProfile{}
	}
	return envelope.Items, nil
}

func (c *Client) ModelProfiles(ctx context.Context, requestID string) ([]ModelProfile, error) {
	var envelope struct {
		Items []ModelProfile `json:"items"`
	}
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/ai-settings/model-profiles", nil, requestID, &envelope); err != nil {
		return nil, err
	}
	if envelope.Items == nil {
		envelope.Items = []ModelProfile{}
	}
	return envelope.Items, nil
}

func (c *Client) Routing(ctx context.Context, requestID string) (RoutingConfig, error) {
	var out RoutingConfig
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/ai-settings/routing", nil, requestID, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = RoutingConfig{}
	}
	return out, nil
}

func (c *Client) Health(ctx context.Context, requestID string) (HealthStatus, error) {
	var out HealthStatus
	if err := c.invoke(ctx, http.MethodGet, "/api/v1/admin/ai-settings/health", nil, requestID, &out); err != nil {
		return nil, err
	}
	if out == nil {
		out = HealthStatus{}
	}
	return out, nil
}

func (c *Client) invoke(ctx context.Context, method, endpoint string, body any, requestID string, out any) error {
	raw, err := c.invokeRawWithMethod(ctx, CapabilityAISettingsAdminRead, method, endpoint, body, requestID)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) invokeRaw(ctx context.Context, endpoint, requestID string) (json.RawMessage, error) {
	return c.invokeRawForCapability(ctx, "", endpoint, requestID)
}

func (c *Client) invokeRawForCapability(ctx context.Context, capabilityID, endpoint, requestID string) (json.RawMessage, error) {
	return c.invokeRawWithMethod(ctx, capabilityID, http.MethodGet, endpoint, nil, requestID)
}

func (c *Client) invokeRawWithMethod(ctx context.Context, capabilityID, method, endpoint string, body any, requestID string) (json.RawMessage, error) {
	return c.invokeRawRESTForCapability(ctx, capabilityID, restPayload{
		Method:   method,
		Endpoint: endpoint,
		Body:     body,
	}, requestID)
}

func (c *Client) invokeRawRESTForCapability(ctx context.Context, capabilityID string, payload restPayload, requestID string) (json.RawMessage, error) {
	if c == nil || c.invoker == nil {
		return nil, errors.New("ai settings: gateway invoker is required")
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      capabilityID,
		PreferredProtocol: "rest",
		RequestID:         strings.TrimSpace(requestID),
		TenantUUID:        c.tenantUUID,
		Payload:           payload,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.Data == nil {
		return nil, errors.New("ai settings: empty gateway response")
	}
	responsePayload, ok := resp.Data["payload"]
	if !ok {
		return nil, errors.New("ai settings: response payload is missing")
	}
	raw, err := json.Marshal(responsePayload)
	if err != nil {
		return nil, err
	}
	return raw, nil
}
