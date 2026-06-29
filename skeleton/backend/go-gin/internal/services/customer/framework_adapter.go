package customer

import (
	"context"

	customerfw "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/runtime/customerfw"
	customerdomain "github.com/ArtisanCloud/PowerXPlugin/skeleton/backend/internal/domain/customer"
)

type FrameworkValidator struct {
	authenticator Authenticator
}

func NewFrameworkValidator(authenticator Authenticator) *FrameworkValidator {
	return &FrameworkValidator{authenticator: authenticator}
}

func (v *FrameworkValidator) Validate(ctx context.Context, token string, tenantUUID string) (*customerfw.CustomerContext, error) {
	if v == nil || v.authenticator == nil {
		return nil, customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer token validator unavailable")
	}
	cc, err := v.authenticator.Authenticate(ctx, tenantUUID, token)
	if err != nil {
		return nil, MapFrameworkError(err)
	}
	return ToFrameworkContext(cc), nil
}

func ToFrameworkContext(cc *customerdomain.CustomerContext) *customerfw.CustomerContext {
	if cc == nil {
		return nil
	}
	return &customerfw.CustomerContext{
		TenantUUID:   cc.TenantUUID,
		CustomerUUID: cc.CustomerUUID,
		Profile: customerfw.CustomerAttributes{
			DisplayName: cc.Profile.DisplayName,
			Nickname:    cc.Profile.Nickname,
			GivenName:   cc.Profile.GivenName,
			FamilyName:  cc.Profile.FamilyName,
			AvatarURL:   cc.Profile.AvatarURL,
			Locale:      cc.Profile.Locale,
			Timezone:    cc.Profile.Timezone,
		},
		Roles:         cc.Roles,
		Source:        toFrameworkSource(cc.SourceMode),
		Authenticated: cc.Authenticated,
		Attributes:    cc.Attributes,
		RawClaims:     cc.RawClaims,
	}
}

func FromFrameworkContext(cc *customerfw.CustomerContext) *customerdomain.CustomerContext {
	if cc == nil {
		return nil
	}
	return &customerdomain.CustomerContext{
		TenantUUID:   cc.TenantUUID,
		CustomerUUID: cc.CustomerUUID,
		Profile: customerdomain.CustomerProfile{
			DisplayName: cc.Profile.DisplayName,
			Nickname:    cc.Profile.Nickname,
			GivenName:   cc.Profile.GivenName,
			FamilyName:  cc.Profile.FamilyName,
			AvatarURL:   cc.Profile.AvatarURL,
			Locale:      cc.Profile.Locale,
			Timezone:    cc.Profile.Timezone,
		},
		Roles:         cc.Roles,
		SourceMode:    fromFrameworkSource(cc.Source),
		Attributes:    cc.Attributes,
		RawClaims:     cc.RawClaims,
		Authenticated: cc.Authenticated,
	}
}

func MapFrameworkError(err error) error {
	switch err {
	case nil:
		return nil
	case ErrCustomerAuthNotImplemented:
		return customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer auth not implemented")
	case ErrCustomerDelegateUnavailable:
		return customerfw.NewError(customerfw.CodeCustomerDelegateUnavailable, "customer auth delegate unavailable")
	case ErrCustomerTokenInvalid:
		return customerfw.NewError(customerfw.CodeCustomerTokenInvalid, "customer token invalid")
	default:
		return err
	}
}

func toFrameworkSource(mode customerdomain.CustomerAuthMode) customerfw.CustomerAuthSource {
	switch mode {
	case customerdomain.CustomerAuthModeLocal:
		return customerfw.CustomerAuthSourceLocal
	case customerdomain.CustomerAuthModeDelegate:
		return customerfw.CustomerAuthSourceDelegate
	default:
		return customerfw.CustomerAuthSourceDelegate
	}
}

func fromFrameworkSource(source customerfw.CustomerAuthSource) customerdomain.CustomerAuthMode {
	switch source {
	case customerfw.CustomerAuthSourceLocal, customerfw.CustomerAuthSourceLocalDev:
		return customerdomain.CustomerAuthModeLocal
	default:
		return customerdomain.CustomerAuthModeDelegate
	}
}
