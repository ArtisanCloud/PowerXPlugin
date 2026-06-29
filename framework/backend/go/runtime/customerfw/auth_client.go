package customerfw

import "context"

type RegisterInput struct {
	TenantUUID string         `json:"tenant_uuid,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	Identifier string         `json:"identifier,omitempty"`
	Password   string         `json:"password,omitempty"`
	Profile    map[string]any `json:"profile,omitempty"`
}

type LoginInput struct {
	TenantUUID string         `json:"tenant_uuid,omitempty"`
	Channel    string         `json:"channel,omitempty"`
	Identifier string         `json:"identifier,omitempty"`
	Password   string         `json:"password,omitempty"`
	Code       string         `json:"code,omitempty"`
	Nickname   string         `json:"nickname,omitempty"`
	AvatarURL  string         `json:"avatar_url,omitempty"`
	Profile    map[string]any `json:"profile,omitempty"`
}

type AuthResult struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	Context      *CustomerContext `json:"context"`
	ExpiresIn    int64            `json:"expires_in,omitempty"`
}

type CustomerAuthClient interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Validate(ctx context.Context, token string) (*CustomerContext, error)
}
