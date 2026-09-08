package customerfw

import "context"

type RegisterInput struct {
	TenantUUID string             `json:"tenant_uuid,omitempty"`
	Channel    string             `json:"channel,omitempty"`
	Identifier string             `json:"identifier,omitempty"`
	Password   string             `json:"password,omitempty"`
	Profile    CustomerAttributes `json:"profile,omitempty"`
	Attributes map[string]any     `json:"attributes,omitempty"`
	Credential CustomerCredential `json:"credential,omitempty"`
}

type LoginInput struct {
	TenantUUID string             `json:"tenant_uuid,omitempty"`
	Channel    string             `json:"channel,omitempty"`
	Identifier string             `json:"identifier,omitempty"`
	Password   string             `json:"password,omitempty"`
	Code       string             `json:"code,omitempty"`
	Nickname   string             `json:"nickname,omitempty"`
	AvatarURL  string             `json:"avatar_url,omitempty"`
	Profile    CustomerAttributes `json:"profile,omitempty"`
	Attributes map[string]any     `json:"attributes,omitempty"`
	Credential CustomerCredential `json:"credential,omitempty"`
}

// CustomerCredential is verified by Core for delegated authentication. Its
// value is opaque to Framework and must never be logged or persisted by a
// plugin. Current Core contract supports shopify_customer_access_token.
type CustomerCredential struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
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
