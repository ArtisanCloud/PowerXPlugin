package customerfw

import (
	"context"
	"errors"
	"fmt"
	"strings"

	miniProgram "github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
)

const CustomerAuthChannelWeChatMiniApp = "wechat_miniapp"

type WeChatMiniAppConfig struct {
	AppID     string
	AppSecret string
}

type WeChatMiniAppSession struct {
	OpenID     string `json:"openid"`
	SessionKey string `json:"session_key"`
	UnionID    string `json:"unionid,omitempty"`
}

type WeChatMiniAppSessionExchanger interface {
	ExchangeCode(ctx context.Context, code string) (*WeChatMiniAppSession, error)
}

type PowerWeChatMiniAppExchanger struct {
	app *miniProgram.MiniProgram
}

func NewPowerWeChatMiniAppExchanger(cfg WeChatMiniAppConfig) (*PowerWeChatMiniAppExchanger, error) {
	appID := strings.TrimSpace(cfg.AppID)
	appSecret := strings.TrimSpace(cfg.AppSecret)
	if appID == "" || appSecret == "" {
		return nil, errors.New("wechat miniapp app_id/app_secret is required")
	}
	app, err := miniProgram.NewMiniProgram(&miniProgram.UserConfig{
		AppID:  appID,
		Secret: appSecret,
	})
	if err != nil {
		return nil, err
	}
	return &PowerWeChatMiniAppExchanger{app: app}, nil
}

func NewPowerWeChatMiniAppExchangerWithApp(app *miniProgram.MiniProgram) *PowerWeChatMiniAppExchanger {
	return &PowerWeChatMiniAppExchanger{app: app}
}

func (e *PowerWeChatMiniAppExchanger) ExchangeCode(ctx context.Context, code string) (*WeChatMiniAppSession, error) {
	if e == nil || e.app == nil || e.app.Auth == nil {
		return nil, errors.New("wechat miniapp exchanger is not initialized")
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("wechat miniapp code is required")
	}
	resp, err := e.app.Auth.Session(ctx, code)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("wechat miniapp code2session returned empty response")
	}
	if resp.ErrCode != 0 {
		return nil, fmt.Errorf("wechat miniapp code2session failed: %d %s", resp.ErrCode, resp.ErrMsg)
	}
	if strings.TrimSpace(resp.OpenID) == "" {
		return nil, errors.New("wechat miniapp code2session returned empty openid")
	}
	return &WeChatMiniAppSession{
		OpenID:     strings.TrimSpace(resp.OpenID),
		SessionKey: strings.TrimSpace(resp.SessionKey),
		UnionID:    strings.TrimSpace(resp.UnionID),
	}, nil
}

func IsWeChatMiniAppLogin(input LoginInput) bool {
	return strings.EqualFold(strings.TrimSpace(input.Channel), CustomerAuthChannelWeChatMiniApp) ||
		strings.EqualFold(strings.TrimSpace(input.Channel), string(CustomerAuthSourceWeChat))
}
