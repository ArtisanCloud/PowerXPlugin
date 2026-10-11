package cache

import (
	"context"
	"encoding/base64"
	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/hostapi"
	"net/http"
	"net/url"
	"time"
)

const ReadCapability = "com.corex.runtime.cache.read"
const ManageCapability = "com.corex.runtime.cache.manage"

type HostProvider struct{ client *hostapi.Client }

func NewHostProvider(cfg hostapi.Config, tokens hostapi.TokenProvider, h *http.Client) (Service, error) {
	c, err := hostapi.New(cfg, tokens, h, "CACHE")
	if err != nil {
		return nil, err
	}
	return checked{&HostProvider{c}}, nil
}
func path(scope Scope, key string) string {
	return "/api/v1/tenant/runtime/cache/entries?" + url.Values{"namespace": {scope.Namespace}, "key": {key}}.Encode()
}
func (p *HostProvider) Get(ctx context.Context, scope Scope, in GetInput) (*Entry, error) {
	var out struct {
		Found     *bool      `json:"found"`
		Value     *string    `json:"value_base64"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := p.client.Do(ctx, scope.TenantUUID, "GET", path(scope, in.Key), nil, &out); err != nil {
		return nil, err
	}
	if out.Found == nil || out.Value == nil {
		return nil, ErrInvalidResponse
	}
	value, err := base64.StdEncoding.Strict().DecodeString(*out.Value)
	if err != nil || base64.StdEncoding.EncodeToString(value) != *out.Value || len(value) > 1<<20 {
		return nil, ErrInvalidResponse
	}
	e := &Entry{Found: *out.Found, Value: value}
	if out.ExpiresAt != nil {
		e.ExpiresAt = *out.ExpiresAt
	}
	return e, nil
}
func (p *HostProvider) Set(ctx context.Context, scope Scope, in SetInput) error {
	body := struct {
		Namespace string `json:"namespace"`
		Key       string `json:"key"`
		Value     string `json:"value_base64"`
		TTL       int64  `json:"ttl_ms"`
	}{scope.Namespace, in.Key, base64.StdEncoding.EncodeToString(in.Value), in.TTL.Milliseconds()}
	return p.client.Do(ctx, scope.TenantUUID, "PUT", "/api/v1/tenant/runtime/cache/entries", body, nil)
}
func (p *HostProvider) Delete(ctx context.Context, scope Scope, in DeleteInput) error {
	return p.client.Do(ctx, scope.TenantUUID, "DELETE", path(scope, in.Key), nil, nil)
}
