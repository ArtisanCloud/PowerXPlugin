package federated

import (
	"time"

	federatedContracts "github.com/ArtisanCloud/PowerXPlugin/framework/backend/go/iam/federated/contracts"
	"github.com/google/uuid"
)

// LoginResult 统一扫码登录输出。
type LoginResult struct {
	TokenType    string          `json:"token_type"`
	AccessToken  string          `json:"access_token"`
	RefreshToken string          `json:"refresh_token"`
	ExpiresIn    int64           `json:"expires_in"`
	IssuedAt     time.Time       `json:"issued_at"`
	Context      IdentityContext `json:"context"`
}

// IdentityContext 统一身份上下文（standalone/delegated 对齐字段形态）。
type IdentityContext struct {
	TenantUUID   string   `json:"tenant_uuid"`
	Provider     string   `json:"provider"`
	ExternalID   string   `json:"external_id"`
	Roles        []string `json:"roles"`
	Permissions  []string `json:"permissions"`
	PolicySource string   `json:"policy_source"`
}

// LoginService 负责将外部身份映射为统一登录态输出。
type LoginService struct{}

func NewLoginService() *LoginService {
	return &LoginService{}
}

func (s *LoginService) Build(identity federatedContracts.ExternalIdentity, tenantUUID string) LoginResult {
	now := time.Now()
	access := "fed-" + uuid.NewString()
	refresh := "refresh-" + uuid.NewString()
	return LoginResult{
		TokenType:    "Bearer",
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresIn:    3600,
		IssuedAt:     now,
		Context: IdentityContext{
			TenantUUID:   tenantUUID,
			Provider:     identity.Provider,
			ExternalID:   identity.ExternalUserID,
			Roles:        []string{"plugins.base.user"},
			Permissions:  []string{"plugins.base.read"},
			PolicySource: "federated-default",
		},
	}
}
