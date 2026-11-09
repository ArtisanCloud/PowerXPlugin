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
	// AppID 是小程序等客户端应用的 ID；渠道为小程序平台时必填（Core 据此查找凭证并校验插件授权）。
	AppID string `json:"app_id,omitempty"`
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
	// AppID 是小程序等客户端应用的 ID；渠道为小程序平台时必填（Core 据此查找凭证并校验插件授权）。
	AppID string `json:"app_id,omitempty"`
}

// CustomerCredential is verified by Core for delegated authentication. Its
// value is opaque to Framework and must never be logged or persisted by a
// plugin. Current Core contract supports shopify_customer_access_token
// (channel shopify_storefront) and wechat_miniapp_code (channel wechat_miniapp,
// together with AppID); alipay_miniapp_code / douyin_miniapp_code are reserved.
type CustomerCredential struct {
	Type  string `json:"type,omitempty"`
	Value string `json:"value,omitempty"`
}

type AuthResult struct {
	AccessToken  string           `json:"access_token"`
	RefreshToken string           `json:"refresh_token,omitempty"`
	Context      *CustomerContext `json:"context"`
	ExpiresIn    int64            `json:"expires_in,omitempty"`
	// PhoneVerified 表示客户是否已完成 Core 手机号认证（委派模式由 Core 返回），插件据此引导认证。
	PhoneVerified bool `json:"phone_verified"`
}

type CustomerAuthClient interface {
	Register(ctx context.Context, input RegisterInput) (*AuthResult, error)
	Login(ctx context.Context, input LoginInput) (*AuthResult, error)
	Validate(ctx context.Context, token string) (*CustomerContext, error)
}
