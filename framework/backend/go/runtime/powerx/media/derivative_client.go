package media

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (c *HostClient) GetImageDerivatives(ctx context.Context, assetUUID string) (*ImageDerivatives, error) {
	return c.invokeImageDerivatives(ctx, ImageDerivativesReadCapability, "status", assetUUID, false)
}
func (c *HostClient) EnsureImageDerivatives(ctx context.Context, assetUUID string, input EnsureImageDerivativesInput) (*ImageDerivatives, error) {
	return c.invokeImageDerivatives(ctx, ImageDerivativesEnsureCapability, "ensure", assetUUID, input.RetryFailed)
}
func (c *HostClient) invokeImageDerivatives(ctx context.Context, capability, operation, assetUUID string, retry bool) (*ImageDerivatives, error) {
	if !validMediaUUID(assetUUID) {
		return nil, derivativeError(400, "MEDIA_ASSET_UUID_INVALID")
	}
	body := struct {
		Operation   string `json:"operation"`
		AssetUUID   string `json:"asset_uuid"`
		RetryFailed bool   `json:"retry_failed,omitempty"`
	}{operation, assetUUID, retry}
	request := struct {
		CapabilityID      string `json:"capability_id"`
		PreferredProtocol string `json:"preferred_protocol"`
		Payload           struct {
			Body any `json:"body"`
		} `json:"payload"`
	}{CapabilityID: capability, PreferredProtocol: "core_internal"}
	request.Payload.Body = body
	var data struct {
		ProtocolUsed string `json:"protocol_used"`
		FallbackUsed *bool  `json:"fallback_used"`
		Payload      struct {
			Result json.RawMessage `json:"result"`
		} `json:"payload"`
	}
	if err := c.do(ctx, http.MethodPost, "/tenant/invocations", request, &data); err != nil {
		return nil, err
	}
	if data.ProtocolUsed != "core_internal" || data.FallbackUsed == nil || *data.FallbackUsed {
		return nil, derivativeError(502, "MEDIA_DERIVATIVES_BINDING_INVALID")
	}
	return decodeDerivativeSnapshot(data.Payload.Result, assetUUID)
}

func decodeDerivativeSnapshot(raw []byte, assetUUID string) (*ImageDerivatives, error) {
	var fields map[string]json.RawMessage
	invalid := func() error { return derivativeError(502, "MEDIA_DERIVATIVES_RESPONSE_INVALID") }
	if json.Unmarshal(raw, &fields) != nil {
		return nil, invalid()
	}
	for _, key := range []string{"asset_uuid", "upload_state", "status", "recipe", "attempts", "retryable", "items"} {
		if len(fields[key]) == 0 || string(fields[key]) == "null" {
			return nil, invalid()
		}
	}
	var out ImageDerivatives
	if json.Unmarshal(raw, &out) != nil {
		return nil, invalid()
	}
	var items []map[string]json.RawMessage
	_ = json.Unmarshal(fields["items"], &items)
	for i, item := range items {
		if _, present := item["variant_uuid"]; present && out.Items[i].Status != "ready" {
			return nil, invalid()
		}
	}
	if err := out.Validate(assetUUID); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *HostClient) authorize(ctx context.Context, request *http.Request) error {
	if c == nil || c.http == nil {
		return derivativeError(503, "MEDIA_CLIENT_UNAVAILABLE")
	}
	if c.apiKey != "" {
		request.Header.Set("Authorization", "ApiKey "+c.apiKey)
		return nil
	}
	if c.tokens == nil {
		return derivativeError(503, "MEDIA_CLIENT_UNAVAILABLE")
	}
	token, err := c.tokens.Token(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if err != nil || strings.TrimSpace(token) == "" {
		return derivativeError(503, "MEDIA_SERVICE_CREDENTIAL_UNAVAILABLE")
	}
	request.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	return nil
}
