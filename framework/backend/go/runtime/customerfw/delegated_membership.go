package customerfw

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/powerx/hostcontract"
)

const (
	CapabilityCustomerMembershipsDelegatedRead = "com.corex.customer.memberships.delegated_read"
	customerDelegationHeader                   = "X-PowerX-Customer-Authorization"
)

// ServiceTokenProvider obtains the short-lived plugin STS credential used as
// Authorization for a Core Host Contract call.
type ServiceTokenProvider interface {
	Token(context.Context) (string, error)
}

type ServiceTokenProviderFunc func(context.Context) (string, error)

func (f ServiceTokenProviderFunc) Token(ctx context.Context) (string, error) { return f(ctx) }

type DelegatedMembershipResolverConfig struct {
	BaseURL       string
	ServiceTokens ServiceTokenProvider
	Timeout       time.Duration
	Client        *http.Client
}

// DelegatedMembershipResolver calls Core's customer-self membership contract.
// It never serializes customer_uuid, tenant_uuid or membership_uuid: Core
// derives those values from the two independently verified credentials.
type DelegatedMembershipResolver struct {
	baseURL       string
	serviceTokens ServiceTokenProvider
	timeout       time.Duration
	client        *http.Client
}

func NewDelegatedMembershipResolver(cfg DelegatedMembershipResolverConfig) (*DelegatedMembershipResolver, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("customer delegated membership: base URL is required")
	}
	if cfg.ServiceTokens == nil {
		return nil, errors.New("customer delegated membership: service token provider is required")
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: timeout + 500*time.Millisecond}
	}
	return &DelegatedMembershipResolver{baseURL: strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/"), serviceTokens: cfg.ServiceTokens, timeout: timeout, client: client}, nil
}

func (r *DelegatedMembershipResolver) Resolve(ctx context.Context, customerUUID, tenantUUID string) (*CustomerMembership, error) {
	var out struct {
		Item CustomerMembership `json:"item"`
	}
	if err := r.call(ctx, http.MethodPost, "/api/v1/tenant/customer/memberships:resolve", &out); err != nil {
		return nil, err
	}
	item := normalizeMembership(&out.Item)
	if item == nil || item.CustomerUUID == "" || item.TenantUUID == "" || item.MembershipUUID == "" {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership response is incomplete")
	}
	// These are assertions made by the Framework caller, never Core query
	// parameters. Reject disagreement instead of accepting a membership for a
	// different authenticated subject.
	if expected := normalizeID(customerUUID); expected != "" && expected != item.CustomerUUID {
		return nil, NewError(CodeCustomerTenantMismatch, "delegated customer membership does not match customer context")
	}
	if expected := normalizeID(tenantUUID); expected != "" && expected != item.TenantUUID {
		return nil, NewError(CodeCustomerTenantMismatch, "delegated customer membership does not match tenant context")
	}
	return item, nil
}

func (r *DelegatedMembershipResolver) List(ctx context.Context, customerUUID string) ([]CustomerMembership, error) {
	var out struct {
		Item CustomerMembership `json:"item"`
	}
	if err := r.call(ctx, http.MethodGet, "/api/v1/tenant/customer/memberships", &out); err != nil {
		return nil, err
	}
	item := normalizeMembership(&out.Item)
	if item == nil || item.CustomerUUID == "" || item.TenantUUID == "" || item.MembershipUUID == "" {
		return nil, NewError(CodeCustomerDelegateUnavailable, "customer membership response is incomplete")
	}
	if expected := normalizeID(customerUUID); expected != "" && expected != item.CustomerUUID {
		return nil, NewError(CodeCustomerTenantMismatch, "delegated customer membership does not match customer context")
	}
	return []CustomerMembership{*item}, nil
}

func (r *DelegatedMembershipResolver) call(ctx context.Context, method, path string, out any) error {
	if r == nil || r.client == nil || r.serviceTokens == nil || r.baseURL == "" {
		return NewError(CodeCustomerDelegateUnavailable, "delegated customer membership resolver unavailable")
	}
	customerToken, ok := CustomerCredentialFromContext(ctx)
	if !ok {
		return NewError(CodeCustomerTokenMissing, "customer credential missing from request context")
	}
	serviceToken, err := r.serviceTokens.Token(ctx)
	if err != nil || strings.TrimSpace(serviceToken) == "" {
		return WrapError(CodeCustomerDelegateUnavailable, "plugin service credential unavailable", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestCtx, method, r.baseURL+path, nil)
	if err != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer membership request unavailable", err)
	}
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(serviceToken))
	req.Header.Set(customerDelegationHeader, "Bearer "+customerToken)
	resp, err := r.client.Do(req)
	if err != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer membership request unavailable", err)
	}
	defer resp.Body.Close()
	raw, readErr := io.ReadAll(io.LimitReader(resp.Body, (1<<20)+1))
	if readErr != nil || len(raw) > 1<<20 {
		return WrapError(CodeCustomerDelegateUnavailable, "customer.membership.response_invalid", readErr)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return mapDelegatedMembershipError(resp.StatusCode, raw)
	}
	if err := hostcontract.DecodeData(raw, out); err != nil {
		return WrapError(CodeCustomerDelegateUnavailable, "customer membership response invalid", err)
	}
	return nil
}

func mapDelegatedMembershipError(status int, raw []byte) (mapped error) {
	reason := hostcontract.ParseReasonCode(raw, "")
	defer func() {
		var typed *Error
		if errors.As(mapped, &typed) && typed != nil {
			typed.StatusCode = status
			typed.ReasonCode = reason
		}
	}()
	switch reason {
	case "CUSTOMER_INVALID_ARGUMENT":
		return NewError(CodeCustomerInvalidArgument, "customer request rejected by Core")
	case "CUSTOMER_UNAUTHORIZED":
		return NewError(CodeCustomerTokenInvalid, "customer credential rejected by Core")
	case "CUSTOMER_CREDENTIAL_INVALID":
		return NewError(CodeCustomerCredentialInvalid, "customer credential rejected by Core")
	case "CUSTOMER_FORBIDDEN":
		return NewError(CodeCustomerForbidden, "customer membership capability forbidden")
	case "CUSTOMER_IDENTITY_NOT_FOUND":
		return NewError(CodeCustomerIdentityNotFound, "customer identity not found")
	case "CUSTOMER_MEMBERSHIP_NOT_FOUND":
		return NewError(CodeCustomerMembershipRequired, "customer membership not found")
	case "CUSTOMER_MEMBERSHIP_INACTIVE":
		return NewError(CodeCustomerMembershipDisabled, "customer membership inactive")
	case "CUSTOMER_UPSTREAM_DEPENDENCY":
		return NewError(CodeCustomerDelegateUnavailable, "customer membership dependency unavailable")
	}
	if status == http.StatusUnauthorized {
		return NewError(CodeCustomerTokenInvalid, "customer credential rejected by Core")
	}
	if status == http.StatusForbidden {
		return NewError(CodeCustomerForbidden, "customer membership capability forbidden")
	}
	if status == http.StatusNotFound {
		return NewError(CodeCustomerMembershipRequired, "customer membership not found")
	}
	return NewError(CodeCustomerDelegateUnavailable, "customer membership dependency unavailable")
}

func normalizeMembership(item *CustomerMembership) *CustomerMembership {
	if item == nil {
		return nil
	}
	copy := *item
	copy.TenantUUID = normalizeID(copy.TenantUUID)
	copy.CustomerUUID = normalizeID(copy.CustomerUUID)
	copy.MembershipUUID = normalizeID(copy.MembershipUUID)
	copy.Roles = compactStrings(copy.Roles)
	copy.Scopes = compactStrings(copy.Scopes)
	return &copy
}

// UnavailableDelegatedMembershipResolver remains useful for explicit test and
// bootstrap failure paths. Production delegated assembly must use
// NewDelegatedMembershipResolver once Core's Host Contract is configured.
type UnavailableDelegatedMembershipResolver struct{}

func NewUnavailableDelegatedMembershipResolver() *UnavailableDelegatedMembershipResolver {
	return &UnavailableDelegatedMembershipResolver{}
}

func (r *UnavailableDelegatedMembershipResolver) Resolve(context.Context, string, string) (*CustomerMembership, error) {
	return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer membership resolver unavailable")
}

func (r *UnavailableDelegatedMembershipResolver) List(context.Context, string) ([]CustomerMembership, error) {
	return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer membership resolver unavailable")
}
