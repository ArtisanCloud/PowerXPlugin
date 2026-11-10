package customerfw

import "strings"

// 小程序平台（委派登录 channel）与对应的一次性 code 凭证类型。
// 凭证由 Core 按客户端应用注册表（customer_auth.client_apps）校验：插件转发 code 与 app_id，
// 不持有 AppSecret，也不能自称 openid。支付宝 / 抖音为预留平台，Core 暂返回 CUSTOMER_CLIENT_PLATFORM_UNSUPPORTED。
const (
	CustomerAuthChannelAlipayMiniApp = "alipay_miniapp"
	CustomerAuthChannelDouyinMiniApp = "douyin_miniapp"

	AlipayMiniAppCredentialType = "alipay_miniapp_code"
	DouyinMiniAppCredentialType = "douyin_miniapp_code"
)

var miniAppCredentialTypes = map[string]string{
	CustomerAuthChannelWeChatMiniApp: WeChatMiniAppCredentialType,
	CustomerAuthChannelAlipayMiniApp: AlipayMiniAppCredentialType,
	CustomerAuthChannelDouyinMiniApp: DouyinMiniAppCredentialType,
}

// IsMiniAppChannel 判断渠道是否为小程序平台（登录需要 app_id）。
func IsMiniAppChannel(channel string) bool {
	_, ok := miniAppCredentialTypes[strings.TrimSpace(channel)]
	return ok
}

// MiniAppCredentialType 返回小程序平台对应的 code 凭证类型。
func MiniAppCredentialType(channel string) (string, bool) {
	t, ok := miniAppCredentialTypes[strings.TrimSpace(channel)]
	return t, ok
}
