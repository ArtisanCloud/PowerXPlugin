package eventfabric

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
	"github.com/google/uuid"
)

type Client struct{ gateway *capability.Client }

func NewClient(cfg Config, h *http.Client) (*Client, error) {
	if _, err := hostcontract.InvocationEndpoint(cfg.BaseURL, cfg.APIPrefix, "/tenant/invocations"); err != nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	c, err := capability.NewClient(cfg, h)
	if err != nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	return &Client{gateway: c}, nil
}
func NewClientWithTokenProvider(cfg Config, tokens TokenProvider, h *http.Client) (*Client, error) {
	// An explicitly selected API Key never invokes the STS provider.
	if strings.EqualFold(cfg.AuthScheme, "apikey") || strings.EqualFold(cfg.AuthScheme, "api_key") || strings.EqualFold(cfg.AuthScheme, "api-key") {
		return NewClient(cfg, h)
	}
	if _, err := hostcontract.InvocationEndpoint(cfg.BaseURL, cfg.APIPrefix, "/tenant/invocations"); err != nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	if tokens == nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	value := reflect.ValueOf(tokens)
	if (value.Kind() == reflect.Func || value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface) && value.IsNil() {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	c, err := capability.NewClientWithTokenProvider(cfg, tokens, h)
	if err != nil {
		return nil, failure(400, "EVENT_FABRIC_RUNTIME_CONFIG_INVALID")
	}
	return &Client{gateway: c}, nil
}
func valid(s string, max int) bool {
	return s != "" && len(s) <= max && s == strings.TrimSpace(s) && !strings.ContainsAny(s, "\r\n\x00")
}
func canonicalUUID(s string) bool {
	id, e := uuid.Parse(s)
	return e == nil && id != uuid.Nil && id.String() == s
}
func identity(tenant, plugin string) bool { return canonicalUUID(tenant) && valid(plugin, 128) }
func physicalTopic(tenant, topic string) bool {
	return valid(topic, 255) && strings.HasPrefix(topic, tenant+".")
}

func (c *Client) invoke(ctx context.Context, capID string, body any, out any) error {
	if c == nil || c.gateway == nil {
		return failure(503, "EVENT_FABRIC_UNAVAILABLE")
	}
	r, err := c.gateway.Invoke(ctx, capability.InvokeInput{CapabilityID: capID, PreferredProtocol: Protocol, Payload: map[string]any{"body": body}})
	if err != nil {
		return err
	}
	if r == nil || !r.HasRoutingEvidence() || r.ProtocolUsed != Protocol || r.FallbackUsed || r.Status != "completed" || r.Result == nil {
		return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
	}
	raw, err := json.Marshal(r.Result)
	if err != nil || json.Unmarshal(raw, out) != nil {
		return failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
	}
	return nil
}
func (c *Client) Publish(ctx context.Context, in PublishRequest) (*PublishReceipt, error) {
	e := in.Event
	if !identity(in.TenantUUID, in.PluginID) || !valid(e.EventID, 128) || !valid(e.IdempotencyKey, 128) || !physicalTopic(in.TenantUUID, e.Topic) || !valid(e.Version, 32) || !json.Valid(e.Payload) || len(e.Payload) > 1<<20 || len(e.Attributes) > 64 || (e.TraceID != "" && !valid(e.TraceID, 128)) {
		return nil, failure(400, "EVENT_FABRIC_INVALID_ARGUMENT")
	}
	for k, v := range e.Attributes {
		if !valid(k, 64) || len(v) > 2048 || k == "principal_id" || k == "tenant_uuid" || k == "plugin_id" || k == "published_by" || strings.HasPrefix(k, "core_") {
			return nil, failure(400, "EVENT_FABRIC_RESERVED_ATTRIBUTE")
		}
	}
	body := map[string]any{"contract_version": Contract, "operation": "publish", "tenant_uuid": in.TenantUUID, "plugin_id": in.PluginID, "event": e}
	var out PublishReceipt
	if err := c.invoke(ctx, PublishCapability, body, &out); err != nil {
		return nil, err
	}
	if out.ContractVersion != Contract || !out.Accepted || out.EventID != e.EventID || !valid(out.TraceID, 256) {
		return nil, failure(502, "EVENT_FABRIC_RESPONSE_INVALID")
	}
	return &out, nil
}
func (c *Client) settle(ctx context.Context, receipt DeliveryReceipt, operation, reason string) (*NackResult, error) {
	wrap := func(err error) (*NackResult, error) {
		return nil, &SettlementError{Receipt: receipt, Uncertain: uncertain(err), Err: err}
	}
	if !identity(receipt.TenantUUID, receipt.PluginID) || receipt.SubscriberID != receipt.PluginID || !canonicalUUID(receipt.DeliveryID) || len(reason) > 256 {
		return wrap(failure(400, "EVENT_FABRIC_INVALID_ARGUMENT"))
	}
	if receipt.AckDeadline.IsZero() || !time.Now().Before(receipt.AckDeadline) {
		return wrap(failure(409, "EVENT_FABRIC_RECEIPT_STALE"))
	}
	body := map[string]any{"contract_version": Contract, "operation": operation, "tenant_uuid": receipt.TenantUUID, "plugin_id": receipt.PluginID, "subscriber_id": receipt.SubscriberID, "delivery_id": receipt.DeliveryID}
	if reason != "" {
		body["reason"] = reason
	}
	// Pointers distinguish missing result fields from meaningful false/zero.
	var raw struct {
		ContractVersion string `json:"contract_version"`
		Confirmed       *bool  `json:"confirmed"`
		DeliveryID      string `json:"delivery_id"`
		Outcome         string `json:"outcome"`
		Remaining       *int   `json:"remaining_attempts"`
		Delay           *int   `json:"next_delay_seconds"`
	}
	if err := c.invoke(ctx, ConsumeCapability, body, &raw); err != nil {
		return wrap(err)
	}
	if raw.ContractVersion != Contract || raw.Confirmed == nil || !*raw.Confirmed || raw.DeliveryID != receipt.DeliveryID || raw.Remaining == nil || *raw.Remaining < 0 || raw.Delay == nil || *raw.Delay < 0 || (operation == "ack" && raw.Outcome != "acked") || (operation == "nack" && raw.Outcome != "retry_scheduled" && raw.Outcome != "dead_lettered") {
		return wrap(failure(502, "EVENT_FABRIC_RESPONSE_INVALID"))
	}
	return &NackResult{raw.ContractVersion, *raw.Confirmed, raw.DeliveryID, raw.Outcome, *raw.Remaining, *raw.Delay}, nil
}
func (c *Client) Ack(ctx context.Context, r DeliveryReceipt) error {
	_, err := c.settle(ctx, r, "ack", "")
	return err
}
func (c *Client) Nack(ctx context.Context, r DeliveryReceipt, reason string) (*NackResult, error) {
	return c.settle(ctx, r, "nack", reason)
}
