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
	if c == nil || c.invoker == nil {
		return errors.New("ai settings: gateway invoker is required")
	}
	resp, err := c.invoker.Invoke(ctx, gateway.InvokeRequest{
		CapabilityID:      CapabilityAISettingsAdminRead,
		PreferredProtocol: "rest",
		RequestID:         strings.TrimSpace(requestID),
		TenantUUID:        c.tenantUUID,
		Payload: restPayload{
			Method:   method,
			Endpoint: endpoint,
			Body:     body,
		},
	})
	if err != nil {
		return err
	}
	if resp == nil || resp.Data == nil {
		return errors.New("ai settings: empty gateway response")
	}
	payload, ok := resp.Data["payload"]
	if !ok {
		return errors.New("ai settings: response payload is missing")
	}
	if out == nil {
		return nil
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}
