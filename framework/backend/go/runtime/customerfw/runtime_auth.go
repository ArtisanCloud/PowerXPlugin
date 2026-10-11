package customerfw

import "context"

// AuthClient resolves only the adapter selected by Runtime, including when the
// runtime or its selected adapter is missing. It never performs mode fallback.
func (r *Runtime) AuthClient() CustomerAuthClient { return runtimeAuthClient{r} }

type runtimeAuthClient struct{ runtime *Runtime }

func (c runtimeAuthClient) resolve() (CustomerAuthClient, error) {
	adapter, err := c.runtime.Auth()
	if err != nil {
		return nil, WrapError(CodeCustomerDelegateUnavailable, string(CodeCustomerDelegateUnavailable), err)
	}
	return adapter, nil
}

func (c runtimeAuthClient) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	adapter, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return adapter.Register(ctx, input)
}
func (c runtimeAuthClient) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	adapter, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return adapter.Login(ctx, input)
}
func (c runtimeAuthClient) Validate(ctx context.Context, token string) (*CustomerContext, error) {
	adapter, err := c.resolve()
	if err != nil {
		return nil, err
	}
	return adapter.Validate(ctx, token)
}
