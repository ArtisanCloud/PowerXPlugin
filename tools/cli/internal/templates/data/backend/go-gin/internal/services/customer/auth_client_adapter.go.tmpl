package customer

import (
	"context"
	"strings"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	"github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/config"
)

type FrameworkAuthClient struct {
	cfg       *config.Config
	local     *LocalAuthService
	delegated customerfw.CustomerAuthClient
	validator customerfw.CustomerTokenValidator
}

func NewFrameworkAuthClient(cfg *config.Config, local *LocalAuthService, validator customerfw.CustomerTokenValidator) *FrameworkAuthClient {
	var delegated customerfw.CustomerAuthClient
	if cfg != nil && cfg.CustomerAuth != nil && strings.TrimSpace(cfg.CustomerAuth.DelegateEndpoint) != "" {
		delegated = customerfw.NewDelegatedCustomerAuthClient(customerfw.DelegatedClientConfig{
			BaseURL: strings.TrimRight(strings.TrimSpace(cfg.CustomerAuth.DelegateEndpoint), "/"),
		})
	}
	return &FrameworkAuthClient{cfg: cfg, local: local, delegated: delegated, validator: validator}
}

func (c *FrameworkAuthClient) Register(ctx context.Context, input customerfw.RegisterInput) (*customerfw.AuthResult, error) {
	if c.useDelegate() {
		if c.delegated == nil {
			return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer auth delegate unavailable")
		}
		return c.delegated.Register(ctx, input)
	}
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
	if c.useDelegate() {
		if c.delegated == nil {
			return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer auth delegate unavailable")
		}
		return c.delegated.Login(ctx, input)
	}
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
	if c.useDelegate() && c.delegated != nil {
		return c.delegated.Validate(ctx, token)
	}
	if c.validator == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer token validator unavailable")
	}
	return c.validator.Validate(ctx, token, "")
}

func (c *FrameworkAuthClient) useDelegate() bool {
	return c != nil && c.cfg != nil && c.cfg.CustomerAuth != nil && strings.TrimSpace(strings.ToLower(c.cfg.CustomerAuth.Mode)) == "delegate"
}
