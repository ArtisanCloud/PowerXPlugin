package knowledge

import "strings"

type SourcePolicy struct {
	Mode        string
	Production  bool
	BreakGlass  bool
	Description string
}

func ValidateSourcePolicy(policy SourcePolicy) error {
	mode := strings.ToLower(strings.TrimSpace(policy.Mode))
	switch mode {
	case ProviderModeDelegated, ProviderModeThirdParty:
		return nil
	case ProviderModeLocal, ProviderModeMock:
		if policy.Production && !policy.BreakGlass {
			return NewError(CodeForbidden, "knowledge source forbidden in production")
		}
		return nil
	default:
		if mode == "" {
			return NewError(CodeForbidden, "knowledge source mode is required")
		}
		if policy.Production && !policy.BreakGlass {
			return NewError(CodeForbidden, "knowledge source forbidden in production")
		}
		return nil
	}
}
