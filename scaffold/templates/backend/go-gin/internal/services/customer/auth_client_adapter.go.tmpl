package customer

import (
	"context"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
)

type FrameworkAuthClient struct {
	local     *LocalAuthService
	validator customerfw.CustomerTokenValidator
}

// NewLocalFrameworkAuthClient pins this adapter to plugin-local persistence.
// Runtime mode selection belongs to customerfw.Runtime, not this adapter.
func NewLocalFrameworkAuthClient(local *LocalAuthService, validator customerfw.CustomerTokenValidator) *FrameworkAuthClient {
	return &FrameworkAuthClient{local: local, validator: validator}
}

func (c *FrameworkAuthClient) Register(ctx context.Context, input customerfw.RegisterInput) (*customerfw.AuthResult, error) {
	if c.local == nil {
		return nil, ErrCustomerAuthNotImplemented
	}
	out, err := c.local.Register(ctx, RegisterInput{
		TenantUUID:  input.TenantUUID,
		Email:       input.Identifier,
		Password:    input.Password,
		DisplayName: input.Profile.DisplayName,
		Nickname:    input.Profile.Nickname,
		GivenName:   input.Profile.GivenName,
		FamilyName:  input.Profile.FamilyName,
		AvatarURL:   input.Profile.AvatarURL,
		Locale:      input.Profile.Locale,
		Timezone:    input.Profile.Timezone,
		Metadata:    input.Attributes,
	})
	if err != nil {
		return nil, err
	}
	return &customerfw.AuthResult{Context: &customerfw.CustomerContext{
		TenantUUID:    out.TenantUUID,
		CustomerUUID:  out.CustomerUUID,
		Profile:       customerfw.NormalizeAttributes(input.Profile),
		Source:        customerfw.CustomerAuthSourceLocal,
		Authenticated: true,
	}}, nil
}

func (c *FrameworkAuthClient) Login(ctx context.Context, input customerfw.LoginInput) (*customerfw.AuthResult, error) {
	if c.local == nil {
		return nil, ErrCustomerAuthNotImplemented
	}
	out, err := c.local.Login(ctx, LoginInput{
		TenantUUID: input.TenantUUID,
		Login:      input.Identifier,
		Password:   input.Password,
	})
	if err != nil {
		return nil, err
	}
	return &customerfw.AuthResult{
		AccessToken: out.Token,
		ExpiresIn:   out.ExpiresIn,
		Context: &customerfw.CustomerContext{
			TenantUUID:    input.TenantUUID,
			CustomerUUID:  out.CustomerUUID,
			Source:        customerfw.CustomerAuthSourceLocal,
			Authenticated: true,
		},
	}, nil
}

func (c *FrameworkAuthClient) Validate(ctx context.Context, token string) (*customerfw.CustomerContext, error) {
	if c.validator == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer token validator unavailable")
	}
	return c.validator.Validate(ctx, token, "")
}
