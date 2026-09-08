package media

import (
	"context"
	"testing"

	powerxcapability "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/capability"
)

type registryStub struct {
	input powerxcapability.InvokeInput
}

func (s *registryStub) List(context.Context, powerxcapability.ListInput) ([]powerxcapability.Capability, error) {
	return nil, nil
}
func (s *registryStub) GrantStatus(context.Context, powerxcapability.GrantStatusInput) ([]powerxcapability.GrantStatusItem, error) {
	return nil, nil
}
func (s *registryStub) Resolve(context.Context, powerxcapability.ResolveInput) (*powerxcapability.ResolveResult, error) {
	return nil, nil
}
func (s *registryStub) Invoke(_ context.Context, input powerxcapability.InvokeInput) (*powerxcapability.InvokeResult, error) {
	s.input = input
	return &powerxcapability.InvokeResult{TraceID: "trace-1", Result: map[string]any{"items": []any{map[string]any{"uuid": "asset-1", "name": "image"}}, "total": 1, "page": 2, "page_size": 10}}, nil
}
func (s *registryStub) GetInvocation(context.Context, string) (*powerxcapability.Invocation, error) {
	return nil, nil
}

func TestListAssetsUsesTypedCapabilityWithoutTenantInput(t *testing.T) {
	registry := &registryStub{}
	result, err := NewClient(registry).ListAssets(context.Background(), ListAssetsInput{Page: 2, PageSize: 10, Keyword: "image"})
	if err != nil {
		t.Fatal(err)
	}
	if registry.input.CapabilityID != CapabilityAssetsRead || registry.input.Payload["tenant_uuid"] != nil {
		t.Fatalf("input=%#v", registry.input)
	}
	if result.TraceID != "trace-1" || len(result.Items) != 1 || result.Items[0].UUID != "asset-1" {
		t.Fatalf("result=%#v", result)
	}
}
