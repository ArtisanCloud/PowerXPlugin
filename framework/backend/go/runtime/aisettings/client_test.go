package aisettings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	powerxai "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/ai"
	"github.com/stretchr/testify/require"
)

type catalogInvokerStub struct {
	request gateway.InvokeRequest
	payload any
	err     error
}

func (s *catalogInvokerStub) Invoke(_ context.Context, request gateway.InvokeRequest) (*gateway.Response, error) {
	s.request = request
	return &gateway.Response{Data: map[string]any{"protocol_used": "core_internal", "payload": map[string]any{"result": s.payload}}}, s.err
}

func TestCatalogUsesOnlyFixedMediaBinding(t *testing.T) {
	stub := &catalogInvokerStub{payload: map[string]any{"contract_version": 2, "items": []any{}}}
	client, err := NewClient(Config{Invoker: stub})
	require.NoError(t, err)
	out, err := client.CatalogModels(context.Background(), powerxai.ImageGeneration, "qwen-cn", "", "request-1")
	require.NoError(t, err)
	require.Empty(t, out.Items)
	require.Equal(t, CapabilityAISettingsCatalogModels, stub.request.CapabilityID)
	require.Equal(t, "core_internal", stub.request.PreferredProtocol)
	raw, err := json.Marshal(stub.request.Payload)
	require.NoError(t, err)
	require.JSONEq(t, `{"body":{"operation":"list_models","capability":"image_generation","provider":"qwen-cn","app":""}}`, string(raw))
	stub.payload = map[string]any{"contract_version": 2, "providers": []any{map[string]string{"provider": "qwen-cn", "label": "qwen"}}}
	providers, err := client.CatalogProviders(context.Background(), powerxai.ImageGeneration, "")
	require.NoError(t, err)
	require.Equal(t, "qwen-cn", providers.Providers[0].Provider)
	_, err = client.CatalogModels(context.Background(), "llm", "qwen-cn", "", "")
	require.Error(t, err)
	stub.payload = map[string]any{"models": []string{"legacy"}}
	_, err = client.CatalogModels(context.Background(), powerxai.ImageGeneration, "qwen-cn", "", "")
	require.EqualError(t, err, "AI_CATALOG_RESPONSE_INVALID")
}

func TestCatalogPreservesGatewayDiagnostics(t *testing.T) {
	stub := &catalogInvokerStub{err: &gateway.InvocationError{StatusCode: 422, TraceID: "trace", UpstreamRequestID: "request", Body: []byte(`{"error_code":"CAPABILITY_INVOKE_FAILED","details":{"reason_code":"AI_PARAMETER_INVALID","parameter":"temperature","reason":"unsupported_parameter"}}`)}}
	client, err := NewClient(Config{Invoker: stub})
	require.NoError(t, err)
	_, err = client.CatalogModels(context.Background(), powerxai.ImageGeneration, "qwen-cn", "", "")
	var typed *powerxai.HTTPError
	require.True(t, errors.As(err, &typed))
	require.Equal(t, 422, typed.StatusCode)
	require.Equal(t, "AI_PARAMETER_INVALID", typed.ReasonCode)
	require.Equal(t, "trace", typed.TraceID)
	require.Equal(t, "request", typed.RequestID)
	require.Equal(t, "temperature", typed.Details["parameter"])
}
