package cache

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/gateway"
)

const ServiceContract = "cache.service.v1"
const ServiceReadCapability = "com.corex.runtime.cache.service_read"
const ServiceManageCapability = "com.corex.runtime.cache.service_manage"

type CapabilityInvoker interface {
	Invoke(context.Context, gateway.InvokeRequest) (*gateway.Response, error)
}

// CapabilityProvider fixes the typed service contract for both API Key and STS.
// Its caller cannot choose a host endpoint, method, headers or caller subject.
type CapabilityProvider struct{ invoker CapabilityInvoker }

func NewCapabilityProvider(invoker CapabilityInvoker) (Service, error) {
	if invoker == nil || (reflect.ValueOf(invoker).Kind() == reflect.Pointer && reflect.ValueOf(invoker).IsNil()) {
		return nil, ErrInvalidArgument
	}
	return checked{&CapabilityProvider{invoker}}, nil
}

func (p *CapabilityProvider) invoke(ctx context.Context, scope Scope, capability string, body map[string]any) (json.RawMessage, error) {
	body["contract_version"] = ServiceContract
	body["namespace"] = scope.Namespace
	body["tenant_key"] = scope.TenantUUID
	response, err := p.invoker.Invoke(ctx, gateway.InvokeRequest{CapabilityID: capability, TenantUUID: scope.TenantUUID, PreferredProtocol: "core_internal", Payload: map[string]any{"body": body}})
	if err != nil {
		return nil, err
	}
	if response == nil || response.Data["status"] != "completed" || response.Data["protocol_used"] != "core_internal" || response.Data["fallback_used"] != false {
		return nil, ErrInvalidResponse
	}
	raw, err := json.Marshal(response.Data["payload"])
	if err != nil {
		return nil, ErrInvalidResponse
	}
	var contract struct {
		Version   string `json:"contract_version"`
		Operation string `json:"operation"`
		Namespace string `json:"namespace"`
		Key       string `json:"key"`
	}
	if json.Unmarshal(raw, &contract) != nil || contract.Version != ServiceContract || contract.Operation != body["operation"] || contract.Namespace != scope.Namespace || contract.Key != body["key"] {
		return nil, ErrInvalidResponse
	}
	return raw, nil
}

func (p *CapabilityProvider) Get(ctx context.Context, scope Scope, in GetInput) (*Entry, error) {
	raw, err := p.invoke(ctx, scope, ServiceReadCapability, map[string]any{"operation": "get", "key": in.Key})
	if err != nil {
		return nil, err
	}
	var out struct {
		Found     *bool      `json:"found"`
		Value     *string    `json:"value_base64"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if json.Unmarshal(raw, &out) != nil || out.Found == nil || out.Value == nil {
		return nil, ErrInvalidResponse
	}
	value, err := base64.StdEncoding.Strict().DecodeString(*out.Value)
	if err != nil || base64.StdEncoding.EncodeToString(value) != *out.Value || len(value) > 1<<20 {
		return nil, ErrInvalidResponse
	}
	entry := &Entry{Found: *out.Found, Value: value}
	if out.ExpiresAt != nil {
		entry.ExpiresAt = *out.ExpiresAt
	}
	return entry, nil
}

func (p *CapabilityProvider) Set(ctx context.Context, scope Scope, in SetInput) error {
	_, err := p.invoke(ctx, scope, ServiceManageCapability, map[string]any{"operation": "set", "key": in.Key, "value_base64": base64.StdEncoding.EncodeToString(in.Value), "ttl_ms": in.TTL.Milliseconds()})
	return err
}

func (p *CapabilityProvider) Delete(ctx context.Context, scope Scope, in DeleteInput) error {
	_, err := p.invoke(ctx, scope, ServiceManageCapability, map[string]any{"operation": "delete", "key": in.Key})
	return err
}
