package customerfw

import "strings"

type SourcePolicy struct {
	Mode        CustomerAuthSource
	Production  bool
	BreakGlass  bool
	Description string
}

func ValidateSourcePolicy(policy SourcePolicy) error {
	mode := NormalizeSource(policy.Mode)
	switch mode {
	case CustomerAuthSourcePlatform, CustomerAuthSourceDelegate, CustomerAuthSourceThirdParty, CustomerAuthSourceCore, CustomerAuthSourceWeChat:
		return nil
	case CustomerAuthSourceLocalDev, CustomerAuthSourceMock, CustomerAuthSourceLocal:
		if policy.Production && !policy.BreakGlass {
			return NewError(CodeCustomerIdentitySourceBlocked, "customer identity source forbidden in production")
		}
		return nil
	default:
		if strings.TrimSpace(string(mode)) == "" {
			return NewError(CodeCustomerIdentitySourceBlocked, "customer identity source required")
		}
		if policy.Production && !policy.BreakGlass {
			return NewError(CodeCustomerIdentitySourceBlocked, "customer identity source forbidden in production")
		}
		return nil
	}
}
