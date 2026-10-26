package aisettings

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
)

type restPayload struct {
	Method   string         `json:"method"`
	Endpoint string         `json:"endpoint"`
	Query    map[string]any `json:"query,omitempty"`
	Body     any            `json:"body,omitempty"`
}

// CatalogProviders and CatalogModels use the fixed media v2 service binding.
func (c *Client) CatalogProviders(ctx context.Context, capability, requestID string) (*powerxai.MediaProviderCatalog, error) {
	if err := powerxai.ValidateMediaCatalogFilter(powerxai.MediaCatalogFilter{Capability: capability}); err != nil {
		return nil, err
	}
	var out powerxai.MediaProviderCatalog
	err := c.invokeCatalog(ctx, CapabilityAISettingsCatalogProviders, map[string]any{"operation": "list_providers", "capability": capability}, requestID, &out)
	if err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) CatalogModels(ctx context.Context, capability, provider, app, requestID string) (*powerxai.MediaModelCatalog, error) {
	if err := powerxai.ValidateMediaCatalogFilter(powerxai.MediaCatalogFilter{Capability: capability}); err != nil {
		return nil, err
	}
	var out powerxai.MediaModelCatalog
	err := c.invokeCatalog(ctx, CapabilityAISettingsCatalogModels, map[string]any{"operation": "list_models", "capability": capability, "provider": provider, "app": app}, requestID, &out)
	if err != nil {
		return nil, err
	}
	if err := out.Validate(); err != nil {
		return nil, err
	}
	return &out, nil
}
func (c *Client) invokeCatalog(ctx context.Context, capability string, body any, requestID string, out any) error {
	if c == nil || c.invoker == nil {
		return errors.New("AI_CLIENT_UNAVAILABLE")
	}
	response, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: capability, PreferredProtocol: "core_internal", TenantUUID: c.tenantUUID, RequestID: requestID, Payload: map[string]any{"body": body}})
	if err != nil {
		var invocation *gateway.InvocationError
		if errors.As(err, &invocation) {
			headers := http.Header{}
			headers.Set("X-Trace-ID", invocation.TraceID)
			headers.Set("X-Request-ID", invocation.UpstreamRequestID)
			return powerxai.DecodeHTTPError(invocation.StatusCode, headers, invocation.Body)
		}
		return err
	}
	if response == nil || response.Data == nil || response.Data["protocol_used"] != "core_internal" || response.Data["fallback_used"] == true {
		return errors.New("AI_CATALOG_RESPONSE_INVALID")
	}
	raw, err := json.Marshal(response.Data["payload"])
	if err != nil {
		return err
	}
	var payload struct {
		Result json.RawMessage `json:"result"`
	}
	if json.Unmarshal(raw, &payload) != nil || len(payload.Result) == 0 || string(payload.Result) == "null" {
		return errors.New("AI_CATALOG_RESPONSE_INVALID")
	}
	return json.Unmarshal(payload.Result, out)
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
