package customer

import (
	"context"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
)

// LocalFrameworkCustomerStore is the one local-mode implementation injected
// into CustomerRuntime. It keeps plugin persistence behind Framework contracts
// and prevents handlers from selecting local/delegated paths themselves.
type LocalFrameworkCustomerStore struct {
	auth       customerfw.CustomerAuthClient
	external   customerfw.ExternalIdentityService
	membership customerfw.CustomerMembershipResolver
}

func NewLocalFrameworkCustomerStore(auth customerfw.CustomerAuthClient, external customerfw.ExternalIdentityService, membership customerfw.CustomerMembershipResolver) *LocalFrameworkCustomerStore {
	return &LocalFrameworkCustomerStore{auth: auth, external: external, membership: membership}
}

func (s *LocalFrameworkCustomerStore) Register(ctx context.Context, input customerfw.RegisterInput) (*customerfw.AuthResult, error) {
	if s == nil || s.auth == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer auth adapter unavailable")
	}
	return s.auth.Register(ctx, input)
}

func (s *LocalFrameworkCustomerStore) Login(ctx context.Context, input customerfw.LoginInput) (*customerfw.AuthResult, error) {
	if s == nil || s.auth == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer auth adapter unavailable")
	}
	return s.auth.Login(ctx, input)
}

func (s *LocalFrameworkCustomerStore) Validate(ctx context.Context, token string) (*customerfw.CustomerContext, error) {
	if s == nil || s.auth == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer auth adapter unavailable")
	}
	return s.auth.Validate(ctx, token)
}

func (s *LocalFrameworkCustomerStore) ResolveExternalIdentity(ctx context.Context, input customerfw.ResolveExternalIdentityRequest) (*customerfw.ExternalIdentityResolution, error) {
	if s == nil || s.external == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer external identity adapter unavailable")
	}
	return s.external.ResolveExternalIdentity(ctx, input)
}

func (s *LocalFrameworkCustomerStore) Resolve(ctx context.Context, customerUUID, tenantUUID string) (*customerfw.CustomerMembership, error) {
	if s == nil || s.membership == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer membership adapter unavailable")
	}
	return s.membership.Resolve(ctx, customerUUID, tenantUUID)
}

func (s *LocalFrameworkCustomerStore) List(ctx context.Context, customerUUID string) ([]customerfw.CustomerMembership, error) {
	if s == nil || s.membership == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "local customer membership adapter unavailable")
	}
	return s.membership.List(ctx, customerUUID)
}
