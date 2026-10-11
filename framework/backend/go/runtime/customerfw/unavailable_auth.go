package customerfw

import "context"

// UnavailableDelegatedCustomerAuthClient is the only permitted delegated auth
// adapter until Core publishes a formal Customer Auth Host Contract. It makes
// the missing contract explicit and never falls back to plugin-local tables.
type UnavailableDelegatedCustomerAuthClient struct{}

func NewUnavailableDelegatedCustomerAuthClient() *UnavailableDelegatedCustomerAuthClient {
	return &UnavailableDelegatedCustomerAuthClient{}
}

func (*UnavailableDelegatedCustomerAuthClient) Register(context.Context, RegisterInput) (*AuthResult, error) {
	return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer auth Host Contract is unavailable")
}

func (*UnavailableDelegatedCustomerAuthClient) Login(context.Context, LoginInput) (*AuthResult, error) {
	return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer auth Host Contract is unavailable")
}

func (*UnavailableDelegatedCustomerAuthClient) Validate(context.Context, string) (*CustomerContext, error) {
	return nil, NewError(CodeCustomerDelegateUnavailable, "delegated customer auth Host Contract is unavailable")
}

var _ CustomerAuthClient = (*UnavailableDelegatedCustomerAuthClient)(nil)
