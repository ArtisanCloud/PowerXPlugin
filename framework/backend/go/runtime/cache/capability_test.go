package cache

import (
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
	"github.com/stretchr/testify/require"
)

type capabilityStub struct {
	mutate   func(map[string]any)
	failure  error
	requests []gateway.InvokeRequest
}

func (s *capabilityStub) Invoke(_ context.Context, in gateway.InvokeRequest) (*gateway.Response, error) {
	s.requests = append(s.requests, in)
	if s.failure != nil {
		return nil, s.failure
	}
	body := in.Payload.(map[string]any)["body"].(map[string]any)
	entry := map[string]any{"contract_version": ServiceContract, "operation": body["operation"], "namespace": body["namespace"], "key": body["key"], "found": true, "value_base64": base64.StdEncoding.EncodeToString([]byte("fixture-value")), "expires_at": time.Now().Add(time.Minute)}
	out := map[string]any{"status": "completed", "protocol_used": "core_internal", "fallback_used": false, "payload": entry}
	if s.mutate != nil {
		s.mutate(out)
	}
	return &gateway.Response{Data: out}, nil
}
func TestCacheCapabilityFixedTypedContractAndResponse(t *testing.T) {
	s := &capabilityStub{}
	client, e := NewCapabilityProvider(s)
	require.NoError(t, e)
	ctx := context.Background()
	entry, e := client.Get(ctx, scope, GetInput{Key: "fixture"})
	require.NoError(t, e)
	require.Equal(t, []byte("fixture-value"), entry.Value)
	require.NoError(t, client.Set(ctx, scope, SetInput{Key: "fixture", Value: []byte{}, TTL: time.Second}))
	require.NoError(t, client.Delete(ctx, scope, DeleteInput{Key: "fixture"}))
	for _, req := range s.requests {
		require.Equal(t, scope.TenantUUID, req.TenantUUID)
		require.Equal(t, "core_internal", req.PreferredProtocol)
		require.Nil(t, req.Headers)
		body := req.Payload.(map[string]any)
		require.Len(t, body, 1)
		op := body["body"].(map[string]any)
		require.Equal(t, scope.TenantUUID, op["tenant_key"])
		require.Equal(t, ServiceContract, op["contract_version"])
		require.NotContains(t, op, "endpoint")
		require.NotContains(t, op, "headers")
	}
	require.Equal(t, ServiceReadCapability, s.requests[0].CapabilityID)
	require.Equal(t, ServiceManageCapability, s.requests[1].CapabilityID)
}
func TestCacheCapabilityRejectsMalformedAndPropagatesFailure(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"fallback":        func(v map[string]any) { v["fallback_used"] = true },
		"protocol":        func(v map[string]any) { v["protocol_used"] = "rest" },
		"version":         func(v map[string]any) { v["payload"].(map[string]any)["contract_version"] = "old" },
		"other_key":       func(v map[string]any) { v["payload"].(map[string]any)["key"] = "other" },
		"missing_found":   func(v map[string]any) { delete(v["payload"].(map[string]any), "found") },
		"miss_with_value": func(v map[string]any) { v["payload"].(map[string]any)["found"] = false },
		"bad_base64":      func(v map[string]any) { v["payload"].(map[string]any)["value_base64"] = "bad" },
		"missing_expiry":  func(v map[string]any) { delete(v["payload"].(map[string]any), "expires_at") },
	} {
		t.Run(name, func(t *testing.T) {
			c, e := NewCapabilityProvider(&capabilityStub{mutate: mutate})
			require.NoError(t, e)
			_, e = c.Get(context.Background(), scope, GetInput{Key: "fixture"})
			require.ErrorIs(t, e, ErrInvalidResponse)
		})
	}
	upstream := errors.New("FIXTURE_CACHE_403")
	c, e := NewCapabilityProvider(&capabilityStub{failure: upstream})
	require.NoError(t, e)
	_, e = c.Get(context.Background(), scope, GetInput{Key: "fixture"})
	require.ErrorIs(t, e, upstream)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, e = c.Get(ctx, scope, GetInput{Key: "fixture"})
	require.ErrorIs(t, e, context.Canceled)
}
