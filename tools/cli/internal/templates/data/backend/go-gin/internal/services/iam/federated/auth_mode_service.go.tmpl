package federated

import "strings"

const (
	AuthModeCoexist       = "coexist"
	AuthModeFederatedOnly = "federated_only"
	AuthModePasswordOnly  = "password_only"
)

// AuthModeService 管理密码/扫码并存模式。
type AuthModeService struct {
	mode string
}

func NewAuthModeService(mode string) *AuthModeService {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch normalized {
	case AuthModeFederatedOnly, AuthModePasswordOnly:
		return &AuthModeService{mode: normalized}
	default:
		return &AuthModeService{mode: AuthModeCoexist}
	}
}

func (s *AuthModeService) Mode() string {
	if s == nil {
		return AuthModeCoexist
	}
	return s.mode
}

func (s *AuthModeService) FederatedEnabled() bool {
	mode := s.Mode()
	return mode == AuthModeCoexist || mode == AuthModeFederatedOnly
}

func (s *AuthModeService) PasswordEnabled() bool {
	mode := s.Mode()
	return mode == AuthModeCoexist || mode == AuthModePasswordOnly
}
