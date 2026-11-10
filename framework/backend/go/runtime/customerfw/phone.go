package customerfw

import (
	"context"
	"net/http"
	"strings"
	"time"
)

// 小程序手机号凭证类型：按钮 open-type=getPhoneNumber 返回的一次性 code。
// 由 Core 用客户端应用注册表中该 app_id 的凭证换出手机号；插件只拿到脱敏号码。
const (
	WeChatMiniAppPhoneCredentialType = "wechat_miniapp_phone_code"
	AlipayMiniAppPhoneCredentialType = "alipay_miniapp_phone_code"
	DouyinMiniAppPhoneCredentialType = "douyin_miniapp_phone_code"
)

var miniAppPhoneCredentialTypes = map[string]string{
	CustomerAuthChannelWeChatMiniApp: WeChatMiniAppPhoneCredentialType,
	CustomerAuthChannelAlipayMiniApp: AlipayMiniAppPhoneCredentialType,
	CustomerAuthChannelDouyinMiniApp: DouyinMiniAppPhoneCredentialType,
}

// VerifyPhoneInput 为当前客户认证手机号（客户身份来自请求上下文中的客户凭证）。
type VerifyPhoneInput struct {
	Channel    string             `json:"channel"`
	AppID      string             `json:"app_id"`
	Credential CustomerCredential `json:"credential"`
}

// PhoneVerification 是认证结果。
type PhoneVerification struct {
	PhoneMasked string    `json:"phone_masked"`
	VerifiedAt  time.Time `json:"verified_at"`
}

// CustomerPhoneVerifier 为当前已登录客户认证手机号。只在委派模式可用（需要 Core 持有的小程序凭证）。
type CustomerPhoneVerifier interface {
	VerifyPhone(ctx context.Context, in VerifyPhoneInput) (*PhoneVerification, error)
}

// VerifyPhone 调用 Core /tenant/customer/phone:verify，同时携带插件 STS 与当前客户凭证。
func (c *DelegatedCoreAuthClient) VerifyPhone(ctx context.Context, in VerifyPhoneInput) (*PhoneVerification, error) {
	want, ok := miniAppPhoneCredentialTypes[strings.TrimSpace(in.Channel)]
	if !ok || strings.TrimSpace(in.Credential.Type) != want || strings.TrimSpace(in.Credential.Value) == "" {
		return nil, NewError(CodeCustomerInvalidArgument, "unsupported phone credential")
	}
	if strings.TrimSpace(in.AppID) == "" {
		return nil, NewError(CodeCustomerInvalidArgument, "phone verification requires app_id")
	}
	if _, ok := CustomerCredentialFromContext(ctx); !ok {
		return nil, NewError(CodeCustomerTokenMissing, "customer credential missing from request context")
	}
	body := map[string]any{"channel": strings.TrimSpace(in.Channel), "app_id": strings.TrimSpace(in.AppID), "credential": in.Credential}
	var out PhoneVerification
	if err := c.call(ctx, http.MethodPost, "/api/v1/tenant/customer/phone:verify", body, &out, true); err != nil {
		return nil, err
	}
	if strings.TrimSpace(out.PhoneMasked) == "" {
		return nil, NewError(CodeCustomerDelegateUnavailable, "phone verification response is incomplete")
	}
	return &out, nil
}

var _ CustomerPhoneVerifier = (*DelegatedCoreAuthClient)(nil)
