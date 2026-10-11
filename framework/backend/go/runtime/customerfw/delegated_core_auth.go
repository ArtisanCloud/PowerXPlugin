package customerfw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

const ShopifyStorefrontChannel = "shopify_storefront"

// DelegatedCoreAuthClient calls the formal Core Customer Auth Host Contract.
// Tenant comes only from the plugin STS credential; TenantUUID input fields
// are rejected rather than forwarded.
type DelegatedCoreAuthClient struct {
	baseURL string
	tokens  ServiceTokenProvider
	timeout time.Duration
	client  *http.Client
}
type DelegatedCoreAuthClientConfig struct {
	BaseURL       string
	ServiceTokens ServiceTokenProvider
	Timeout       time.Duration
	Client        *http.Client
}

func NewDelegatedCoreAuthClient(cfg DelegatedCoreAuthClientConfig) (*DelegatedCoreAuthClient, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("customer delegated auth: base URL is required")
	}
	if cfg.ServiceTokens == nil {
		return nil, errors.New("customer delegated auth: service token provider is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 500*time.Millisecond}
	}
	return &DelegatedCoreAuthClient{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), tokens: cfg.ServiceTokens, timeout: timeout, client: client}, nil
}

func (c *DelegatedCoreAuthClient) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	return c.issue(ctx, "/api/v1/tenant/customer/auth/register", in.TenantUUID, in.Channel, in.Credential)
}
func (c *DelegatedCoreAuthClient) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	return c.issue(ctx, "/api/v1/tenant/customer/auth/login", in.TenantUUID, in.Channel, in.Credential)
}
func (c *DelegatedCoreAuthClient) Validate(ctx context.Context, _ string) (*CustomerContext, error) {
	if _, ok := CustomerCredentialFromContext(ctx); !ok {
		return nil, NewError(CodeCustomerTokenMissing, "customer credential missing from request context")
	}
	var out struct {
		Item membershipWire `json:"item"`
	}
	if err := c.call(ctx, http.MethodPost, "/api/v1/tenant/customer/auth/validate", nil, &out, true); err != nil {
		return nil, err
	}
	if out.Item.TenantUUID == "" || out.Item.CustomerUUID == "" || out.Item.MembershipUUID == "" {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer.auth.response_incomplete")
	}
	if out.Item.Status != "active" {
		return nil, NewError(CodeCustomerMembershipDisabled, "customer.membership.inactive")
	}
	return contextFromMembership(out.Item), nil
}

func (c *DelegatedCoreAuthClient) issue(ctx context.Context, path, tenant, channel string, credential CustomerCredential) (*AuthResult, error) {
	if strings.TrimSpace(tenant) != "" {
		return nil, NewError(CodeCustomerTenantMismatch, "delegated customer auth does not accept tenant_uuid")
	}
	if strings.TrimSpace(channel) != ShopifyStorefrontChannel || strings.TrimSpace(credential.Type) != "shopify_customer_access_token" || strings.TrimSpace(credential.Value) == "" {
		return nil, NewError(CodeCustomerTokenInvalid, "unsupported delegated customer credential")
	}
	var out struct {
		AccessToken  string              `json:"access_token"`
		RefreshToken string              `json:"refresh_token"`
		ExpiresIn    int64               `json:"expires_in"`
		Context      customerContextWire `json:"context"`
	}
	if err := c.call(ctx, http.MethodPost, path, map[string]any{"channel": channel, "credential": credential}, &out, false); err != nil {
		return nil, err
	}
	context := CustomerContext{TenantUUID: out.Context.TenantUUID, CustomerUUID: out.Context.CustomerUUID, MembershipUUID: out.Context.MembershipUUID, Roles: out.Context.Roles, Scopes: out.Context.Scopes, Source: CustomerAuthSourceDelegate, Authenticated: out.Context.Authenticated, TokenExpiresAt: out.Context.ExpiresAt}
	if strings.TrimSpace(out.AccessToken) == "" || context.TenantUUID == "" || context.CustomerUUID == "" || context.MembershipUUID == "" || !context.Authenticated || out.ExpiresIn <= 0 {
		return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer auth response is incomplete")
	}
	if out.Context.Status != "active" {
		return nil, NewError(CodeCustomerMembershipDisabled, "customer.membership.inactive")
	}
	return &AuthResult{AccessToken: out.AccessToken, RefreshToken: out.RefreshToken, ExpiresIn: out.ExpiresIn, Context: NormalizeContext(&context)}, nil
}

type customerContextWire struct {
	Status         string     `json:"status"`
	TenantUUID     string     `json:"tenant_uuid"`
	CustomerUUID   string     `json:"customer_uuid"`
	MembershipUUID string     `json:"membership_uuid"`
	Roles          []string   `json:"roles"`
	Scopes         []string   `json:"scopes"`
	Authenticated  bool       `json:"authenticated"`
	ExpiresAt      *time.Time `json:"expires_at"`
}
type membershipWire struct {
	Status         string     `json:"status"`
	TenantUUID     string     `json:"tenant_uuid"`
	CustomerUUID   string     `json:"customer_uuid"`
	MembershipUUID string     `json:"membership_uuid"`
	Roles          []string   `json:"roles"`
	Scopes         []string   `json:"scopes"`
	ExpiresAt      *time.Time `json:"expires_at"`
}

func contextFromMembership(in membershipWire) *CustomerContext {
	return NormalizeContext(&CustomerContext{TenantUUID: in.TenantUUID, CustomerUUID: in.CustomerUUID, MembershipUUID: in.MembershipUUID, Roles: in.Roles, Scopes: in.Scopes, Source: CustomerAuthSourceDelegate, Authenticated: true, TokenExpiresAt: in.ExpiresAt})
}
func (c *DelegatedCoreAuthClient) call(ctx context.Context, method, path string, input, out any, customer bool) error {
	if c == nil || c.client == nil || c.tokens == nil {
		return NewError(CodeCustomerDelegateUnavailable, "delegated customer auth client unavailable")
	}
	token, err := c.tokens.Token(ctx)
	if err != nil || strings.TrimSpace(token) == "" {
		return WrapError(CodeCustomerDelegateUnavailable, "plugin service credential unavailable", err)
	}
	var body io.Reader
	if input != nil {
		raw, e := json.Marshal(input)
		if e != nil {
			return WrapError(CodeCustomerTokenInvalid, "customer auth request invalid", e)
		}
		body = bytes.NewReader(raw)
	}
	callCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	req, e := http.NewRequestWithContext(callCtx, method, c.baseURL+path, body)
	if e != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer auth request unavailable", e)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	req.Header.Set("Accept", "application/json")
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if customer {
		raw, ok := CustomerCredentialFromContext(ctx)
		if !ok {
			return NewError(CodeCustomerTokenMissing, "customer credential missing from request context")
		}
		req.Header.Set(customerDelegationHeader, "Bearer "+raw)
	}
	resp, e := c.client.Do(req)
	if e != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer auth request unavailable", e)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return WrapError(CodeCustomerDelegateUnavailable, "customer.auth.response_invalid", readErr)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return mapDelegatedMembershipError(resp.StatusCode, raw)
	}
	if err := hostcontract.DecodeData(raw, out); err != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer.auth.response_invalid", err)
	}
	return nil
}

var _ CustomerAuthClient = (*DelegatedCoreAuthClient)(nil)
